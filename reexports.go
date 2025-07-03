package highlight

import (
	"unsafe"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"
)

func NewLanguage(ptr unsafe.Pointer) *tree_sitter.Language {
	return tree_sitter.NewLanguage(ptr)
}
