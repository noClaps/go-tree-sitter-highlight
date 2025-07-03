package language

import (
	"unsafe"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type Language struct {
	Name            string
	HighlightsQuery []byte
	InjectionQuery  []byte
	LocalsQuery     []byte
	Lang            *tree_sitter.Language
}

func NewLanguage(name string, ptr unsafe.Pointer, highlightsQuery, injectionQuery, localsQuery []byte) Language {
	return Language{
		Name:            name,
		HighlightsQuery: highlightsQuery,
		InjectionQuery:  injectionQuery,
		LocalsQuery:     localsQuery,
		Lang:            tree_sitter.NewLanguage(ptr),
	}
}
