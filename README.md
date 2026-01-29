# codesitter

CLI to run tree-sitter queries across a repo.

## Build

```
go build ./...
```

## Usage

```
./codesitter query --lang go --query '(function_declaration name: (identifier) @name)' --path .
```

Formats:
- pretty (default)
- jsonl
- both (jsonl on stdout, pretty on stderr)

