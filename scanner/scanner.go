// Package scanner provides file discovery for codesitter.
package scanner

import (
	"codesitter/lang"
	"codesitter/types"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// DefaultIgnoreDirs returns the default list of directories to ignore.
func DefaultIgnoreDirs() map[string]struct{} {
	return map[string]struct{}{
		".git":          {},
		".hg":           {},
		".svn":          {},
		".jj":           {},
		"node_modules":  {},
		"vendor":        {},
		"dist":          {},
		"build":         {},
		"target":        {},
		".venv":         {},
		"__pycache__":   {},
		".mypy_cache":   {},
		".pytest_cache": {},
		".next":         {},
		".cache":        {},
		".turbo":        {},
		"coverage":      {},
	}
}

// Config holds scanner configuration.
type Config struct {
	Root       string
	Language   lang.Language
	IgnoreDirs map[string]struct{}
	MaxBytes   int64
}

// Scanner discovers files for processing.
type Scanner struct {
	cfg Config
}

// New creates a new Scanner with the given configuration.
func New(cfg Config) *Scanner {
	if cfg.IgnoreDirs == nil {
		cfg.IgnoreDirs = DefaultIgnoreDirs()
	}
	return &Scanner{cfg: cfg}
}

// Collect finds all matching files and returns them as FileJobs.
func (s *Scanner) Collect() ([]types.FileJob, error) {
	absRoot, err := filepath.Abs(s.cfg.Root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	var jobs []types.FileJob
	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			if path == absRoot {
				return nil
			}
			if s.shouldIgnoreDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}

		if !s.isSupportedFile(d.Name()) {
			return nil
		}

		if s.cfg.MaxBytes > 0 {
			info, err := d.Info()
			if err != nil {
				// Skip files we can't stat
				return nil
			}
			if info.Size() > s.cfg.MaxBytes {
				return nil
			}
		}

		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			rel = path
		}

		jobs = append(jobs, types.FileJob{
			AbsPath:     path,
			DisplayPath: filepath.ToSlash(rel),
		})
		return nil
	})

	if err != nil {
		return nil, err
	}

	return jobs, nil
}

// CollectSingle returns a single file as a FileJob.
func (s *Scanner) CollectSingle(filePath string) (types.FileJob, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return types.FileJob{}, fmt.Errorf("resolve path: %w", err)
	}

	return types.FileJob{
		AbsPath:     absPath,
		DisplayPath: filepath.Base(absPath),
	}, nil
}

func (s *Scanner) shouldIgnoreDir(name string) bool {
	_, ok := s.cfg.IgnoreDirs[name]
	return ok
}

func (s *Scanner) isSupportedFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return false
	}
	for _, e := range s.cfg.Language.Extensions() {
		if ext == e {
			return true
		}
	}
	return false
}
