package tsq

// QueryOptions configures the Query function.
type QueryOptions struct {
	// Query is the tree-sitter query string to execute.
	Query string

	// Language specifies which language to use (e.g., "go").
	Language string

	// Path is the root directory to scan for files.
	// If empty, current directory is used.
	Path string

	// File is a single file to query.
	// If set, Path is ignored.
	File string

	// Jobs is the number of parallel workers.
	// If 0, defaults to number of CPUs.
	Jobs int

	// MaxBytes skips files larger than this size.
	// If 0, no size limit is enforced.
	MaxBytes int64
}

// SymbolsOptions configures the Symbols function.
type SymbolsOptions struct {
	// Language specifies which language to use (e.g., "go").
	Language string

	// Path is the root directory to scan for files.
	// If empty, current directory is used.
	Path string

	// File is a single file to analyze.
	// If set, Path is ignored.
	File string

	// Visibility filters symbols: "all", "public", or "private".
	// Defaults to "all".
	Visibility string

	// IncludeSource includes source code snippets in results.
	IncludeSource bool

	// MaxSourceLines limits the number of lines in source snippets.
	MaxSourceLines int

	// Jobs is the number of parallel workers.
	// If 0, defaults to number of CPUs.
	Jobs int

	// MaxBytes skips files larger than this size.
	// If 0, no size limit is enforced.
	MaxBytes int64
}

// OutlineOptions configures the Outline function.
type OutlineOptions struct {
	// Language specifies which language to use (e.g., "go").
	Language string

	// File is the file to analyze (required).
	File string

	// IncludeSource includes source code snippets in results.
	IncludeSource bool

	// MaxSourceLines limits the number of lines in source snippets.
	MaxSourceLines int
}

// RefsOptions configures the Refs function.
type RefsOptions struct {
	// Symbol is the symbol name to find references for (required).
	Symbol string

	// Language specifies which language to use (e.g., "go").
	Language string

	// Path is the root directory to scan for files.
	// If empty, current directory is used.
	Path string

	// File is a single file to search.
	// If set, Path is ignored.
	File string

	// IncludeContext includes surrounding code context in results.
	IncludeContext bool

	// Jobs is the number of parallel workers.
	// If 0, defaults to number of CPUs.
	Jobs int

	// MaxBytes skips files larger than this size.
	// If 0, no size limit is enforced.
	MaxBytes int64
}
