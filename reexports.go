package highlight

import (
	"unsafe"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

type Language = tree_sitter.Language

func NewLanguage(ptr unsafe.Pointer) *Language {
	return tree_sitter.NewLanguage(ptr)
}
