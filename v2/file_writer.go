package output

import (
	"bytes"
	"context"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// FileWriter writes rendered output to files using a pattern
//
// HTML Rendering Mode Selection:
// When appending HTML content in append mode, callers should:
// 1. For new files: Render with HTML format (full page with <!-- go-output-append --> marker)
// 2. For existing files: Render with HTMLFragment format (no page structure, content only)
//
// The FileWriter will automatically detect file existence and append appropriately.
// The marker positioning ensures that fragments are inserted before the marker,
// preserving the original HTML structure and allowing multiple appends.
type FileWriter struct {
	baseWriter
	dir                  string            // Base directory for files
	pattern              string            // e.g., "report-{format}.{ext}"
	extensions           map[string]string // format to extension mapping
	allowAbsolute        bool              // Allow absolute paths in filenames
	mu                   sync.Mutex        // For concurrent access protection
	appendMode           bool              // Enable append mode instead of replace
	permissions          os.FileMode       // File permissions (default 0644)
	disallowUnsafeAppend bool              // Prevent appending to JSON/YAML
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
		baseWriter:    baseWriter{name: "file"},
		dir:           absDir,
		pattern:       pattern,
		extensions:    defaultExtensions(),
		allowAbsolute: false,
		permissions:   0644,
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
	var fullPath string
	if fw.allowAbsolute && filepath.IsAbs(filename) {
		fullPath = filename
	} else {
		fullPath = filepath.Join(fw.dir, filename)
		// Ensure the path is still within our directory (defense in depth)
		if !strings.HasPrefix(fullPath, fw.dir) {
			return fw.wrapError(format, fmt.Errorf("path escapes directory: %q", filename))
		}
	}

	// Create any necessary subdirectories
	if dir := filepath.Dir(fullPath); dir != "." && dir != fw.dir {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fw.wrapError(format, fmt.Errorf("failed to create subdirectory: %w", err))
		}
	}

	// Check if file exists
	fileExists := fw.fileExists(fullPath)

	// Handle append mode
	if fw.appendMode && fileExists {
		return fw.appendToFile(ctx, format, fullPath, data)
	}

	// Use OpenFile with CREATE and TRUNCATE to overwrite existing files
	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fw.permissions)
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
	// Always check for directory traversal attempts
	if strings.Contains(filename, "..") {
		return fmt.Errorf("invalid filename %q: contains '..'", filename)
	}

	// Check for invalid characters (basic check)
	if strings.ContainsAny(filename, "\x00") {
		return fmt.Errorf("invalid filename %q: contains null bytes", filename)
	}

	// If absolute paths are not allowed, perform additional validation
	if !fw.allowAbsolute {
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

		// Ensure the filename doesn't start with a separator
		if strings.HasPrefix(filename, string(filepath.Separator)) {
			return fmt.Errorf("invalid filename %q: starts with separator", filename)
		}

		// Additional check for backslash on Unix systems
		if filepath.Separator != '\\' && strings.HasPrefix(filename, "\\") {
			return fmt.Errorf("invalid filename %q: starts with backslash", filename)
		}
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

// fileExists checks if a file exists and is readable
func (fw *FileWriter) fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// appendToFile handles appending data to an existing file
func (fw *FileWriter) appendToFile(ctx context.Context, format string, fullPath string, data []byte) error {
	// Validate format extension if file exists
	if err := fw.validateFormatMatch(format, fullPath); err != nil {
		return fw.wrapError(format, err)
	}

	// Check if append is disabled for unsafe formats
	if fw.disallowUnsafeAppend && (format == FormatJSON || format == FormatYAML) {
		return fw.wrapError(format, fmt.Errorf("append to %s files is not allowed (unsafe format)", format))
	}

	// Format-specific append handling
	switch format {
	case FormatHTML:
		return fw.appendHTMLWithMarker(ctx, fullPath, data)
	case FormatCSV:
		return fw.appendCSVWithoutHeaders(ctx, fullPath, data)
	default:
		return fw.appendByteLevel(ctx, fullPath, data)
	}
}

// appendByteLevel performs byte-level appending to a file
func (fw *FileWriter) appendByteLevel(ctx context.Context, fullPath string, data []byte) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return fmt.Errorf("operation cancelled: %w", ctx.Err())
	default:
	}

	file, err := os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, fw.permissions)
	if err != nil {
		return fmt.Errorf("failed to open file for append %q: %w", fullPath, err)
	}
	defer func() {
		// Close file, errors are intentionally ignored as we already have the data
		_ = file.Close()
	}()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to append data: %w", err)
	}

	// Ensure data is flushed to disk
	if err := file.Sync(); err != nil {
		return fmt.Errorf("failed to sync file: %w", err)
	}

	return nil
}

