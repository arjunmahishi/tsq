package tsq

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunWorkers tests the generic worker pool for concurrency correctness.
// Run with -race flag to detect race conditions: go test -race
func TestRunWorkers(t *testing.T) {
	tests := []struct {
		name      string
		fileCount int
		jobs      int
	}{
		{"single_file_single_worker", 1, 1},
		{"multiple_files_single_worker", 5, 1},
		{"multiple_files_multiple_workers", 10, 4},
		{"more_workers_than_files", 3, 10},
		{"many_files_high_concurrency", 50, 16},
		{"zero_jobs_defaults_to_one", 5, 0},
		{"empty_files", 0, 4},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create temp directory
			tmpDir, err := os.MkdirTemp("", "tsq-workers-test-*")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			// Generate test files and collect expected function names
			expectedFuncs := generateTestFiles(t, tmpDir, tc.fileCount)

			if tc.fileCount == 0 {
				// Edge case: no files to process
				language := Get("go")
				require.NotNil(t, language)

				query, err := newQuery(`(function_declaration name: (identifier) @name)`, language)
				require.NoError(t, err)

				results := runWorkers(language, query, []FileJob{}, tc.jobs, extractFunctionNames)
				require.Empty(t, results)
				return
			}

			// Scan files
			language := Get("go")
			require.NotNil(t, language)

			scanner := newScanner(scannerConfig{
				root:     tmpDir,
				language: language,
				maxBytes: 2 * 1024 * 1024,
			})

			files, err := scanner.collect()
			require.NoError(t, err)
			require.Len(t, files, tc.fileCount)

			// Create query to find function names
			query, err := newQuery(`(function_declaration name: (identifier) @name)`, language)
			require.NoError(t, err)

			// Run workers with a process function that extracts function names
			results := runWorkers(language, query, files, tc.jobs, extractFunctionNames)

			// Verify results
			require.Len(t, results, tc.fileCount, "should have one result per file")

			// Sort both slices for comparison (order may vary due to concurrency)
			sort.Strings(results)
			sort.Strings(expectedFuncs)

			require.Equal(t, expectedFuncs, results, "all functions should be found exactly once")
		})
	}
}

// generateTestFiles creates N Go files, each with a unique function.
// Returns the expected function names.
func generateTestFiles(t *testing.T, dir string, count int) []string {
	t.Helper()

	var expected []string
	for i := range count {
		funcName := fmt.Sprintf("Func%d", i)
		fileName := fmt.Sprintf("file_%d.go", i)
		filePath := filepath.Join(dir, fileName)

		content := fmt.Sprintf(`package testpkg

func %s() {}
`, funcName)

		err := os.WriteFile(filePath, []byte(content), 0644)
		require.NoError(t, err)

		expected = append(expected, funcName)
	}

	return expected
}

// extractFunctionNames is a process function that extracts function names from matches.
func extractFunctionNames(job FileJob, matches []QueryMatch, _ []byte) []string {
	var names []string
	for _, m := range matches {
		for _, c := range m.Captures {
			if c.Name == "name" {
				names = append(names, c.Text)
			}
		}
	}
	return names
}
