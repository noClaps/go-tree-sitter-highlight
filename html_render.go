package highlight

import (
	"fmt"
	"io"
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
type AttributeCallback func(h Highlight, languageName string) []byte

func addText(w io.Writer, source []byte, hs []Highlight, languages []string, callback AttributeCallback) error {
	for len(source) > 0 {
		c, l := utf8.DecodeRune(source)
		source = source[l:]

		if c == utf8.RuneError || c == '\r' {
			continue
		}

		if c == '\n' {
			for range len(hs) - 1 {
				if err := endHighlight(w); err != nil {
					return err
				}
			}

			if _, err := w.Write([]byte(string(c))); err != nil {
				return err
			}

			nextLanguage, closeLanguage := iter.Pull(slices.Values(languages))
			defer closeLanguage()

			languageName, _ := nextLanguage()
			for i, h := range hs {
				if i == 0 {
					continue
				}
				if err := startHighlight(w, h, languageName, callback); err != nil {
					return err
				}
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

		if _, err := w.Write(b); err != nil {
			return err
		}
	}

	return nil
}

func startHighlight(w io.Writer, h Highlight, languageName string, callback AttributeCallback) error {
	if _, err := fmt.Fprintf(w, "<span"); err != nil {
		return err
	}

	var attributes []byte
	if callback != nil {
		attributes = callback(h, languageName)
	}

	if len(attributes) > 0 {
		if _, err := w.Write([]byte(" ")); err != nil {
			return err
		}
		if _, err := w.Write(attributes); err != nil {
			return err
		}
	}

	_, err := w.Write([]byte(">"))
	return err
}

func endHighlight(w io.Writer) error {
	_, err := w.Write([]byte("</span>"))
	return err
}

// Render renders the code code to the writer with spans for each highlight capture.
// The [AttributeCallback] is used to generate the classes or inline styles for each span.
func Render(w io.Writer, events iter.Seq2[event, error], source []byte, callback AttributeCallback) error {
	var (
		highlights []Highlight
		languages  []string
	)
	for event, err := range events {
		if err != nil {
			return fmt.Errorf("error while rendering: %w", err)
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
			if err = startHighlight(w, e.Highlight, language, callback); err != nil {
				return fmt.Errorf("error while starting highlight: %w", err)
			}
		case eventCaptureEnd:
			highlights = highlights[:len(highlights)-1]
			if err = endHighlight(w); err != nil {
				return fmt.Errorf("error while ending highlight: %w", err)
			}
		case eventSource:
			if err = addText(w, source[e.StartByte:e.EndByte], highlights, languages, callback); err != nil {
				return fmt.Errorf("error while writing source: %w", err)
			}
		}
	}

	return nil
}
