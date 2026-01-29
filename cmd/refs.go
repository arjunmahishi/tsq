package cmd

import (
	"codesitter/lang"
	"codesitter/output"
	"codesitter/parser"
	"codesitter/scanner"
	"codesitter/types"
	"context"
	"errors"
	"runtime"
	"strings"
	"sync"

	"github.com/urfave/cli/v3"
)

// RefsResult is the output format for the refs command.
type RefsResult struct {
	Symbol     string            `json:"symbol"`
	References []types.Reference `json:"references"`
}

// RefsCommand returns the refs subcommand.
func RefsCommand() *cli.Command {
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
	symbol := cmd.String("symbol")
	path := cmd.String("path")
	file := cmd.String("file")
	compact := cmd.Bool("compact")
	includeContext := cmd.Bool("include-context")
	jobs := cmd.Int("jobs")
	maxBytes := cmd.Int64("max-bytes")

	if symbol == "" {
		return errors.New("--symbol is required")
	}

	language := lang.Get("go")
	if language == nil {
		return errors.New("go language not registered")
	}

	query, err := parser.NewQuery(language.RefsQuery(), language)
	if err != nil {
		return err
	}

	var files []types.FileJob
	if file != "" {
		sc := scanner.New(scanner.Config{Language: language})
		job, err := sc.CollectSingle(file)
		if err != nil {
			return err
		}
		files = []types.FileJob{job}
	} else {
		sc := scanner.New(scanner.Config{
			Root:     path,
			Language: language,
			MaxBytes: maxBytes,
		})
		files, err = sc.Collect()
		if err != nil {
			return err
		}
	}

	if len(files) == 0 {
		return nil
	}

	results := make(chan types.Reference, 128)
	jobQueue := make(chan types.FileJob, 128)
	var wg sync.WaitGroup

	workerCount := jobs
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(files) {
		workerCount = len(files)
	}

	worker := func() {
		defer wg.Done()
		p := parser.New(language)
		for job := range jobQueue {
			tree, source, err := p.ParseFile(job.AbsPath)
			if err != nil {
				continue
			}
			matches := query.Run(tree, source, job.DisplayPath)
			refs := findReferences(matches, source, symbol, includeContext)
			for _, ref := range refs {
				results <- ref
			}
		}
	}

	wg.Add(workerCount)
	for i := 0; i < workerCount; i++ {
		go worker()
	}

	go func() {
		for _, f := range files {
			jobQueue <- f
		}
		close(jobQueue)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all references
	var allRefs []types.Reference
	for ref := range results {
		allRefs = append(allRefs, ref)
	}

	out := output.New(output.Config{Compact: compact})
	return out.Write(RefsResult{
		Symbol:     symbol,
		References: allRefs,
	})
}

func findReferences(
	matches []types.QueryMatch, source []byte, symbolName string, includeContext bool,
) []types.Reference {
	var refs []types.Reference
	lines := strings.Split(string(source), "\n")

	for _, match := range matches {
		for _, capture := range match.Captures {
			// Check if this capture matches the symbol we're looking for
			if capture.Text != symbolName {
				continue
			}

			ref := types.Reference{
				Symbol: symbolName,
				File:   match.File,
				Position: types.Position{
					Line:   capture.Range.Start.Line,
					Column: capture.Range.Start.Column,
				},
			}

			// Determine reference kind based on capture name
			switch capture.Name {
			case "call":
				ref.Kind = "call"
			case "type_ref", "composite_type":
				ref.Kind = "type_ref"
			case "field":
				ref.Kind = "field_access"
			case "ident", "short_var":
				ref.Kind = "identifier"
			default:
				ref.Kind = "reference"
			}

			// Add context if requested
			if includeContext {
				lineIdx := capture.Range.Start.Line - 1
				if lineIdx >= 0 && lineIdx < len(lines) {
					ref.Context = strings.TrimSpace(lines[lineIdx])
				}
			}

			refs = append(refs, ref)
		}
	}

	return refs
}
