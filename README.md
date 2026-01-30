# tsq - Tree-Sitter Query Tool

> **Note:** This project is in early development. APIs and CLI flags may change.

**tsq** (tree-sitter query) is a command-line tool and Go library for exploring
code structure using [tree-sitter](https://tree-sitter.github.io/tree-sitter/).
Think of it as `jq` for code - query and extract structured information from
source files.

Currently supports: **Go**. Extensible to other languages.

## Features

- **Query**: Run custom tree-sitter queries on code
- **Symbols**: Extract functions, types, methods, variables, constants
- **Outline**: Get structural overview of a file (package, imports, symbols)
- **Refs**: Find references to symbols across your codebase
- **Fast**: Parallel processing with worker pools
- **Library**: Use as a Go library in your own projects

## Installation

```bash
go install github.com/arjunmahishi/tsq/cmd/tsq@latest
```

Or build from source:

```bash
git clone https://github.com/arjunmahishi/tsq.git
cd tsq
go build -o tsq ./cmd/tsq
```

## CLI Usage

### Query - Run custom tree-sitter queries

```bash
# Query all function declarations (captures whole function)
tsq query -q '(function_declaration) @fn' --path .

# Extract function names only
tsq query -q '(function_declaration name: (identifier) @name)' --path .

# Query from a file
tsq query --query-file myquery.scm --path ./src

# Query a single file
tsq query -q '(type_declaration) @type' --file main.go
```

> **Tip:** Queries need `@name` captures to return useful data. Without captures,
> you'll get matches with empty results. For example:
> - `(function_declaration)` - matches but returns `"captures": null`
> - `(function_declaration) @fn` - captures the whole function node
> - `(function_declaration name: (identifier) @name)` - captures just the function name
>
> Run `tsq examples` for more query patterns.

### Symbols - Extract code symbols

```bash
# Extract all symbols from current directory
tsq symbols --path .

# Extract symbols from a single file
tsq symbols --file main.go

# Filter by visibility
tsq symbols --path . --visibility public

# Include source code
tsq symbols --file main.go --include-source --max-source-lines 5
```

### Outline - Get file structure

```bash
# Get outline of a file
tsq outline --file main.go

# Include source snippets
tsq outline --file main.go --include-source
```

### Refs - Find symbol references

```bash
# Find all references to a symbol
tsq refs --symbol MyFunc --path .

# Search in a single file
tsq refs --symbol MyType --file main.go

# Include surrounding code context
tsq refs --symbol MyVar --path . --include-context
```

### Common Flags

Most commands support these flags:

- `--compact`: Minimize JSON output
- `--jobs`, `-j`: Number of parallel workers (default: CPU count)
- `--max-bytes`: Skip files larger than this (default: 2MB)

## Library Usage

Import tsq as a library in your Go projects:

```go
package main

import (
    "fmt"
    "log"
    
    "github.com/arjunmahishi/tsq/tsq"
)

func main() {
    // Extract symbols from a file
    results, err := tsq.Symbols(tsq.SymbolsOptions{
        File:       "main.go",
        Visibility: "public",
    })
    if err != nil {
        log.Fatal(err)
    }
    
    for _, result := range results {
        fmt.Printf("File: %s\n", result.File)
        for _, sym := range result.Symbols {
            fmt.Printf("  - %s (%s)\n", sym.Name, sym.Kind)
        }
    }
}
```

### API Functions

#### `Query(opts QueryOptions) ([]QueryMatch, error)`
Run a custom tree-sitter query.

#### `Symbols(opts SymbolsOptions) ([]SymbolsResult, error)`
Extract symbols (functions, types, methods, etc.) from code.

#### `Outline(opts OutlineOptions) (FileOutline, error)`
Get the structural overview of a file (package, imports, symbols).

#### `Refs(opts RefsOptions) (*RefsResult, error)`
Find all references to a symbol.

See [GoDoc](https://pkg.go.dev/github.com/arjunmahishi/tsq/tsq) for full API documentation.

## Output Format

All commands output JSON:

```json
{
  "file": "main.go",
  "symbols": [
    {
      "name": "main",
      "kind": "function",
      "visibility": "private",
      "file": "main.go",
      "range": {
        "start": {"line": 10, "column": 1},
        "end": {"line": 15, "column": 2}
      }
    }
  ]
}
```
