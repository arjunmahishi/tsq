package tsq

import (
	"errors"
	"runtime"
	"strings"
	"sync"
	"unicode"
)

// Query executes a custom tree-sitter query and returns matches.
func Query(opts QueryOptions) ([]QueryMatch, error) {
	if opts.Query == "" {
		return nil, errors.New("query is required")
	}
	if opts.Language == "" {
		opts.Language = "go" // Default to Go
	}
	if opts.Path == "" {
		opts.Path = "."
	}
	if opts.Jobs == 0 {
		opts.Jobs = runtime.NumCPU()
	}
	if opts.MaxBytes == 0 {
		opts.MaxBytes = 2 * 1024 * 1024
	}

	language := Get(opts.Language)
	if language == nil {
		return nil, errors.New(opts.Language + " language not registered")
	}

	query, err := newQuery(opts.Query, language)
	if err != nil {
		return nil, err
	}

	var files []FileJob
	if opts.File != "" {
		sc := newScanner(scannerConfig{language: language})
		job, err := sc.collectSingle(opts.File)
		if err != nil {
			return nil, err
		}
		files = []FileJob{job}
	} else {
		sc := newScanner(scannerConfig{
			root:     opts.Path,
			language: language,
			maxBytes: opts.MaxBytes,
		})
		files, err = sc.collect()
		if err != nil {
			return nil, err
		}
	}

	if len(files) == 0 {
		return []QueryMatch{}, nil
	}

	return runQueryWorkers(language, query, files, opts.Jobs), nil
}

// SymbolsResult is the output format for symbols extraction.
type SymbolsResult struct {
	File    string   `json:"file"`
	Symbols []Symbol `json:"symbols"`
}

// Symbols extracts symbols from code files.
func Symbols(opts SymbolsOptions) ([]SymbolsResult, error) {
	if opts.Language == "" {
		opts.Language = "go"
	}
	if opts.Path == "" {
		opts.Path = "."
	}
	if opts.Visibility == "" {
		opts.Visibility = "all"
	}
	if opts.MaxSourceLines == 0 {
		opts.MaxSourceLines = 10
	}
	if opts.Jobs == 0 {
		opts.Jobs = runtime.NumCPU()
	}
	if opts.MaxBytes == 0 {
		opts.MaxBytes = 2 * 1024 * 1024
	}

	language := Get(opts.Language)
	if language == nil {
		return nil, errors.New(opts.Language + " language not registered")
	}

	query, err := newQuery(language.SymbolsQuery(), language)
	if err != nil {
		return nil, err
	}

	var files []FileJob
	if opts.File != "" {
		sc := newScanner(scannerConfig{language: language})
		job, err := sc.collectSingle(opts.File)
		if err != nil {
			return nil, err
		}
		files = []FileJob{job}
	} else {
		sc := newScanner(scannerConfig{
			root:     opts.Path,
			language: language,
			maxBytes: opts.MaxBytes,
		})
		files, err = sc.collect()
		if err != nil {
			return nil, err
		}
	}

	if len(files) == 0 {
		return []SymbolsResult{}, nil
	}

	return runSymbolsWorkers(language, query, files, opts.Jobs, opts.Visibility, opts.IncludeSource, opts.MaxSourceLines), nil
}

// Outline returns the structural overview of a file.
func Outline(opts OutlineOptions) (FileOutline, error) {
	if opts.File == "" {
		return FileOutline{}, errors.New("file is required")
	}
	if opts.Language == "" {
		opts.Language = "go"
	}
	if opts.MaxSourceLines == 0 {
		opts.MaxSourceLines = 5
	}

	language := Get(opts.Language)
	if language == nil {
		return FileOutline{}, errors.New(opts.Language + " language not registered")
	}

	query, err := newQuery(language.OutlineQuery(), language)
	if err != nil {
		return FileOutline{}, err
	}

	sc := newScanner(scannerConfig{language: language})
	job, err := sc.collectSingle(opts.File)
	if err != nil {
		return FileOutline{}, err
	}

	p := newParser(language)
	tree, source, err := p.parseFile(job.AbsPath)
	if err != nil {
		return FileOutline{}, err
	}

	matches := query.run(tree, source, job.DisplayPath)
	outline := buildOutline(job.DisplayPath, matches, source, opts.IncludeSource, opts.MaxSourceLines)
	return outline, nil
}

// RefsResult is the output format for reference finding.
type RefsResult struct {
	Symbol     string      `json:"symbol"`
	References []Reference `json:"references"`
}

