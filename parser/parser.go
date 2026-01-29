// Package parser provides tree-sitter parsing and query execution.
package parser

import (
	"codesitter/lang"
	"codesitter/types"
	"fmt"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
)

// Parser wraps a tree-sitter parser for a specific language.
type Parser struct {
	parser *sitter.Parser
	lang   lang.Language
}

// New creates a new Parser for the given language.
func New(language lang.Language) *Parser {
	p := sitter.NewParser()
	p.SetLanguage(language.TreeSitterLang())
	return &Parser{
		parser: p,
		lang:   language,
	}
}

// Parse parses source code and returns the syntax tree.
func (p *Parser) Parse(source []byte) *sitter.Tree {
	return p.parser.Parse(nil, source)
}

// ParseFile reads and parses a file.
func (p *Parser) ParseFile(path string) (*sitter.Tree, []byte, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}
	return p.Parse(source), source, nil
}

// Query represents a compiled tree-sitter query.
type Query struct {
	query        *sitter.Query
	captureNames []string
}

// NewQuery compiles a tree-sitter query string.
func NewQuery(queryStr string, language lang.Language) (*Query, error) {
	q, err := sitter.NewQuery([]byte(queryStr), language.TreeSitterLang())
	if err != nil {
		return nil, fmt.Errorf("compile query: %w", err)
	}

	captureCount := int(q.CaptureCount())
	captureNames := make([]string, captureCount)
	for i := 0; i < captureCount; i++ {
		captureNames[i] = q.CaptureNameForId(uint32(i))
	}

	return &Query{
		query:        q,
		captureNames: captureNames,
	}, nil
}

// Run executes the query on a syntax tree and returns matches.
func (q *Query) Run(tree *sitter.Tree, source []byte, displayPath string) []types.QueryMatch {
	cursor := sitter.NewQueryCursor()
	cursor.Exec(q.query, tree.RootNode())

	var matches []types.QueryMatch
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		result := types.QueryMatch{
			File:    displayPath,
			Pattern: int(match.PatternIndex),
		}

		for _, capture := range match.Captures {
			name := q.captureName(capture.Index)
			node := capture.Node
			start := node.StartPoint()
			end := node.EndPoint()

			result.Captures = append(result.Captures, types.CaptureResult{
				Name:     name,
				NodeType: node.Type(),
				Text:     node.Content(source),
				Range: types.Range{
					Start: types.Position{Line: int(start.Row) + 1, Column: int(start.Column) + 1},
					End:   types.Position{Line: int(end.Row) + 1, Column: int(end.Column) + 1},
				},
			})
		}

		matches = append(matches, result)
	}

	return matches
}

func (q *Query) captureName(index uint32) string {
	if int(index) >= len(q.captureNames) {
		return fmt.Sprintf("capture_%d", index)
	}
	return q.captureNames[index]
}
