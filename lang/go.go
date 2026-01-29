package lang

import (
	_ "embed"

	sitter "github.com/smacker/go-tree-sitter"
	golang "github.com/smacker/go-tree-sitter/golang"
)

//go:embed queries/go/symbols.scm
var goSymbolsQuery string

//go:embed queries/go/outline.scm
var goOutlineQuery string

//go:embed queries/go/refs.scm
var goRefsQuery string

// Go implements the Language interface for Go source code.
type Go struct{}

func init() {
	Register(&Go{})
}

func (g *Go) Name() string {
	return "go"
}

func (g *Go) Extensions() []string {
	return []string{".go"}
}

func (g *Go) TreeSitterLang() *sitter.Language {
	return golang.GetLanguage()
}

func (g *Go) SymbolsQuery() string {
	return goSymbolsQuery
}

func (g *Go) OutlineQuery() string {
	return goOutlineQuery
}

func (g *Go) RefsQuery() string {
	return goRefsQuery
}
