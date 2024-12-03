package highlight

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tree-sitter/go-tree-sitter"
	"github.com/tree-sitter/tree-sitter-go/bindings/go"
)

var cssTheme = map[string]string{
	"variable": "color: #FEFEF8;",
	"function": "color: #73FBF1;",
	"string":   "color: #B8E466;",
	"keyword":  "color: #A578EA;",
	"comment":  "color: #8A8A8A;",
}

func attributeCallback(captureNames []string) AttributeCallback {
	return func(h Highlight, languageName string) []byte {
		if h == DefaultHighlight {
			return nil
		}

		return []byte(`class="hl-` + captureNames[h] + `"`)
	}
}

func TestHTMLRender_Render(t *testing.T) {
	// Get the capture names from the theme
	captureNames := make([]string, 0, len(theme))
	for name := range theme {
		captureNames = append(captureNames, name)
	}

	source, err := os.ReadFile("testdata/test.go")
	require.NoError(t, err)

	language := tree_sitter.NewLanguage(tree_sitter_go.Language())

	highlightsQuery, err := os.ReadFile("testdata/highlights.scm")
	require.NoError(t, err)

	cfg, err := NewConfiguration(language, "go", highlightsQuery, nil, nil)
	require.NoError(t, err)

	cfg.Configure(captureNames)

	highlighter := New()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	events := highlighter.Highlight(ctx, *cfg, source, func(name string) *Configuration {
		return nil
	})

	f, err := os.Create("out.html")
	require.NoError(t, err)
	defer func() {
		err = f.Close()
		require.NoError(t, err)
	}()

	htmlRender := NewHTMLRender()
	_, err = fmt.Fprintf(f, `<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>Test</title>
<style>`)
	require.NoError(t, err)

	err = htmlRender.RenderCSS(f, cssTheme)
	require.NoError(t, err)

	_, err = fmt.Fprintf(f, `</style>
</head>
<body>
<pre><code>
`)
	require.NoError(t, err)

	err = htmlRender.Render(f, events, source, attributeCallback(captureNames))
	assert.NoError(t, err)

	_, err = fmt.Fprintf(f, `</code></pre>
</body>
</html>
`)
	require.NoError(t, err)
}
