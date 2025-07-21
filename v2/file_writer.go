package output

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileWriter writes rendered output to files using a pattern
type FileWriter struct {
	baseWriter
	dir        string            // Base directory for files
	pattern    string            // e.g., "report-{format}.{ext}"
	extensions map[string]string // format to extension mapping
	mu         sync.Mutex        // For concurrent access protection
}

// NewFileWriter creates a new FileWriter with the specified directory and pattern
func NewFileWriter(dir, pattern string) (*FileWriter, error) {
	// Ensure directory exists and is accessible
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory %q: %w", dir, err)
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %q: %w", absDir, err)
	}

	// Verify it's a directory
	info, err := os.Stat(absDir)
	if err != nil {
		return nil, fmt.Errorf("failed to stat directory %q: %w", absDir, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %q is not a directory", absDir)
	}

	// Default pattern if not specified
	if pattern == "" {
		pattern = "output-{format}.{ext}"
	}

	return &FileWriter{
		baseWriter: baseWriter{name: "file"},
		dir:        absDir,
		pattern:    pattern,
		extensions: defaultExtensions(),
	}, nil
}

// Write implements the Writer interface
func (fw *FileWriter) Write(ctx context.Context, format string, data []byte) (returnErr error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return fw.wrapError(format, ctx.Err())
	default:
	}

	// Validate input
	if err := fw.validateInput(format, data); err != nil {
		return err
	}

	// Generate filename from pattern
	filename, err := fw.generateFilename(format)
	if err != nil {
		return fw.wrapError(format, err)
	}

	// Validate the filename for security
	if err := fw.validateFilename(filename); err != nil {
		return fw.wrapError(format, err)
	}

	// Write the file with proper locking
	fw.mu.Lock()
	defer fw.mu.Unlock()

	// Create the full file path
	fullPath := filepath.Join(fw.dir, filename)

	// Ensure the path is still within our directory (defense in depth)
	if !strings.HasPrefix(fullPath, fw.dir) {
		return fw.wrapError(format, fmt.Errorf("path escapes directory: %q", filename))
	}

	// Create any necessary subdirectories
	if dir := filepath.Dir(fullPath); dir != "." && dir != fw.dir {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fw.wrapError(format, fmt.Errorf("failed to create subdirectory: %w", err))
		}
	}

	// Use OpenFile with CREATE and TRUNCATE to overwrite existing files
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fw.wrapError(format, fmt.Errorf("failed to create file %q: %w", fullPath, err))
	}
	defer func() {
		if err := file.Close(); err != nil && returnErr == nil {
			returnErr = fw.wrapError(format, fmt.Errorf("failed to close file: %w", err))
		}
	}()

	// Write data
	_, err = file.Write(data)
	if err != nil {
		return fw.wrapError(format, fmt.Errorf("failed to write data: %w", err))
	}

	// Ensure data is flushed to disk
	if err := file.Sync(); err != nil {
		return fw.wrapError(format, fmt.Errorf("failed to sync file: %w", err))
	}

	return nil
}

// SetExtension sets a custom extension for a format
func (fw *FileWriter) SetExtension(format, ext string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.extensions[format] = ext
}

// GetDirectory returns the base directory for this writer
func (fw *FileWriter) GetDirectory() string {
	return fw.dir
}

// generateFilename generates a filename from the pattern
func (fw *FileWriter) generateFilename(format string) (string, error) {
	// Get extension for format
	ext, ok := fw.extensions[format]
	if !ok {
		ext = format // Use format as extension if not mapped
	}

	// Replace placeholders in pattern
	filename := fw.pattern
	filename = strings.ReplaceAll(filename, "{format}", format)
	filename = strings.ReplaceAll(filename, "{ext}", ext)

	// Clean the filename
	filename = filepath.Clean(filename)

	return filename, nil
}

// validateFilename ensures the filename is safe
func (fw *FileWriter) validateFilename(filename string) error {
	// Check for directory traversal attempts
	if strings.Contains(filename, "..") {
		return fmt.Errorf("invalid filename %q: contains '..'", filename)
	}

	// Check for absolute paths (Unix and Windows)
	if filepath.IsAbs(filename) {
		return fmt.Errorf("invalid filename %q: must be relative", filename)
	}

	// Check for Windows absolute paths (C:, D:, etc.)
	if len(filename) >= 2 && filename[1] == ':' {
		return fmt.Errorf("invalid filename %q: contains drive letter", filename)
	}

	// Check for UNC paths
	if strings.HasPrefix(filename, "\\\\") || strings.HasPrefix(filename, "//") {
		return fmt.Errorf("invalid filename %q: UNC paths not allowed", filename)
	}

	// Check for invalid characters (basic check)
	if strings.ContainsAny(filename, "\x00") {
		return fmt.Errorf("invalid filename %q: contains null bytes", filename)
	}

	// Ensure the filename doesn't start with a separator
	if strings.HasPrefix(filename, string(filepath.Separator)) {
		return fmt.Errorf("invalid filename %q: starts with separator", filename)
	}

	// Additional check for backslash on Unix systems
	if filepath.Separator != '\\' && strings.HasPrefix(filename, "\\") {
		return fmt.Errorf("invalid filename %q: starts with backslash", filename)
	}

	return nil
}

// defaultExtensions returns the default format to extension mappings
func defaultExtensions() map[string]string {
	return map[string]string{
		FormatJSON:     "json",
		FormatYAML:     "yaml",
		"yml":          "yml",
		FormatCSV:      "csv",
		FormatHTML:     "html",
		FormatTable:    "txt",
		FormatMarkdown: "md",
		FormatDOT:      "dot",
		FormatMermaid:  "mmd",
		FormatDrawIO:   "csv", // Draw.io CSV format
	}
}

// FileWriterOption configures a FileWriter
type FileWriterOption func(*FileWriter)

// WithExtensions sets custom format to extension mappings
func WithExtensions(extensions map[string]string) FileWriterOption {
	return func(fw *FileWriter) {
		for format, ext := range extensions {
			fw.extensions[format] = ext
		}
	}
}

// NewFileWriterWithOptions creates a FileWriter with options
func NewFileWriterWithOptions(dir, pattern string, opts ...FileWriterOption) (*FileWriter, error) {
	fw, err := NewFileWriter(dir, pattern)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		opt(fw)
	}

	return fw, nil
}
