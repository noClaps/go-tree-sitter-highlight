package highlight

import (
	"context"
	"iter"

	"github.com/noclaps/go-tree-sitter-highlight/internal/events"
	"github.com/noclaps/go-tree-sitter-highlight/internal/highlight"
	"github.com/noclaps/go-tree-sitter-highlight/internal/html"
	ts_iter "github.com/noclaps/go-tree-sitter-highlight/internal/iter"
	"github.com/noclaps/go-tree-sitter-highlight/types"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

// Highlight highlights the given source code using the given configuration. The source code is expected to be UTF-8 encoded.
// The function returns the highlighted HTML or an error.
func Highlight(cfg types.Configuration, source string, injectionCallback types.InjectionCallback, attributeCallback types.AttributeCallback) (string, error) {
	h := &highlight.Highlighter{
		Parser: tree_sitter.NewParser(),
	}
	layers, err := ts_iter.NewIterLayers([]byte(source), "", h, types.InjectionCallback(injectionCallback), types.Configuration(cfg), 0, []tree_sitter.Range{
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

	i := &ts_iter.Iterator{
		Ctx:                context.Background(),
		Source:             []byte(source),
		LanguageName:       cfg.LanguageName,
		ByteOffset:         0,
		Highlighter:        h,
		InjectionCallback:  types.InjectionCallback(injectionCallback),
		Layers:             layers,
		NextEvents:         nil,
		LastHighlightRange: nil,
	}
	i.SortLayers()

	var events iter.Seq2[events.Event, error] = func(yield func(events.Event, error) bool) {
		for {
			event, err := i.Next()
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

	return html.Render(events, source, types.AttributeCallback(attributeCallback))
}
