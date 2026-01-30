package tsq

import sitter "github.com/smacker/go-tree-sitter"

// Language defines the interface for a supported programming language.
type Language interface {
	// Name returns the language identifier (e.g., "go", "python").
	Name() string

	// Extensions returns file extensions for this language (e.g., [".go"]).
	Extensions() []string

	// TreeSitterLang returns the tree-sitter language grammar.
	TreeSitterLang() *sitter.Language

	// SymbolsQuery returns the tree-sitter query for extracting symbols.
	SymbolsQuery() string

	// OutlineQuery returns the tree-sitter query for file outline.
	OutlineQuery() string

	// RefsQuery returns the tree-sitter query for finding references.
	RefsQuery() string
}

// registry holds all registered languages.
var registry = make(map[string]Language)

// Register adds a language to the registry.
// This is typically called from init() functions in language implementation files.
func Register(lang Language) {
	registry[lang.Name()] = lang
}

// Get returns a language by name, or nil if not found.
func Get(name string) Language {
	return registry[name]
}

// List returns all registered language names.
func List() []string {
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}

// ByExtension finds a language by file extension.
func ByExtension(ext string) Language {
	for _, lang := range registry {
		for _, e := range lang.Extensions() {
			if e == ext {
				return lang
			}
		}
	}
	return nil
}
