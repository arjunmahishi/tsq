// Package output provides output formatting for codesitter.
package output

import (
	"encoding/json"
	"io"
	"os"
)

// Writer handles structured output.
type Writer struct {
	encoder *json.Encoder
	compact bool
}

// Config holds output configuration.
type Config struct {
	Compact bool
	Output  io.Writer
}

// New creates a new output Writer.
func New(cfg Config) *Writer {
	if cfg.Output == nil {
		cfg.Output = os.Stdout
	}

	enc := json.NewEncoder(cfg.Output)
	enc.SetEscapeHTML(false)
	if !cfg.Compact {
		enc.SetIndent("", "  ")
	}

	return &Writer{
		encoder: enc,
		compact: cfg.Compact,
	}
}

// Write outputs a value as JSON.
func (w *Writer) Write(v any) error {
	return w.encoder.Encode(v)
}

// WriteError writes an error message to stderr.
func WriteError(format string, args ...any) {
	enc := json.NewEncoder(os.Stderr)
	enc.Encode(map[string]any{
		"error": formatMessage(format, args...),
	})
}

func formatMessage(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	// Simple formatting without importing fmt to keep output package minimal
	result := format
	for _, arg := range args {
		if s, ok := arg.(string); ok {
			result += ": " + s
		}
	}
	return result
}
