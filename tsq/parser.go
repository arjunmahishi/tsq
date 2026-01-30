package tsq

import (
	"fmt"
	"os"

	sitter "github.com/smacker/go-tree-sitter"
)

// parser wraps a tree-sitter parser for a specific language.
type parser struct {
	parser *sitter.Parser
	lang   Language
}

// newParser creates a new parser for the given language.
func newParser(language Language) *parser {
	p := sitter.NewParser()
	p.SetLanguage(language.TreeSitterLang())
	return &parser{
		parser: p,
		lang:   language,
	}
}

// parse parses source code and returns the syntax tree.
func (p *parser) parse(source []byte) *sitter.Tree {
	return p.parser.Parse(nil, source)
}

// parseFile reads and parses a file.
func (p *parser) parseFile(path string) (*sitter.Tree, []byte, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read file: %w", err)
	}
	return p.parse(source), source, nil
}

// query represents a compiled tree-sitter query.
type query struct {
	query        *sitter.Query
	captureNames []string
}

// newQuery compiles a tree-sitter query string.
func newQuery(queryStr string, language Language) (*query, error) {
	q, err := sitter.NewQuery([]byte(queryStr), language.TreeSitterLang())
	if err != nil {
		return nil, fmt.Errorf("compile query: %w", err)
	}

	captureCount := int(q.CaptureCount())
	captureNames := make([]string, captureCount)
	for i := 0; i < captureCount; i++ {
		captureNames[i] = q.CaptureNameForId(uint32(i))
	}

	return &query{
		query:        q,
		captureNames: captureNames,
	}, nil
}

// run executes the query on a syntax tree and returns matches.
func (q *query) run(tree *sitter.Tree, source []byte, displayPath string) []QueryMatch {
	cursor := sitter.NewQueryCursor()
	cursor.Exec(q.query, tree.RootNode())

	var matches []QueryMatch
	for {
		match, ok := cursor.NextMatch()
		if !ok {
			break
		}

		result := QueryMatch{
			File:    displayPath,
			Pattern: int(match.PatternIndex),
		}

		for _, capture := range match.Captures {
			name := q.captureName(capture.Index)
			node := capture.Node
			start := node.StartPoint()
			end := node.EndPoint()

			result.Captures = append(result.Captures, CaptureResult{
				Name:     name,
				NodeType: node.Type(),
				Text:     node.Content(source),
				Range: Range{
					Start: Position{Line: int(start.Row) + 1, Column: int(start.Column) + 1},
					End:   Position{Line: int(end.Row) + 1, Column: int(end.Column) + 1},
				},
			})
		}

		matches = append(matches, result)
	}

	return matches
}

func (q *query) captureName(index uint32) string {
	if int(index) >= len(q.captureNames) {
		return fmt.Sprintf("capture_%d", index)
	}
	return q.captureNames[index]
}
