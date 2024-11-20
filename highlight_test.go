package highlight

import (
	"context"
	"log"
	"os"
	"slices"
	"testing"

	"github.com/tree-sitter/go-tree-sitter"
	"github.com/tree-sitter/tree-sitter-go/bindings/go"
)

const (
	ResetStyle = "\x1b[m"

	GreenStyle   = "\x1b[32m"
	BlueStyle    = "\x1b[34m"
	MagentaStyle = "\x1b[35m"
)

var theme = map[string]string{
	"variable": BlueStyle,
	"keyword":  MagentaStyle,
	"string":   GreenStyle,
}

func TestHighlighter_Highlight(t *testing.T) {
	highlightsQuery, err := os.ReadFile("testdata/highlights.scm")
	if err != nil {
		log.Fatalf("failed to read highlights query: %v", err)
	}

	source, err := os.ReadFile("testdata/source.go")
	if err != nil {
		log.Fatalf("failed to read source file: %v", err)
	}

	language := tree_sitter.NewLanguage(tree_sitter_go.Language())

	cfg, err := NewConfiguration(language, "go", highlightsQuery, nil, nil)
	if err != nil {
		log.Fatalf("failed to create highlight config: %v", err)
	}

	captureNames := make([]string, len(theme))
	for name := range theme {
		captureNames = append(captureNames, name)
	}

	cfg.Configure(captureNames)

	highlighter := New()
	highlights := highlighter.Highlight(context.Background(), *cfg, source, func(name string) *Configuration {
		log.Println("loading highlight config for", name)
		return nil
	})

	var (
		activeHighlights []Highlight
		usedCaptureNames []string
	)
	for event, err := range highlights {
		if err != nil {
			log.Panicf("failed to highlight source: %v", err)
		}
		switch e := event.(type) {
		case EventStart:
			activeHighlights = append(activeHighlights, e.Highlight)
		case EventEnd:
			activeHighlights = activeHighlights[:len(activeHighlights)-1]
		case EventSource:
			var style string
			if len(activeHighlights) > 0 {
				activeHighlight := activeHighlights[len(activeHighlights)-1]
				captureName := captureNames[activeHighlight]
				usedCaptureNames = append(usedCaptureNames, captureName)
				style = theme[captureName]
			}
			renderStyle(style, string(source[e.Start:e.End]))
		}
	}
	t.Logf("used capture names: %v", slices.Compact(usedCaptureNames))
}

func renderStyle(style string, source string) {
	if style == "" {
		print(source)
		return
	}
	print(style + source + ResetStyle)
}
