package tsq

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/cockroachdb/datadriven"
	"github.com/stretchr/testify/require"
)

func TestDataDriven(t *testing.T) {
	datadriven.Walk(t, "testdata", func(t *testing.T, path string) {
		// Create temp dir for this test file
		tmpDir, err := os.MkdirTemp("", "tsq-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		// Track files created by "file" commands
		files := make(map[string]string) // name -> abs path

		datadriven.RunTest(t, path, func(t *testing.T, d *datadriven.TestData) string {
			switch d.Cmd {
			case "file":
				return handleFile(t, d, tmpDir, files)
			case "query":
				return handleQuery(t, d, tmpDir, files)
			case "symbols":
				return handleSymbols(t, d, tmpDir, files)
			case "outline":
				return handleOutline(t, d, tmpDir, files)
			case "refs":
				return handleRefs(t, d, tmpDir, files)
			default:
				t.Fatalf("unknown command: %s", d.Cmd)
				return ""
			}
		})
	})
}

// handleFile creates a file in the temp directory
func handleFile(
	t *testing.T, d *datadriven.TestData, tmpDir string, files map[string]string,
) string {
	var name string
	d.ScanArgs(t, "name", &name)

	absPath := filepath.Join(tmpDir, name)

	// Create parent dirs if needed
	err := os.MkdirAll(filepath.Dir(absPath), 0755)
	require.NoError(t, err)

	// Write file content
	err = os.WriteFile(absPath, []byte(d.Input), 0644)
	require.NoError(t, err)

	files[name] = absPath
	return "" // file command produces no output
}

// handleQuery runs Query() and formats results
func handleQuery(
	t *testing.T, d *datadriven.TestData, tmpDir string, files map[string]string,
) string {
	var query string
	d.ScanArgs(t, "q", &query)

	opts := QueryOptions{
		Query:    query,
		Language: "go",
		Path:     tmpDir,
		Jobs:     1, // single-threaded for deterministic ordering
	}

	// Allow file= to target specific file
	if d.HasArg("file") {
		var fileName string
		d.ScanArgs(t, "file", &fileName)
		opts.File = files[fileName]
		opts.Path = ""
	}

	results, err := Query(opts)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}

	return formatQueryResults(results, tmpDir)
}

// handleSymbols runs Symbols() and formats results
func handleSymbols(
	t *testing.T, d *datadriven.TestData, tmpDir string, files map[string]string,
) string {
	opts := SymbolsOptions{
		Language:   "go",
		Path:       tmpDir,
		Visibility: "all",
		Jobs:       1, // single-threaded for deterministic ordering
	}

	if d.HasArg("file") {
		var fileName string
		d.ScanArgs(t, "file", &fileName)
		opts.File = files[fileName]
		opts.Path = ""
	}

	if d.HasArg("visibility") {
		d.ScanArgs(t, "visibility", &opts.Visibility)
	}

	if d.HasArg("source") {
		opts.IncludeSource = true
		if d.HasArg("maxlines") {
			d.ScanArgs(t, "maxlines", &opts.MaxSourceLines)
		} else {
			opts.MaxSourceLines = 10
		}
	}

	results, err := Symbols(opts)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}

	return formatSymbolsResults(results)
}

// handleOutline runs Outline() and formats results
func handleOutline(
	t *testing.T, d *datadriven.TestData, tmpDir string, files map[string]string,
) string {
	var fileName string
	d.ScanArgs(t, "file", &fileName)

	opts := OutlineOptions{
		Language: "go",
		File:     files[fileName],
	}

	if d.HasArg("source") {
		opts.IncludeSource = true
		if d.HasArg("maxlines") {
			d.ScanArgs(t, "maxlines", &opts.MaxSourceLines)
		} else {
			opts.MaxSourceLines = 5
		}
	}

	result, err := Outline(opts)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}

	return formatOutlineResult(result)
}

