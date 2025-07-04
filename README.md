# go-tree-sitter-highlight

This highlighter is based on the Rust [tree-sitter-highlight](https://crates.io/crates/tree-sitter-highlight) crate. It provides a simple way to highlight text via [tree-sitter](https://github.com/tree-sitter/tree-sitter), using the [go-tree-sitter](https://github.com/tree-sitter/go-tree-sitter) module.

# Usage

Add the package as a dependency:

```sh
go get -u github.com/noclaps/go-tree-sitter-highlight
```

Then you can use it in your code:

```go
package main

import (
	"os"
	"fmt"

	tsh "github.com/noclaps/go-tree-sitter-highlight"
	tsh_language "github.com/noclaps/go-tree-sitter-highlight/language"
	tsh_types "github.com/noclaps/go-tree-sitter-highlight/types"

	tree_sitter_go "github.com/tree-sitter/tree-sitter-go"
)

func getLang(langName string) tsh_language.Language {
	// Here you would have a switch statement to select the language based on `langName`.

	highlights, _ := os.ReadFile("path/to/highlights.scm")
	injections, _ := os.ReadFile("path/to/injections.scm")
	locals, _ := os.ReadFile("path/to/locals.scm")

	language := tsh_language.NewLanguage(langName, tree_sitter_go.Language(), highlights, injections, locals)
	return language
}

func main() {
	code := `
package main

import "fmt"

func fib(n int) int {
	a := 0
	b := 1
	for range n {
		a, b = b, a+b
	}
	return a
}

func main() {
	fmt.Println(fib(10))
}
`

	language := getLang("go")

	// The node names you want to match. These can be anything, and each language
	// has its own set of queries that you can look at for relevant names.
	highlightNames := []string{"function", "variable", "keyword", "constant"}
	config, _ := tsh.NewConfiguration(language, highlightNames)

	// This function runs when tree-sitter encounters an injection. This is when
	// another language is embedded into one currently being parsed. For example,
	// CSS and JS can be embedded into HTML. If you were parsing HTML, this
	// function would run when it encountered CSS or JS, provided it was included
	// in `injections.scm`. Simply return a new configuration from inside the
	// function for the new language.
	var injectionCallback tsh_types.InjectionCallback = func(languageName string) *tsh_types.Configuration {
		language := getLang("go")
		config, _ := tsh.NewConfiguration(language, highlightNames)
		return config
	}

	// This runs for every single output `<span>` element, and is used to add
	// attributes to the element. You can use this to add class names, inline
	// styles based on a theme, or whatever else you'd like. For example, if you
	// return `class="ts-highlight"` from inside the function, every `<span>`
	// element in your output will look like `<span class="ts-highlight">`.
	var attributeCallback tsh_types.AttributeCallback = func(h tsh_types.CaptureIndex, languageName string) string {
		className := highlightNames[h]
		return fmt.Sprintf(`class="%s"`, className)
	}
	highlightedText, _ := tsh.Highlight(*config, code, injectionCallback, attributeCallback)

	fmt.Println(highlightedText) // <span class="..."> ... </span>
}
```

The errors have been omitted for brevity, but should be handled properly when using the library.
