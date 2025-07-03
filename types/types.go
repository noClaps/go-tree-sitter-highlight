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

// InjectionCallback is called when a language injection is found to load the configuration for the injected language.
type InjectionCallback func(languageName string) *Configuration

// AttributeCallback is a callback function that returns the html element attributes for a highlight span.
// This can be anything from classes, ids, or inline styles.
type AttributeCallback func(h CaptureIndex, languageName string) string