// validateFormatMatch validates that the file extension matches the expected format
func (fw *FileWriter) validateFormatMatch(format string, fullPath string) error {
	// Get the file extension
	fileExt := filepath.Ext(fullPath)

	// If no extension, skip validation
	if fileExt == "" {
		return nil
	}

	// Remove leading dot from extension
	fileExt = fileExt[1:]

	// Get expected extension for format
	expectedExt, ok := fw.extensions[format]
	if !ok {
		expectedExt = format
	}

	// Check if they match
	if fileExt != expectedExt {
		return fmt.Errorf("file extension mismatch: expected %q but file has %q", expectedExt, fileExt)
	}

	return nil
}

// appendHTMLWithMarker appends HTML content to an existing file using atomic write-to-temp-and-rename pattern
func (fw *FileWriter) appendHTMLWithMarker(ctx context.Context, fullPath string, data []byte) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return fw.wrapError(FormatHTML, ctx.Err())
	default:
	}

	// Read existing file content
	existing, err := os.ReadFile(fullPath)
	if err != nil {
		return fw.wrapError(FormatHTML, fmt.Errorf("failed to read existing file: %w", err))
	}

	// Find marker using bytes.Index
	markerIndex := bytes.Index(existing, []byte(HTMLAppendMarker))
	if markerIndex == -1 {
		return fw.wrapError(FormatHTML, fmt.Errorf("HTML append marker not found in file: %s", fullPath))
	}

	// Create temp file in same directory with cryptographically random suffix
	tempFile, err := os.CreateTemp(filepath.Dir(fullPath), ".go-output-*.tmp")
	if err != nil {
		return fw.wrapError(FormatHTML, fmt.Errorf("failed to create temp file: %w", err))
	}
	tempPath := tempFile.Name()
	// Cleanup temp file on error using defer
	defer func() {
		// Attempt to remove temp file - ignore errors (file may have been renamed successfully)
		_ = os.Remove(tempPath)
	}()

	// Build new content: [before marker] + [new data] + [marker] + [after marker]
	var buf bytes.Buffer
	buf.Write(existing[:markerIndex])
	buf.Write(data)
	buf.WriteString(HTMLAppendMarker)
	buf.Write(existing[markerIndex+len(HTMLAppendMarker):])

	// Write to temp file
	if _, err := tempFile.Write(buf.Bytes()); err != nil {
		tempFile.Close()
		return fw.wrapError(FormatHTML, fmt.Errorf("failed to write temp file: %w", err))
	}

	// Ensure data is flushed to disk before rename (durability requirement)
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		return fw.wrapError(FormatHTML, fmt.Errorf("failed to sync temp file: %w", err))
	}
	tempFile.Close()

	// Atomic rename (atomic on same filesystem)
	if err := os.Rename(tempPath, fullPath); err != nil {
		return fw.wrapError(FormatHTML, fmt.Errorf("failed to rename temp file: %w", err))
	}

	return nil
}

// appendCSVWithoutHeaders appends CSV data to an existing file, stripping the header line
func (fw *FileWriter) appendCSVWithoutHeaders(ctx context.Context, fullPath string, data []byte) error {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return fw.wrapError(FormatCSV, ctx.Err())
	default:
	}

	// Normalize line endings (handle both LF and CRLF)
	data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))

	// Strip first line (header) from data
	lines := bytes.SplitN(data, []byte("\n"), 2)
	if len(lines) < 2 {
		// Only one line (or empty) - nothing to append after removing header
		return nil
	}

	dataWithoutHeader := lines[1]
	return fw.appendByteLevel(ctx, fullPath, dataWithoutHeader)
}

// FileWriterOption configures a FileWriter
type FileWriterOption func(*FileWriter)

// WithExtensions sets custom format to extension mappings
func WithExtensions(extensions map[string]string) FileWriterOption {
	return func(fw *FileWriter) {
		maps.Copy(fw.extensions, extensions)
	}
}

// WithAbsolutePaths allows absolute paths in filenames
func WithAbsolutePaths() FileWriterOption {
	return func(fw *FileWriter) {
		fw.allowAbsolute = true
	}
}

// WithAppendMode enables append mode for the FileWriter
func WithAppendMode() FileWriterOption {
	return func(fw *FileWriter) {
		fw.appendMode = true
	}
}

// WithPermissions sets custom file permissions (default 0644)
func WithPermissions(perm os.FileMode) FileWriterOption {
	return func(fw *FileWriter) {
		fw.permissions = perm
	}
}

// WithDisallowUnsafeAppend prevents appending to JSON/YAML files
func WithDisallowUnsafeAppend() FileWriterOption {
	return func(fw *FileWriter) {
		fw.disallowUnsafeAppend = true
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
