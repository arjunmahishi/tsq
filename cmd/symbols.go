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
	"unicode"

	"github.com/urfave/cli/v3"
)

// SymbolsResult is the output format for the symbols command.
type SymbolsResult struct {
	File    string         `json:"file"`
	Symbols []types.Symbol `json:"symbols"`
}

// SymbolsCommand returns the symbols subcommand.
func SymbolsCommand() *cli.Command {
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
	path := cmd.String("path")
	file := cmd.String("file")
	visibility := cmd.String("visibility")
	includeSource := cmd.Bool("include-source")
	maxSourceLines := cmd.Int("max-source-lines")
	compact := cmd.Bool("compact")
	jobs := cmd.Int("jobs")
	maxBytes := cmd.Int64("max-bytes")

	language := lang.Get("go")
	if language == nil {
		return errors.New("go language not registered")
	}

	query, err := parser.NewQuery(language.SymbolsQuery(), language)
	if err != nil {
		return err
	}

	out := output.New(output.Config{Compact: compact})

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

	results := make(chan SymbolsResult, 128)
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
			symbols := extractSymbols(matches, source, visibility, includeSource, maxSourceLines)
			if len(symbols) > 0 {
				results <- SymbolsResult{
					File:    job.DisplayPath,
					Symbols: symbols,
				}
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

	for result := range results {
		out.Write(result)
	}

	return nil
}

func extractSymbols(
	matches []types.QueryMatch,
	source []byte,
	visibility string,
	includeSource bool,
	maxSourceLines int,
) []types.Symbol {
	var symbols []types.Symbol

	for _, match := range matches {
		sym := parseSymbolFromMatch(match, source, includeSource, maxSourceLines)
		if sym == nil {
			continue
		}

		// Filter by visibility
		switch visibility {
		case "public":
			if sym.Visibility != "public" {
				continue
			}
		case "private":
			if sym.Visibility != "private" {
				continue
			}
		}

		symbols = append(symbols, *sym)
	}

	return symbols
}

func parseSymbolFromMatch(
	match types.QueryMatch, source []byte, includeSource bool, maxSourceLines int,
) *types.Symbol {
	captures := make(map[string]types.CaptureResult)
	for _, c := range match.Captures {
		captures[c.Name] = c
	}

	var sym types.Symbol

	// Determine kind based on capture names
	if _, ok := captures["function"]; ok {
		sym.Kind = "function"
		if name, ok := captures["name"]; ok {
			sym.Name = name.Text
			sym.Range = name.Range
		}
		sym.Signature = buildFuncSignature(captures)
	} else if _, ok := captures["method"]; ok {
		sym.Kind = "method"
		if name, ok := captures["name"]; ok {
			sym.Name = name.Text
			sym.Range = name.Range
		}
		if recv, ok := captures["receiver"]; ok {
			sym.Receiver = extractReceiverType(recv.Text)
		}
		sym.Signature = buildFuncSignature(captures)
	} else if typeDef, ok := captures["type"]; ok {
		if typeSpec, ok := captures["type_def"]; ok {
			if strings.HasPrefix(typeSpec.NodeType, "struct") {
				sym.Kind = "struct"
			} else if strings.HasPrefix(typeSpec.NodeType, "interface") {
				sym.Kind = "interface"
			} else {
				sym.Kind = "type"
			}
		} else {
			sym.Kind = "type"
		}
		if name, ok := captures["name"]; ok {
			sym.Name = name.Text
			sym.Range = name.Range
		}
		sym.Range = typeDef.Range
	} else if _, ok := captures["const"]; ok {
		sym.Kind = "const"
		if name, ok := captures["name"]; ok {
			sym.Name = name.Text
			sym.Range = name.Range
		}
	} else if _, ok := captures["var"]; ok {
		sym.Kind = "var"
		if name, ok := captures["name"]; ok {
			sym.Name = name.Text
			sym.Range = name.Range
		}
	} else {
		return nil
	}

	if sym.Name == "" {
		return nil
	}

	// Determine visibility
	sym.Visibility = getVisibility(sym.Name)

	// Include source if requested
	if includeSource {
		for _, c := range match.Captures {
			// Find the outermost capture (function, method, type, const, var)
			if c.Name == "function" || c.Name == "method" || c.Name == "type" || c.Name == "const" || c.Name == "var" {
				sym.Source = truncateSource(c.Text, maxSourceLines)
				sym.Range = c.Range
				break
			}
		}
	}

	sym.File = match.File
	return &sym
}

func getVisibility(name string) string {
	if len(name) == 0 {
		return "private"
	}
	r := rune(name[0])
	if unicode.IsUpper(r) {
		return "public"
	}
	return "private"
}

func buildFuncSignature(captures map[string]types.CaptureResult) string {
	var sb strings.Builder
	sb.WriteString("func")

	if recv, ok := captures["receiver"]; ok {
		sb.WriteString(" ")
		sb.WriteString(recv.Text)
	}

	if name, ok := captures["name"]; ok {
		sb.WriteString(" ")
		sb.WriteString(name.Text)
	}

	if params, ok := captures["params"]; ok {
		sb.WriteString(params.Text)
	}

	if result, ok := captures["result"]; ok {
		sb.WriteString(" ")
		sb.WriteString(result.Text)
	}

	return sb.String()
}

func extractReceiverType(receiver string) string {
	// Extract type from receiver like "(r *MyType)" -> "MyType"
	receiver = strings.TrimPrefix(receiver, "(")
	receiver = strings.TrimSuffix(receiver, ")")
	parts := strings.Fields(receiver)
	if len(parts) >= 2 {
		t := parts[len(parts)-1]
		return strings.TrimPrefix(t, "*")
	}
	if len(parts) == 1 {
		return strings.TrimPrefix(parts[0], "*")
	}
	return receiver
}

func truncateSource(source string, maxLines int) string {
	if maxLines <= 0 {
		return source
	}

	lines := strings.Split(source, "\n")
	if len(lines) <= maxLines {
		return source
	}

	return strings.Join(lines[:maxLines], "\n") + "\n..."
}
