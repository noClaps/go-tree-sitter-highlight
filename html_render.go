package highlight

import (
	"fmt"
	"iter"
	"slices"
	"unicode/utf8"
)

var (
	escapeAmpersand   = []byte("&amp;")
	escapeSingle      = []byte("&#39;")
	escapeLessThan    = []byte("&lt;")
	escapeGreaterThan = []byte("&gt;")
	escapeDouble      = []byte("&#34;")
)

// AttributeCallback is a callback function that returns the html element attributes for a highlight span.
// This can be anything from classes, ids, or inline styles.
type AttributeCallback func(h CaptureIndex, languageName string) string

func addText(source []byte, hs []CaptureIndex, languages []string, callback AttributeCallback) string {
	output := ""

	for len(source) > 0 {
		c, l := utf8.DecodeRune(source)
		source = source[l:]

		if c == utf8.RuneError || c == '\r' {
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
				if h == DefaultHighlight {
					languageName, _ = nextLanguage()
				}
			}

			continue
		}

		var b []byte
		switch c {
		case '&':
			b = escapeAmpersand
		case '\'':
			b = escapeSingle
		case '<':
			b = escapeLessThan
		case '>':
			b = escapeGreaterThan
		case '"':
			b = escapeDouble
		default:
			b = []byte(string(c))
		}

		output += string(b)
	}

	return output
}

func startHighlight(h CaptureIndex, languageName string, callback AttributeCallback) string {
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

// Render renders the code and returns it as a string, with spans for each highlight capture.
// The [AttributeCallback] is used to generate the classes or inline styles for each span.
func Render(events iter.Seq2[event, error], source []byte, callback AttributeCallback) (string, error) {
	output := ""

	var (
		highlights []CaptureIndex
		languages  []string
	)
	for event, err := range events {
		if err != nil {
			return "", fmt.Errorf("error while rendering: %w", err)
		}

		switch e := event.(type) {
		case eventLayerStart:
			highlights = append(highlights, DefaultHighlight)
			languages = append(languages, e.LanguageName)
		case eventLayerEnd:
			highlights = highlights[:len(highlights)-1]
			languages = languages[:len(languages)-1]
		case eventCaptureStart:
			highlights = append(highlights, e.Highlight)
			language := languages[len(languages)-1]
			output += startHighlight(e.Highlight, language, callback)
		case eventCaptureEnd:
			highlights = highlights[:len(highlights)-1]
			output += endHighlight()
		case eventSource:
			output += addText(source[e.StartByte:e.EndByte], highlights, languages, callback)
		}
	}

	return output, nil
}
