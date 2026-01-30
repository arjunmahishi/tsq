# AGENTS.md - tsq Development Guide

This file provides guidance for AI coding agents working on the tsq codebase.
Always updated this file when the information here becomes stale.

## Project Overview

**tsq** (tree-sitter query) is a CLI tool and Go library for exploring code
structure using tree-sitter. Think of it as `jq` for code. This is meant to be
used by LLMs in an AI-assisted coding context.

Module: `github.com/arjunmahishi/tsq`

## Project Structure

```
tsq/
├── cmd/tsq/main.go      # CLI wrapper ONLY (flags, JSON output, no business logic)
├── tsq/                 # Public API library
│   ├── codesitter.go    # Main API: Query(), Symbols(), Outline(), Refs()
│   ├── types.go         # Public types (Position, Symbol, FileOutline, etc.)
│   ├── options.go       # Option structs for each API function
│   ├── language.go      # Language interface and registry
│   ├── go.go            # Go language implementation
│   ├── parser.go        # Tree-sitter parsing (internal)
│   ├── scanner.go       # File discovery (internal)
│   └── queries/go/      # Tree-sitter query files (.scm)
├── go.mod
└── README.md
```

## Dogfooding: Use tsq to Explore This Codebase

When working on this project, ALWAYS use tsq itself to understand the code:

```bash
# Get outline of a file
go run ./cmd/tsq outline --file tsq/codesitter.go

# List all public symbols
go run ./cmd/tsq symbols --path tsq/ --visibility public

# Find references to a symbol
go run ./cmd/tsq refs --symbol Language --path .

# Run a custom tree-sitter query
go run ./cmd/tsq query -q '(function_declaration name: (identifier) @name)' --path tsq/
```

- When you feel like there is something limiting about this tool, make a note of
  it and call it out once you're done making the current change.
- If you find something intersting and useful, call that out too
- You should only use grep, sed, read and other tools when tsq cannot do what you need.

## Architecture Guidelines

### CLI vs Library Separation

**CLI (`cmd/tsq/main.go`)** - Thin wrapper only:
- Parse command-line flags using urfave/cli
- Call `tsq.Query()`, `tsq.Symbols()`, etc.
- Format output as JSON
- NO business logic

**Library (`tsq/`)** - All the logic:
- Public API functions
- Language interface and implementations
- Parser, scanner, query execution
- All types and options

### Adding a New Language

1. Create `tsq/<lang>.go` implementing the `Language` interface
2. Add query files in `tsq/queries/<lang>/` (symbols.scm, outline.scm, refs.scm)
3. Use `//go:embed` to embed query files
4. Register in `init()` with `Register(&MyLang{})`

Example:
```go
//go:embed queries/python/symbols.scm
var pythonSymbolsQuery string

type Python struct{}

func init() {
    Register(&Python{})
}

func (p *Python) Name() string { return "python" }
// ... implement other interface methods
```

### Worker Pool Pattern

For operations across multiple files, use the worker pool pattern:
- Create buffered channels for jobs and results
- Spawn N workers (default: `runtime.NumCPU()`)
- Feed jobs, collect results, wait for completion

See `runQueryWorkers()`, `runSymbolsWorkers()`, `runRefsWorkers()` in `codesitter.go`.

## Tree-Sitter Query Files (.scm)

Query files use S-expression syntax. Capture names (prefixed with `@`) become
keys in the result. See existing queries in `tsq/queries/go/` for examples.

```scheme
; Function declarations
(function_declaration
  name: (identifier) @name
  parameters: (parameter_list) @params
  result: (_)? @result) @function
```

## Dependencies

- `github.com/smacker/go-tree-sitter` - Tree-sitter Go bindings
- `github.com/smacker/go-tree-sitter/golang` - Go language grammar
- `github.com/urfave/cli/v3` - CLI framework
- `github.com/cockroachdb/datadriven` - Data-driven testing
- `github.com/stretchr/testify` - Test assertions

## Testing

### Data-Driven Test Harness

Tests use `github.com/cockroachdb/datadriven` for declarative, data-driven testing.
**No mocking** - tests generate real Go code files on the fly and run the APIs against them.

**Test files location:** `tsq/testdata/`
- `symbols.txt` - Symbol extraction tests
- `outline.txt` - File outline tests
- `refs.txt` - Reference finding tests
- `query.txt` - Custom query tests

**Test file format:**
```
# Comments start with #

file name=example.go
package main

func Hello() {}
----

symbols file=example.go visibility=public
----
function Hello public
```

**Commands available in test files:**

| Command | Args | Description |
|---------|------|-------------|
| `file` | `name=<path>` | Create a file with the input content |
| `query` | `q=<query>` `[file=<name>]` | Run tsq.Query() |
| `symbols` | `[file=<name>]` `[visibility=all\|public\|private]` | Run tsq.Symbols() |
| `outline` | `file=<name>` | Run tsq.Outline() |
| `refs` | `symbol=<name>` `[file=<name>]` | Run tsq.Refs() |

**Writing new tests:**
1. Add test cases to existing `testdata/*.txt` files or create new ones
2. Use `file` command to create code snippets
3. Call the API with appropriate command and args
4. Specify expected output after `----`
5. Run with `-rewrite` to capture initial output, then verify correctness

## Debugging tree-sitter queries

Use the `query` command to experiment:
```bash
go run ./cmd/tsq query -q '(function_declaration) @fn' --file tsq/codesitter.go
```

Check the captures and adjust your query accordingly.
