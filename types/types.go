package types

import tree_sitter "github.com/tree-sitter/go-tree-sitter"

// CaptureIndex represents the index of a capture name.
type CaptureIndex uint

type Configuration struct {
	Language                      *tree_sitter.Language
	LanguageName                  string
	Query                         *tree_sitter.Query
	CombinedInjectionsQuery       *tree_sitter.Query
	LocalsPatternIndex            uint
	HighlightsPatternIndex        uint
	HighlightIndices              []*CaptureIndex
	NonLocalVariablePatterns      []bool
	InjectionContentCaptureIndex  *uint
	InjectionLanguageCaptureIndex *uint
	LocalScopeCaptureIndex        *uint
	LocalDefCaptureIndex          *uint
	LocalDefValueCaptureIndex     *uint
	LocalRefCaptureIndex          *uint
}

// This function runs when tree-sitter encounters an injection. This is when
// another language is embedded into one currently being parsed. For example,
// CSS and JS can be embedded into HTML. If you were parsing HTML, this
// function would run when it encountered CSS or JS, provided it was included
// in `injections.scm`. Simply return a new configuration from inside the
// function for the new language.
type InjectionCallback func(languageName string) *Configuration

// This runs for every single output `<span>` element, and is used to add
// attributes to the element. You can use this to add class names, inline
// styles based on a theme, or whatever else you'd like. For example, if you
// return `class="ts-highlight"` from inside the function, every `<span>`
// element in your output will look like `<span class="ts-highlight">`.
type AttributeCallback func(h CaptureIndex, languageName string) string
