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

// NewHTMLRender returns a new HTMLRender.
func NewHTMLRender() *HTMLRender {
	return &HTMLRender{
		ClassNamePrefix: "hl-",
	}
}

// HTMLRender is a renderer that outputs HTML.
type HTMLRender struct {
	ClassNamePrefix string
}

func (r *HTMLRender) addText(w io.Writer, source []byte, hs []Highlight, languages []string, callback AttributeCallback) error {
	for len(source) > 0 {
		c, l := utf8.DecodeRune(source)
		source = source[l:]

		if c == utf8.RuneError || c == '\r' {
			continue
		}

		if c == '\n' {
			for range len(hs) - 1 {
				if err := r.endHighlight(w); err != nil {
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
				if err := r.startHighlight(w, h, languageName, callback); err != nil {
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

func (r *HTMLRender) startHighlight(w io.Writer, h Highlight, languageName string, callback AttributeCallback) error {
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

func (r *HTMLRender) endHighlight(w io.Writer) error {
	_, err := w.Write([]byte("</span>"))
	return err
}

// Render renders the code code to the writer with spans for each highlight capture.
// The [AttributeCallback] is used to generate the classes or inline styles for each span.
func (r *HTMLRender) Render(w io.Writer, events iter.Seq2[Event, error], source []byte, callback AttributeCallback) error {
	var (
		highlights []Highlight
		languages  []string
	)
	for event, err := range events {
		if err != nil {
			return fmt.Errorf("error while rendering: %w", err)
		}

		switch e := event.(type) {
		case EventLayerStart:
			highlights = append(highlights, DefaultHighlight)
			languages = append(languages, e.LanguageName)
		case EventLayerEnd:
			highlights = highlights[:len(highlights)-1]
			languages = languages[:len(languages)-1]
		case EventCaptureStart:
			highlights = append(highlights, e.Highlight)
			language := languages[len(languages)-1]
			if err = r.startHighlight(w, e.Highlight, language, callback); err != nil {
				return fmt.Errorf("error while starting highlight: %w", err)
			}
		case EventCaptureEnd:
			highlights = highlights[:len(highlights)-1]
			if err = r.endHighlight(w); err != nil {
				return fmt.Errorf("error while ending highlight: %w", err)
			}
		case EventSource:
			if err = r.addText(w, source[e.StartByte:e.EndByte], highlights, languages, callback); err != nil {
				return fmt.Errorf("error while writing source: %w", err)
			}
		}
	}

	return nil
}

// RenderCSS renders the css classes for a theme to the writer.
func (r *HTMLRender) RenderCSS(w io.Writer, theme map[string]string) error {
	for name, style := range theme {
		_, err := fmt.Fprintf(w, ".%s%s{%s}", r.ClassNamePrefix, name, style)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *HTMLRender) themeAttributeCallback(captureNames []string) AttributeCallback {
	return func(h Highlight, languageName string) []byte {
		if h == DefaultHighlight {
			return nil
		}

		return []byte(fmt.Sprintf(`class="%s%s"`, r.ClassNamePrefix, captureNames[h]))
	}

}

// RenderDocument renders a full HTML document with the code and theme embedded.
func (r *HTMLRender) RenderDocument(w io.Writer, events iter.Seq2[Event, error], title string, source []byte, captureNames []string, theme map[string]string) error {
	if _, err := fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>%s</title>
<style>
`, title); err != nil {
		return err
	}

	if err := r.RenderCSS(w, theme); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(w, `</style>
</head>
<body>
<pre><code>
`); err != nil {
		return err
	}

	if err := r.Render(w, events, source, r.themeAttributeCallback(captureNames)); err != nil {
		return err
	}

	_, err := fmt.Fprintf(w, `</code></pre>
</body>
</html>
`)
	return err
}