// handleRefs runs Refs() and formats results
func handleRefs(
	t *testing.T, d *datadriven.TestData, tmpDir string, files map[string]string,
) string {
	var symbol string
	d.ScanArgs(t, "symbol", &symbol)

	opts := RefsOptions{
		Symbol:   symbol,
		Language: "go",
		Path:     tmpDir,
		Jobs:     1, // single-threaded for deterministic ordering
	}

	if d.HasArg("file") {
		var fileName string
		d.ScanArgs(t, "file", &fileName)
		opts.File = files[fileName]
		opts.Path = ""
	}

	if d.HasArg("context") {
		opts.IncludeContext = true
	}

	result, err := Refs(opts)
	if err != nil {
		return fmt.Sprintf("error: %s", err)
	}

	return formatRefsResult(result)
}

// formatQueryResults formats query matches as text
func formatQueryResults(results []QueryMatch, tmpDir string) string {
	if len(results) == 0 {
		return "(no matches)"
	}

	var lines []string
	for _, match := range results {
		for _, cap := range match.Captures {
			// Make file path relative to tmpDir for cleaner output
			relFile := strings.TrimPrefix(match.File, tmpDir+"/")
			line := fmt.Sprintf("@%s: %s (%s:%d:%d)",
				cap.Name,
				cap.Text,
				relFile,
				cap.Range.Start.Line,
				cap.Range.Start.Column,
			)
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// formatSymbolsResults formats symbols as text
func formatSymbolsResults(results []SymbolsResult) string {
	if len(results) == 0 {
		return "(no symbols)"
	}

	var lines []string
	for _, fileResult := range results {
		for _, sym := range fileResult.Symbols {
			var line string
			if sym.Receiver != "" {
				// Method with receiver
				line = fmt.Sprintf("%s (%s) %s %s",
					sym.Kind,
					sym.Receiver,
					sym.Name,
					sym.Visibility,
				)
			} else {
				// Regular symbol
				line = fmt.Sprintf("%s %s %s",
					sym.Kind,
					sym.Name,
					sym.Visibility,
				)
			}

			if sym.Source != "" {
				// Include source on separate lines, indented
				line += "\n" + indentLines(sym.Source, "  ")
			}

			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// formatOutlineResult formats outline as text
func formatOutlineResult(outline FileOutline) string {
	var lines []string

	if outline.Package != "" {
		lines = append(lines, fmt.Sprintf("package: %s", outline.Package))
	}

	if len(outline.Imports) > 0 {
		lines = append(lines, "imports:")
		for _, imp := range outline.Imports {
			if imp.Alias != "" {
				lines = append(lines, fmt.Sprintf("  %s (alias: %s)", imp.Path, imp.Alias))
			} else {
				lines = append(lines, fmt.Sprintf("  %s", imp.Path))
			}
		}
	}

	if len(outline.Symbols) > 0 {
		lines = append(lines, "symbols:")
		for _, sym := range outline.Symbols {
			var symLine string
			if sym.Receiver != "" {
				symLine = fmt.Sprintf("  %s (%s) %s %s",
					sym.Kind,
					sym.Receiver,
					sym.Name,
					sym.Visibility,
				)
			} else {
				symLine = fmt.Sprintf("  %s %s %s",
					sym.Kind,
					sym.Name,
					sym.Visibility,
				)
			}

			if sym.Source != "" {
				symLine += "\n" + indentLines(sym.Source, "    ")
			}

			lines = append(lines, symLine)
		}
	}

	if len(lines) == 0 {
		return "(empty outline)"
	}

	return strings.Join(lines, "\n")
}

// formatRefsResult formats references as text
func formatRefsResult(result *RefsResult) string {
	if len(result.References) == 0 {
		return "(no references)"
	}

	// Sort references by file, line, column for deterministic output
	refs := make([]Reference, len(result.References))
	copy(refs, result.References)
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].File != refs[j].File {
			return refs[i].File < refs[j].File
		}
		if refs[i].Position.Line != refs[j].Position.Line {
			return refs[i].Position.Line < refs[j].Position.Line
		}
		return refs[i].Position.Column < refs[j].Position.Column
	})

	var lines []string
	for _, ref := range refs {
		line := fmt.Sprintf("%s %s:%d:%d",
			ref.Kind,
			filepath.Base(ref.File),
			ref.Position.Line,
			ref.Position.Column,
		)

		if ref.Context != "" {
			line += fmt.Sprintf(" | %s", ref.Context)
		}

		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// indentLines adds indent prefix to each line
func indentLines(text, indent string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}
