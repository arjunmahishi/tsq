package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"runtime"

	"github.com/arjunmahishi/tsq/tsq"
	"github.com/urfave/cli/v3"
)

func main() {
	// Import side effect: register Go language
	_ = tsq.Go{}

	app := &cli.Command{
		Name:  "tsq",
		Usage: "tree-sitter query tool (like jq for code)",
		Commands: []*cli.Command{
			queryCommand(),
			symbolsCommand(),
			outlineCommand(),
			refsCommand(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		writeError(err)
		os.Exit(1)
	}
}

func queryCommand() *cli.Command {
	return &cli.Command{
		Name:  "query",
		Usage: "run a tree-sitter query",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "query",
				Aliases: []string{"q"},
				Usage:   "tree-sitter query string",
			},
			&cli.StringFlag{
				Name:  "query-file",
				Usage: "path to a tree-sitter query file",
			},
			&cli.StringFlag{
				Name:  "path",
				Value: ".",
				Usage: "root path to scan",
			},
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "single file to query",
			},
			&cli.BoolFlag{
				Name:  "compact",
				Usage: "minimize output for LLM context limits",
			},
			&cli.IntFlag{
				Name:    "jobs",
				Aliases: []string{"j"},
				Value:   runtime.NumCPU(),
				Usage:   "number of parallel workers",
			},
			&cli.Int64Flag{
				Name:  "max-bytes",
				Value: 2 * 1024 * 1024,
				Usage: "skip files larger than this",
			},
		},
		Action: runQuery,
	}
}

func runQuery(_ context.Context, cmd *cli.Command) error {
	queryText := cmd.String("query")
	queryFile := cmd.String("query-file")

	// Resolve query
	querySource, err := resolveQuery(queryText, queryFile)
	if err != nil {
		return err
	}

	opts := tsq.QueryOptions{
		Query:    querySource,
		Language: "go",
		Path:     cmd.String("path"),
		File:     cmd.String("file"),
		Jobs:     cmd.Int("jobs"),
		MaxBytes: cmd.Int64("max-bytes"),
	}

	matches, err := tsq.Query(opts)
	if err != nil {
		return err
	}

	return writeJSON(matches, cmd.Bool("compact"))
}

func resolveQuery(text, filePath string) (string, error) {
	if text != "" && filePath != "" {
		return "", errors.New("use --query or --query-file, not both")
	}
	if text != "" {
		return text, nil
	}
	if filePath == "" {
		return "", errors.New("--query or --query-file is required")
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func symbolsCommand() *cli.Command {
	return &cli.Command{
		Name:  "symbols",
		Usage: "extract symbols from code",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Value: ".",
				Usage: "root path to scan",
			},
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "single file to analyze",
			},
			&cli.StringFlag{
				Name:  "visibility",
				Value: "all",
				Usage: "filter: all, public, private",
			},
			&cli.BoolFlag{
				Name:  "include-source",
				Usage: "include source code snippets",
			},
			&cli.IntFlag{
				Name:  "max-source-lines",
				Value: 10,
				Usage: "max lines for source snippets",
			},
			&cli.BoolFlag{
				Name:  "compact",
				Usage: "minimize output",
			},
			&cli.IntFlag{
				Name:    "jobs",
				Aliases: []string{"j"},
				Value:   runtime.NumCPU(),
				Usage:   "number of parallel workers",
			},
			&cli.Int64Flag{
				Name:  "max-bytes",
				Value: 2 * 1024 * 1024,
				Usage: "skip files larger than this",
			},
		},
		Action: runSymbols,
	}
}

func runSymbols(_ context.Context, cmd *cli.Command) error {
	opts := tsq.SymbolsOptions{
		Language:       "go",
		Path:           cmd.String("path"),
		File:           cmd.String("file"),
		Visibility:     cmd.String("visibility"),
		IncludeSource:  cmd.Bool("include-source"),
		MaxSourceLines: cmd.Int("max-source-lines"),
		Jobs:           cmd.Int("jobs"),
		MaxBytes:       cmd.Int64("max-bytes"),
	}

	results, err := tsq.Symbols(opts)
	if err != nil {
		return err
	}

	return writeJSON(results, cmd.Bool("compact"))
}

func outlineCommand() *cli.Command {
	return &cli.Command{
		Name:  "outline",
		Usage: "get file structure overview",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "file",
				Aliases:  []string{"f"},
				Usage:    "file to analyze (required)",
				Required: true,
			},
			&cli.BoolFlag{
				Name:  "compact",
				Usage: "minimize output",
			},
			&cli.BoolFlag{
				Name:  "include-source",
				Usage: "include source code snippets",
			},
			&cli.IntFlag{
				Name:  "max-source-lines",
				Value: 5,
				Usage: "max lines for source snippets",
			},
		},
		Action: runOutline,
	}
}

func runOutline(_ context.Context, cmd *cli.Command) error {
	opts := tsq.OutlineOptions{
		Language:       "go",
		File:           cmd.String("file"),
		IncludeSource:  cmd.Bool("include-source"),
		MaxSourceLines: cmd.Int("max-source-lines"),
	}

	outline, err := tsq.Outline(opts)
	if err != nil {
		return err
	}

	return writeJSON(outline, cmd.Bool("compact"))
}

func refsCommand() *cli.Command {
	return &cli.Command{
		Name:  "refs",
		Usage: "find references to a symbol",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "symbol",
				Aliases:  []string{"s"},
				Usage:    "symbol name to find references for (required)",
				Required: true,
			},
			&cli.StringFlag{
				Name:  "path",
				Value: ".",
				Usage: "root path to scan",
			},
			&cli.StringFlag{
				Name:    "file",
				Aliases: []string{"f"},
				Usage:   "single file to search",
			},
			&cli.BoolFlag{
				Name:  "compact",
				Usage: "minimize output",
			},
			&cli.BoolFlag{
				Name:  "include-context",
				Value: true,
				Usage: "include surrounding code context",
			},
			&cli.IntFlag{
				Name:    "jobs",
				Aliases: []string{"j"},
				Value:   runtime.NumCPU(),
				Usage:   "number of parallel workers",
			},
			&cli.Int64Flag{
				Name:  "max-bytes",
				Value: 2 * 1024 * 1024,
				Usage: "skip files larger than this",
			},
		},
		Action: runRefs,
	}
}

func runRefs(_ context.Context, cmd *cli.Command) error {
	opts := tsq.RefsOptions{
		Symbol:         cmd.String("symbol"),
		Language:       "go",
		Path:           cmd.String("path"),
		File:           cmd.String("file"),
		IncludeContext: cmd.Bool("include-context"),
		Jobs:           cmd.Int("jobs"),
		MaxBytes:       cmd.Int64("max-bytes"),
	}

	result, err := tsq.Refs(opts)
	if err != nil {
		return err
	}

	return writeJSON(result, cmd.Bool("compact"))
}

// JSON output helpers
func writeJSON(v any, compact bool) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	if !compact {
		enc.SetIndent("", "  ")
	}
	return enc.Encode(v)
}

func writeError(err error) {
	enc := json.NewEncoder(os.Stderr)
	enc.Encode(map[string]string{
		"error": err.Error(),
	})
}
