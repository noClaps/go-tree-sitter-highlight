package html

import (
	"fmt"
	"html"
	"iter"
	"slices"

	"github.com/noclaps/go-tree-sitter-highlight/internal/events"
	"github.com/noclaps/go-tree-sitter-highlight/internal/highlight"
	"github.com/noclaps/go-tree-sitter-highlight/types"
)

func addText(source string, hs []types.CaptureIndex, languages []string, callback types.AttributeCallback) string {
	output := ""

	for _, c := range source {
		if c == '\r' {
			continue
		}

		if c == '\n' {
			for range len(hs) - 1 {
				output += endHighlight()
			}

			output += string(c)

			nextLanguage, closeLanguage := iter.Pull(slices.Values(languages))
			defer closeLanguage()

			languageName, _ := nextLanguage()
			for i, h := range hs {
				if i == 0 {
					continue
				}
				output += startHighlight(h, languageName, callback)
				if h == highlight.DefaultHighlight {
					languageName, _ = nextLanguage()
				}
			}

			continue
		}

		output += html.EscapeString(string(c))
	}

	return output
}

func startHighlight(h types.CaptureIndex, languageName string, callback types.AttributeCallback) string {
	output := "<span"

	var attributes string
	if callback != nil {
		attributes = callback(h, languageName)
	}

	if len(attributes) > 0 {
		output += " " + string(attributes)
	}

	output += ">"
	return output
}

func endHighlight() string {
	return "</span>"
}

// render renders the code and returns it as a string, with spans for each highlight capture.
// The [AttributeCallback] is used to generate the classes or inline styles for each span.
func Render(highlightEvents iter.Seq2[events.Event, error], source string, callback types.AttributeCallback) (string, error) {
	output := ""

	var (
		highlights []types.CaptureIndex
		languages  []string
	)
	for event, err := range highlightEvents {
		if err != nil {
			return "", fmt.Errorf("error while rendering: %w", err)
		}

		switch e := event.(type) {
		case events.EventLayerStart:
			highlights = append(highlights, highlight.DefaultHighlight)
			languages = append(languages, e.LanguageName)
		case events.EventLayerEnd:
			highlights = highlights[:len(highlights)-1]
			languages = languages[:len(languages)-1]
		case events.EventCaptureStart:
			highlights = append(highlights, e.Highlight)
			language := languages[len(languages)-1]
			output += startHighlight(e.Highlight, language, callback)
		case events.EventCaptureEnd:
			highlights = highlights[:len(highlights)-1]
			output += endHighlight()
		case events.EventSource:
			output += addText(source[e.StartByte:e.EndByte], highlights, languages, callback)
		}
	}

	return output, nil
}
