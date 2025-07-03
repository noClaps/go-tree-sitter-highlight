package highlight

import (
	"fmt"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type highlightQueueItem struct {
	config Configuration
	depth  uint
	ranges []tree_sitter.Range
}

type injectionItem struct {
	languageName    string
	nodes           []tree_sitter.Node
	includeChildren bool
}

type sortKey struct {
	offset uint
	start  bool
	depth  int
}

// compare compares the current sortKey (k) with another sortKey (other) lexicographically.
// Returns:
//
// -1 if other is greater
//
//	1 if k is greater
//
// 0 if both are equal
func (k sortKey) compare(other sortKey) int {
	if k.offset < other.offset {
		return -1
	}
	if k.offset > other.offset {
		return 1
	}

	if !k.start && other.start {
		return -1
	}
	if k.start && !other.start {
		return 1
	}

	if k.depth < other.depth {
		return -1
	}
	if k.depth > other.depth {
		return 1
	}

	return 0
}

func (k sortKey) greaterThan(other sortKey) bool {
	return k.compare(other) == 1
}

func (k sortKey) lessThan(other sortKey) bool {
	return k.compare(other) == -1
}

type _queryCapture struct {
	Match tree_sitter.QueryMatch
	Index uint
}

type localDef struct {
	Name      string
	Range     tree_sitter.Range
	Highlight *Highlight
}

type localScope struct {
	Inherits  bool
	Range     tree_sitter.Range
	LocalDefs []localDef
}

func newIterLayers(
	source []byte,
	parentName string,
	highlighter *Highlighter,
	injectionCallback injectionCallback,
	config Configuration,
	depth uint,
	ranges []tree_sitter.Range,
) ([]*iterLayer, error) {
	var result []*iterLayer
	var queue []highlightQueueItem
	for {
		if err := highlighter.Parser.SetIncludedRanges(ranges); err == nil {
			if err = highlighter.Parser.SetLanguage(config.Language); err != nil {
				return nil, fmt.Errorf("error setting language: %w", err)
			}
			tree := highlighter.Parser.ParseWithOptions(func(i int, p tree_sitter.Point) []byte {
				return source[i:]
			}, nil, nil)

			cursor := highlighter.popCursor()

			// Process combined injections.
			if config.CombinedInjectionsQuery != nil {
				injectionsByPatternIndex := make([]injectionItem, config.CombinedInjectionsQuery.PatternCount())

				matches := cursor.Matches(config.CombinedInjectionsQuery, tree.RootNode(), source)
				for {
					match := matches.Next()
					if match == nil {
						break
					}

					languageName, contentNode, includeChildren := injectionForMatch(config, parentName, config.CombinedInjectionsQuery, *match, source)

					if languageName == "" {
						injectionsByPatternIndex[match.PatternIndex].languageName = languageName
					}
					if contentNode != nil {
						injectionsByPatternIndex[match.PatternIndex].nodes = append(injectionsByPatternIndex[match.PatternIndex].nodes, *contentNode)
					}
					injectionsByPatternIndex[match.PatternIndex].includeChildren = includeChildren
				}

				for _, injection := range injectionsByPatternIndex {
					if injection.languageName != "" && len(injection.nodes) > 0 {
						nextConfig := injectionCallback(injection.languageName)
						if nextConfig != nil {
							nextRanges := intersectRanges(ranges, injection.nodes, injection.includeChildren)
							if len(nextRanges) > 0 {
								queue = append(queue, highlightQueueItem{
									config: *nextConfig,
									depth:  depth + 1,
									ranges: nextRanges,
								})
							}
						}
					}
				}
			}

			queryCaptures := newQueryCapturesIter(cursor.Captures(config.Query, tree.RootNode(), source))
			if _, _, ok := queryCaptures.peek(); !ok {
				continue
			}

			result = append(result, &iterLayer{
				Tree:              tree,
				Cursor:            cursor,
				Config:            config,
				HighlightEndStack: nil,
				ScopeStack: []localScope{
					{
						Inherits: false,
						Range: tree_sitter.Range{
							StartByte: 0,
							StartPoint: tree_sitter.Point{
								Row:    0,
								Column: 0,
							},
							EndByte: ^uint(0),
							EndPoint: tree_sitter.Point{
								Row:    ^uint(0),
								Column: ^uint(0),
							},
						},
						LocalDefs: nil,
					},
				},
				Captures: queryCaptures,
				Ranges:   ranges,
				Depth:    depth,
			})
		}

		if len(queue) == 0 {
			break
		}

		var next highlightQueueItem
		next, queue = queue[0], append(queue, queue[1:]...)

		config = next.config
		depth = next.depth
		ranges = next.ranges
	}

	return result, nil
}

type iterLayer struct {
	Tree              *tree_sitter.Tree
	Cursor            *tree_sitter.QueryCursor
	Config            Configuration
	HighlightEndStack []uint
	ScopeStack        []localScope
	Captures          *queryCapturesIter
	Ranges            []tree_sitter.Range
	Depth             uint
}

func (h *iterLayer) sortKey() *sortKey {
	depth := -int(h.Depth)

	var nextStart *uint
	if match, index, ok := h.Captures.peek(); ok {
		startByte := match.Captures[index].Node.StartByte()
		nextStart = &startByte
	}

	var nextEnd *uint
	if len(h.HighlightEndStack) > 0 {
		endByte := h.HighlightEndStack[len(h.HighlightEndStack)-1]
		nextEnd = &endByte
	}

	switch {
	case nextStart != nil && nextEnd != nil:
		if *nextStart < *nextEnd {
			return &sortKey{
				offset: *nextStart,
				start:  true,
				depth:  depth,
			}
		} else {
			return &sortKey{
				offset: *nextEnd,
				start:  false,
				depth:  depth,
			}
		}
	case nextStart != nil:
		return &sortKey{
			offset: *nextStart,
			start:  true,
			depth:  depth,
		}
	case nextEnd != nil:
		return &sortKey{
			offset: *nextEnd,
			start:  false,
			depth:  depth,
		}
	default:
		return nil
	}
}
