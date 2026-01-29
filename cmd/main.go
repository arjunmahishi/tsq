package main

import (
	"codesitter/cmd"
	_ "codesitter/lang" // Register languages
	"context"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "codesitter",
		Usage: "explore code with tree-sitter",
		Commands: []*cli.Command{
			cmd.QueryCommand(),
			cmd.SymbolsCommand(),
			cmd.OutlineCommand(),
			cmd.RefsCommand(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
