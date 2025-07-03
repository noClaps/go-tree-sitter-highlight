# go-tree-sitter-highlight

This highlighter is based on the Rust [tree-sitter-highlight](https://crates.io/crates/tree-sitter-highlight) crate. It provides a simple way to highlight text via [tree-sitter](https://github.com/tree-sitter/tree-sitter), using the [go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter) module.

# Usage

To highlight your text:

1. Create a `[highlight.Configuration]`. This struct holds the configuration for the highlighter, including the language, the queries to use.
2. Call `highlight.Configuration.Configure` to configure the capture names used by your theme.
3. Create a new `highlight.Highlighter` and call the `highlight.Highlighter.Highlight` method to highlight your text. This returns a `iter.Seq2[Event, error]` that you can iterate over to get the highlighted text areas & languages in your text.

```go
source := []byte("package main\n\n import \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, World!\")\n}")

captureNames := []string{
	"variable",
	"function",
	"string",
	"keyword",
	"comment",
}

language := tree_sitter.NewLanguage(tree_sitter_go.Language())

cfg, err := NewConfiguration(language, "go", highlightsQuery, injectionQuery, localsQuery)
if err != nil {
	log.Fatal(err)
}

cfg.Configure(captureNames)

highlighter := New()
events := highlighter.Highlight(context.Background(), cfg, source, func(name string) *Configuration {
	return nil
})

for event, err := range events {
	if err != nil {
		log.Fatal(err)
	}

	switch e := event.(type) {
		case EventLayerStart:
			log.Printf("Layer start: %s", e.LanguageName)
		case EventLayerEnd:
			log.Printf("Layer end")
		case EventCaptureStart:
			log.Printf("Capture start: %d", e.Highlight)
		case EventCaptureEnd:
			log.Printf("Capture end")
		case EventSource:
			log.Printf("Source: %d-%d", e.StartByte, e.EndByte)
		}
	}
}
```
