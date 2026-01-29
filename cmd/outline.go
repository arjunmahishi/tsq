package cmd

import (
	"codesitter/lang"
	"codesitter/output"
	"codesitter/parser"
	"codesitter/scanner"
	"codesitter/types"
	"context"
	"errors"
	"strings"

	"github.com/urfave/cli/v3"
)

// OutlineCommand returns the outline subcommand.
func OutlineCommand() *cli.Command {
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
	file := cmd.String("file")
	compact := cmd.Bool("compact")
	includeSource := cmd.Bool("include-source")
	maxSourceLines := cmd.Int("max-source-lines")

	if file == "" {
		return errors.New("--file is required")
	}

	language := lang.Get("go")
	if language == nil {
		return errors.New("go language not registered")
	}

	query, err := parser.NewQuery(language.OutlineQuery(), language)
	if err != nil {
		return err
	}

	sc := scanner.New(scanner.Config{Language: language})
	job, err := sc.CollectSingle(file)
	if err != nil {
		return err
	}

	p := parser.New(language)
	tree, source, err := p.ParseFile(job.AbsPath)
	if err != nil {
		return err
	}

	matches := query.Run(tree, source, job.DisplayPath)
	outline := buildOutline(job.DisplayPath, matches, source, includeSource, maxSourceLines)

	out := output.New(output.Config{Compact: compact})
	return out.Write(outline)
}

func buildOutline(
	file string, matches []types.QueryMatch, _ []byte, includeSource bool, maxSourceLines int,
) types.Outline {
	outline := types.Outline{
		File:    file,
		Symbols: []types.Symbol{},
		Imports: []types.ImportInfo{},
	}

	for _, match := range matches {
		captures := make(map[string]types.CaptureResult)
		for _, c := range match.Captures {
			captures[c.Name] = c
		}

		// Package
		if pkg, ok := captures["package"]; ok {
			outline.Package = pkg.Text
			continue
		}

		// Imports
		if path, ok := captures["path"]; ok {
			imp := types.ImportInfo{
				Path: strings.Trim(path.Text, `"`),
			}
			if alias, ok := captures["alias"]; ok {
				imp.Alias = alias.Text
			}
			outline.Imports = append(outline.Imports, imp)
			continue
		}

		// Functions
		if _, ok := captures["function"]; ok {
			if name, ok := captures["func_name"]; ok {
				sym := types.Symbol{
					Kind:       "function",
					Name:       name.Text,
					File:       file,
					Range:      captures["function"].Range,
					Visibility: getVisibility(name.Text),
				}
				if includeSource {
					sym.Source = truncateSource(captures["function"].Text, maxSourceLines)
				}
				outline.Symbols = append(outline.Symbols, sym)
			}
			continue
		}

		// Methods
		if _, ok := captures["method"]; ok {
			if name, ok := captures["method_name"]; ok {
				sym := types.Symbol{
					Kind:       "method",
					Name:       name.Text,
					File:       file,
					Range:      captures["method"].Range,
					Visibility: getVisibility(name.Text),
				}
				if recv, ok := captures["receiver_type"]; ok {
					sym.Receiver = strings.TrimPrefix(recv.Text, "*")
				}
				if includeSource {
					sym.Source = truncateSource(captures["method"].Text, maxSourceLines)
				}
				outline.Symbols = append(outline.Symbols, sym)
			}
			continue
		}

		// Structs
		if _, ok := captures["struct"]; ok {
			if name, ok := captures["type_name"]; ok {
				sym := types.Symbol{
					Kind:       "struct",
					Name:       name.Text,
					File:       file,
					Range:      captures["struct"].Range,
					Visibility: getVisibility(name.Text),
				}
				if includeSource {
					sym.Source = truncateSource(captures["struct"].Text, maxSourceLines)
				}
				outline.Symbols = append(outline.Symbols, sym)
			}
			continue
		}

		// Interfaces
		if _, ok := captures["interface"]; ok {
			if name, ok := captures["type_name"]; ok {
				sym := types.Symbol{
					Kind:       "interface",
					Name:       name.Text,
					File:       file,
					Range:      captures["interface"].Range,
					Visibility: getVisibility(name.Text),
				}
				if includeSource {
					sym.Source = truncateSource(captures["interface"].Text, maxSourceLines)
				}
				outline.Symbols = append(outline.Symbols, sym)
			}
			continue
		}

		// Type aliases and other type declarations
		for _, typeCat := range []string{"type_alias", "type_ptr", "type_slice", "type_map", "type_func"} {
			if typeDecl, ok := captures[typeCat]; ok {
				if name, ok := captures["type_name"]; ok {
					sym := types.Symbol{
						Kind:       "type",
						Name:       name.Text,
						File:       file,
						Range:      typeDecl.Range,
						Visibility: getVisibility(name.Text),
					}
					if includeSource {
						sym.Source = truncateSource(typeDecl.Text, maxSourceLines)
					}
					outline.Symbols = append(outline.Symbols, sym)
				}
				break
			}
		}

		// Constants
		if _, ok := captures["const"]; ok {
			if name, ok := captures["const_name"]; ok {
				sym := types.Symbol{
					Kind:       "const",
					Name:       name.Text,
					File:       file,
					Range:      captures["const"].Range,
					Visibility: getVisibility(name.Text),
				}
				if includeSource {
					sym.Source = truncateSource(captures["const"].Text, maxSourceLines)
				}
				outline.Symbols = append(outline.Symbols, sym)
			}
			continue
		}

		// Variables
		if _, ok := captures["var"]; ok {
			if name, ok := captures["var_name"]; ok {
				sym := types.Symbol{
					Kind:       "var",
					Name:       name.Text,
					File:       file,
					Range:      captures["var"].Range,
					Visibility: getVisibility(name.Text),
				}
				if includeSource {
					sym.Source = truncateSource(captures["var"].Text, maxSourceLines)
				}
				outline.Symbols = append(outline.Symbols, sym)
			}
			continue
		}
	}

	return outline
}