// Refs finds references to a symbol.
func Refs(opts RefsOptions) (*RefsResult, error) {
	if opts.Symbol == "" {
		return nil, errors.New("symbol is required")
	}
	if opts.Language == "" {
		opts.Language = "go"
	}
	if opts.Path == "" {
		opts.Path = "."
	}
	if opts.Jobs == 0 {
		opts.Jobs = runtime.NumCPU()
	}
	if opts.MaxBytes == 0 {
		opts.MaxBytes = 2 * 1024 * 1024
	}

	language := Get(opts.Language)
	if language == nil {
		return nil, errors.New(opts.Language + " language not registered")
	}

	query, err := newQuery(language.RefsQuery(), language)
	if err != nil {
		return nil, err
	}

	var files []FileJob
	if opts.File != "" {
		sc := newScanner(scannerConfig{language: language})
		job, err := sc.collectSingle(opts.File)
		if err != nil {
			return nil, err
		}
		files = []FileJob{job}
	} else {
		sc := newScanner(scannerConfig{
			root:     opts.Path,
			language: language,
			maxBytes: opts.MaxBytes,
		})
		files, err = sc.collect()
		if err != nil {
			return nil, err
		}
	}

	if len(files) == 0 {
		return &RefsResult{Symbol: opts.Symbol, References: []Reference{}}, nil
	}

	refs := runRefsWorkers(language, query, files, opts.Jobs, opts.Symbol, opts.IncludeContext)
	return &RefsResult{
		Symbol:     opts.Symbol,
		References: refs,
	}, nil
}

// Worker pool for Query
func runQueryWorkers(language Language, query *query, files []FileJob, jobs int) []QueryMatch {
	results := make(chan QueryMatch, 128)
	jobQueue := make(chan FileJob, 128)
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
		p := newParser(language)
		for job := range jobQueue {
			tree, source, err := p.parseFile(job.AbsPath)
			if err != nil {
				continue
			}
			matches := query.run(tree, source, job.DisplayPath)
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

	var allMatches []QueryMatch
	for match := range results {
		allMatches = append(allMatches, match)
	}

	return allMatches
}

// Worker pool for Symbols
func runSymbolsWorkers(
	language Language,
	query *query,
	files []FileJob,
	jobs int,
	visibility string,
	includeSource bool,
	maxSourceLines int,
) []SymbolsResult {
	results := make(chan SymbolsResult, 128)
	jobQueue := make(chan FileJob, 128)
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
		p := newParser(language)
		for job := range jobQueue {
			tree, source, err := p.parseFile(job.AbsPath)
			if err != nil {
				continue
			}
			matches := query.run(tree, source, job.DisplayPath)
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

	var allResults []SymbolsResult
	for result := range results {
		allResults = append(allResults, result)
	}

	return allResults
}

// Worker pool for Refs
func runRefsWorkers(
	language Language,
	query *query,
	files []FileJob,
	jobs int,
	symbolName string,
	includeContext bool,
) []Reference {
	results := make(chan Reference, 128)
	jobQueue := make(chan FileJob, 128)
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
		p := newParser(language)
		for job := range jobQueue {
			tree, source, err := p.parseFile(job.AbsPath)
			if err != nil {
				continue
			}
			matches := query.run(tree, source, job.DisplayPath)
			refs := findReferences(matches, source, symbolName, includeContext)
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

	var allRefs []Reference
	for ref := range results {
		allRefs = append(allRefs, ref)
	}

	return allRefs
}

// Symbol extraction logic
func extractSymbols(
	matches []QueryMatch, source []byte, visibility string, includeSource bool, maxSourceLines int,
) []Symbol {
	var symbols []Symbol

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
	match QueryMatch, source []byte, includeSource bool, maxSourceLines int,
) *Symbol {
	captures := make(map[string]CaptureResult)
	for _, c := range match.Captures {
		captures[c.Name] = c
	}

	var sym Symbol

	// Determine kind based on capture names
	// Check const/var FIRST before checking for "type" capture
	// because const/var have a @type capture for type annotations
	if _, ok := captures["const"]; ok {
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
	} else if _, ok := captures["function"]; ok {
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

func buildFuncSignature(captures map[string]CaptureResult) string {
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

// Outline building logic
func buildOutline(
	file string, matches []QueryMatch, _ []byte, includeSource bool, maxSourceLines int,
) FileOutline {
	outline := FileOutline{
		File:    file,
		Symbols: []Symbol{},
		Imports: []ImportInfo{},
	}

	for _, match := range matches {
		captures := make(map[string]CaptureResult)
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
			imp := ImportInfo{
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
				sym := Symbol{
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
				sym := Symbol{
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
				sym := Symbol{
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
				sym := Symbol{
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
					sym := Symbol{
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
				sym := Symbol{
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
				sym := Symbol{
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

// Reference finding logic
func findReferences(
	matches []QueryMatch, source []byte, symbolName string, includeContext bool,
) []Reference {
	var refs []Reference
	lines := strings.Split(string(source), "\n")

	for _, match := range matches {
		for _, capture := range match.Captures {
			// Check if this capture matches the symbol we're looking for
			if capture.Text != symbolName {
				continue
			}

			ref := Reference{
				Symbol: symbolName,
				File:   match.File,
				Position: Position{
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
