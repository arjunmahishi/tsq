package tsq

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// defaultIgnoreDirs returns the default list of directories to ignore.
func defaultIgnoreDirs() map[string]struct{} {
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

// scannerConfig holds scanner configuration.
type scannerConfig struct {
	root       string
	language   Language
	ignoreDirs map[string]struct{}
	maxBytes   int64
}

// scanner discovers files for processing.
type scanner struct {
	cfg scannerConfig
}

// newScanner creates a new scanner with the given configuration.
func newScanner(cfg scannerConfig) *scanner {
	if cfg.ignoreDirs == nil {
		cfg.ignoreDirs = defaultIgnoreDirs()
	}
	return &scanner{cfg: cfg}
}

// collect finds all matching files and returns them as FileJobs.
func (s *scanner) collect() ([]FileJob, error) {
	absRoot, err := filepath.Abs(s.cfg.root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}

	var jobs []FileJob
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

		if s.cfg.maxBytes > 0 {
			info, err := d.Info()
			if err != nil {
				// Skip files we can't stat
				return nil
			}
			if info.Size() > s.cfg.maxBytes {
				return nil
			}
		}

		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			rel = path
		}

		jobs = append(jobs, FileJob{
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

// collectSingle returns a single file as a FileJob.
func (s *scanner) collectSingle(filePath string) (FileJob, error) {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return FileJob{}, fmt.Errorf("resolve path: %w", err)
	}

	return FileJob{
		AbsPath:     absPath,
		DisplayPath: filepath.Base(absPath),
	}, nil
}

func (s *scanner) shouldIgnoreDir(name string) bool {
	_, ok := s.cfg.ignoreDirs[name]
	return ok
}

func (s *scanner) isSupportedFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if ext == "" {
		return false
	}
	for _, e := range s.cfg.language.Extensions() {
		if ext == e {
			return true
		}
	}
	return false
}
