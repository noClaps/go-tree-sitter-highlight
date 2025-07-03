package highlight

import (
	"github.com/noclaps/go-tree-sitter-highlight/types"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

const DefaultHighlight = types.CaptureIndex(^uint(0))

// Highlighter is a syntax Highlighter that uses tree-sitter to parse source code and apply syntax highlighting. It is not thread-safe.
type Highlighter struct {
	Parser  *tree_sitter.Parser
	cursors []*tree_sitter.QueryCursor
}

func (h *Highlighter) PushCursor(cursor *tree_sitter.QueryCursor) {
	h.cursors = append(h.cursors, cursor)
}

func (h *Highlighter) PopCursor() *tree_sitter.QueryCursor {
	if len(h.cursors) == 0 {
		return tree_sitter.NewQueryCursor()
	}

	cursor := h.cursors[len(h.cursors)-1]
	h.cursors = h.cursors[:len(h.cursors)-1]
	return cursor
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
func IntersectRanges(parentRanges []tree_sitter.Range, nodes []tree_sitter.Node, includesChildren bool) []tree_sitter.Range {
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

func InjectionForMatch(config types.Configuration, parentName string, query *tree_sitter.Query, match tree_sitter.QueryMatch, source []byte) (string, *tree_sitter.Node, bool) {
	if config.InjectionContentCaptureIndex == nil || config.InjectionLanguageCaptureIndex == nil {
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
		case *config.InjectionLanguageCaptureIndex:
			languageName = capture.Node.Utf8Text(source)
		case *config.InjectionContentCaptureIndex:
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
				languageName = config.LanguageName
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
