package cmd

import (
	"codesitter/lang"
	"codesitter/output"
	"codesitter/parser"
	"codesitter/scanner"
	"codesitter/types"
	"context"
	"errors"
	"os"
	"runtime"
	"sync"

	"github.com/urfave/cli/v3"
)

// QueryCommand returns the query subcommand.
func QueryCommand() *cli.Command {
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
	path := cmd.String("path")
	file := cmd.String("file")
	compact := cmd.Bool("compact")
	jobs := cmd.Int("jobs")
	maxBytes := cmd.Int64("max-bytes")

	// Resolve query
	querySource, err := resolveQuery(queryText, queryFile)
	if err != nil {
		return err
	}

	// Get Go language (hardcoded for now)
	language := lang.Get("go")
	if language == nil {
		return errors.New("go language not registered")
	}

	// Compile query
	query, err := parser.NewQuery(querySource, language)
	if err != nil {
		return err
	}

	// Create output writer
	out := output.New(output.Config{Compact: compact})

	// Collect files
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

	// Run query with worker pool
	results := make(chan types.QueryMatch, 128)
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
			for _, m := range matches {
				results <- m
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

	for match := range results {
		out.Write(match)
	}

	return nil
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
