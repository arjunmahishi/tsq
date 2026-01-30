package main

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/urfave/cli/v3"
)

//go:embed example_queries.txt
var examplesText string

func examplesCommand() *cli.Command {
	return &cli.Command{
		Name:  "example-queries",
		Usage: "show example tree-sitter queries",
		Description: "Print example query patterns for common Go code structures.\n" +
			"Output is designed to be grep-friendly.\n\n" +
			"Examples:\n" +
			"  tsq example-queries                   # show all examples\n" +
			"  tsq example-queries | grep func       # find function-related patterns\n" +
			"  tsq example-queries | grep -A2 struct # find struct patterns with context",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Print(examplesText)
			return nil
		},
	}
}
