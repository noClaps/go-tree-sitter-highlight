package highlight

import (
	"context"
	"iter"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// CaptureIndex represents the index of a capture name.
type CaptureIndex uint

const defaultHighlight = CaptureIndex(^uint(0))

// event is an interface that represents a highlight event.
// Possible implementations are:
// - [eventLayerStart]
// - [eventLayerEnd]
// - [eventCaptureStart]
// - [eventCaptureEnd]
// - [eventSource]
type event interface {
	highlightEvent()
}

// eventSource is emitted when a source code range is highlighted.
type eventSource struct {
	StartByte uint
	EndByte   uint
}

func (eventSource) highlightEvent() {}

// eventLayerStart is emitted when a language injection starts.
type eventLayerStart struct {
	// LanguageName is the name of the language that is being injected.
	LanguageName string
}

func (eventLayerStart) highlightEvent() {}

// eventLayerEnd is emitted when a language injection ends.
type eventLayerEnd struct{}

func (eventLayerEnd) highlightEvent() {}

// eventCaptureStart is emitted when a highlight region starts.
type eventCaptureStart struct {
	// Highlight is the capture name of the highlight.
	Highlight CaptureIndex
}

func (eventCaptureStart) highlightEvent() {}

// eventCaptureEnd is emitted when a highlight region ends.
type eventCaptureEnd struct{}

func (eventCaptureEnd) highlightEvent() {}

// InjectionCallback is called when a language injection is found to load the configuration for the injected language.
type InjectionCallback func(languageName string) *Configuration

// highlighter is a syntax highlighter that uses tree-sitter to parse source code and apply syntax highlighting. It is not thread-safe.
type highlighter struct {
	parser  *tree_sitter.Parser
	cursors []*tree_sitter.QueryCursor
}

func (h *highlighter) pushCursor(cursor *tree_sitter.QueryCursor) {
	h.cursors = append(h.cursors, cursor)
}

func (h *highlighter) popCursor() *tree_sitter.QueryCursor {
	if len(h.cursors) == 0 {
		return tree_sitter.NewQueryCursor()
	}

	cursor := h.cursors[len(h.cursors)-1]
	h.cursors = h.cursors[:len(h.cursors)-1]
	return cursor
}

// Highlight highlights the given source code using the given configuration. The source code is expected to be UTF-8 encoded.
// The function returns the highlighted HTML or an error.
func Highlight(cfg Configuration, source string, injectionCallback InjectionCallback, attributeCallback AttributeCallback) (string, error) {
	h := &highlighter{
		parser: tree_sitter.NewParser(),
	}
	layers, err := newIterLayers([]byte(source), "", h, injectionCallback, cfg, 0, []tree_sitter.Range{
		{
			StartByte:  0,
			EndByte:    ^uint(0),
			StartPoint: tree_sitter.NewPoint(0, 0),
			EndPoint:   tree_sitter.NewPoint(^uint(0), ^uint(0)),
		},
	})
	if err != nil {
		return "", err
	}

	i := &iterator{
		Ctx:                context.Background(),
		Source:             []byte(source),
		LanguageName:       cfg.languageName,
		ByteOffset:         0,
		Highlighter:        h,
		InjectionCallback:  injectionCallback,
		Layers:             layers,
		NextEvents:         nil,
		LastHighlightRange: nil,
	}
	i.sortLayers()

	var events iter.Seq2[event, error] = func(yield func(event, error) bool) {
		for {
			event, err := i.next()
			if err != nil {
				yield(nil, err)

				// error we are done
				return
			}

			if event == nil {
				// we're done if there are no more events
				return
			}

			// yield the event
			if !yield(event, nil) {
				// if the consumer returns false we can stop
				return
			}
		}
	}

	return render(events, source, attributeCallback)
}

// Compute the ranges that should be included when parsing an injection.
// This takes into account three things:
//   - `parent_ranges` - The ranges must all fall within the *current* layer's ranges.
//   - `nodes` - Every injection takes place within a set of nodes. The injection ranges are the
//     ranges of those nodes.
//   - `includes_children` - For some injections, the content nodes' children should be excluded
//     from the nested document, so that only the content nodes' *own* content is reparsed. For
//     other injections, the content nodes' entire ranges should be reparsed, including the ranges
//     of their children.
func intersectRanges(parentRanges []tree_sitter.Range, nodes []tree_sitter.Node, includesChildren bool) []tree_sitter.Range {
	cursor := nodes[0].Walk()
	defer cursor.Close()

	result := []tree_sitter.Range{}

	if len(parentRanges) == 0 {
		panic("Layers should only be constructed with non-empty ranges")
	}

	parentRange := parentRanges[0]
	parentRanges = parentRanges[1:]

	for _, node := range nodes {
		precedingRange := tree_sitter.Range{
			EndByte:  node.StartByte(),
			EndPoint: node.StartPosition(),
		}
		followingRange := tree_sitter.Range{
			StartByte:  node.EndByte(),
			StartPoint: node.EndPosition(),
			EndByte:    ^uint(0),
			EndPoint:   tree_sitter.NewPoint(^uint(0), ^uint(0)),
		}

		excludedRanges := []tree_sitter.Range{}
		for _, child := range node.Children(cursor) {
			if !includesChildren {
				excludedRanges = append(excludedRanges, child.Range())
			}
		}
		excludedRanges = append(excludedRanges, followingRange)

		for _, excludedRange := range excludedRanges {
			r := tree_sitter.Range{
				StartByte:  precedingRange.EndByte,
				StartPoint: precedingRange.EndPoint,
				EndByte:    excludedRange.StartByte,
				EndPoint:   excludedRange.StartPoint,
			}
			precedingRange = excludedRange

			if r.EndByte < parentRange.StartByte {
				continue
			}

			for parentRange.StartByte <= r.EndByte {
				if parentRange.EndByte > r.StartByte {
					if r.StartByte < parentRange.StartByte {
						r.StartByte = parentRange.StartByte
						r.StartPoint = parentRange.StartPoint
					}

					if parentRange.EndByte < r.EndByte {
						if r.StartByte < parentRange.EndByte {
							result = append(result, tree_sitter.Range{
								StartByte:  r.StartByte,
								StartPoint: r.StartPoint,
								EndByte:    parentRange.EndByte,
								EndPoint:   precedingRange.EndPoint,
							})
						}
						r.StartByte = parentRange.EndByte
						r.StartPoint = parentRange.EndPoint
					} else {
						if r.StartByte < r.EndByte {
							result = append(result, r)
						}
						break
					}
				}

				if len(parentRanges) > 0 {
					parentRange = parentRanges[0]
					parentRanges = parentRanges[1:]
				} else {
					return result
				}
			}
		}
	}

	return result
}

func injectionForMatch(config Configuration, parentName string, query *tree_sitter.Query, match tree_sitter.QueryMatch, source []byte) (string, *tree_sitter.Node, bool) {
	if config.injectionContentCaptureIndex == nil || config.injectionLanguageCaptureIndex == nil {
		return "", nil, false
	}

	var (
		languageName    string
		contentNode     *tree_sitter.Node
		includeChildren bool
	)

	for _, capture := range match.Captures {
		index := uint(capture.Index)
		switch index {
		case *config.injectionLanguageCaptureIndex:
			languageName = capture.Node.Utf8Text(source)
		case *config.injectionContentCaptureIndex:
			contentNode = &capture.Node
		}
	}

	for _, property := range query.PropertySettings(match.PatternIndex) {
		switch property.Key {
		case captureInjectionLanguage:
			if languageName == "" {
				languageName = *property.Value
			}
		case captureInjectionSelf:
			if languageName == "" {
				languageName = config.languageName
			}
		case captureInjectionParent:
			if languageName == "" {
				languageName = parentName
			}
		case captureInjectionIncludeChildren:
			includeChildren = true
		}
	}

	return languageName, contentNode, includeChildren
}
