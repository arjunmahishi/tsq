// Package types defines shared data types for codesitter.
package types

// Position represents a location in a source file.
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// Range represents a span in a source file.
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Symbol represents a code symbol (function, type, variable, etc).
type Symbol struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`       // function, type, method, var, const, interface, struct, field
	Visibility string `json:"visibility"` // public, private
	File       string `json:"file"`
	Range      Range  `json:"range"`
	Signature  string `json:"signature,omitempty"` // function signature or type definition
	Source     string `json:"source,omitempty"`    // actual source code (optional)
	Receiver   string `json:"receiver,omitempty"`  // for methods: the receiver type
	Doc        string `json:"doc,omitempty"`       // documentation comment
}

// ImportInfo represents an import statement.
type ImportInfo struct {
	Path  string `json:"path"`
	Alias string `json:"alias,omitempty"`
}

// Outline represents the structural overview of a file.
type Outline struct {
	File    string       `json:"file"`
	Package string       `json:"package"`
	Imports []ImportInfo `json:"imports,omitempty"`
	Symbols []Symbol     `json:"symbols"`
}

// Reference represents a usage of a symbol.
type Reference struct {
	Symbol   string   `json:"symbol"`
	Kind     string   `json:"kind"` // call, type_ref, field_access, identifier
	File     string   `json:"file"`
	Position Position `json:"position"`
	Context  string   `json:"context,omitempty"` // surrounding code snippet
}

// QueryMatch represents a raw tree-sitter query match.
type QueryMatch struct {
	File     string          `json:"file"`
	Pattern  int             `json:"pattern"`
	Captures []CaptureResult `json:"captures"`
}

// CaptureResult represents a single capture within a query match.
type CaptureResult struct {
	Name     string `json:"name"`
	NodeType string `json:"node_type"`
	Text     string `json:"text"`
	Range    Range  `json:"range"`
}

// FileJob represents a file to be processed.
type FileJob struct {
	AbsPath     string
	DisplayPath string
}
