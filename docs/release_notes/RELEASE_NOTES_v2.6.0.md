# Release Notes: go-output v2.6.0

**Release Date:** November 7, 2025

## Overview

Version 2.6.0 adds support for writing output to standard error, enabling proper stream separation for CLI tools and applications that need to distinguish normal output from diagnostic information.

---

## ✨ New Features

### StderrWriter for Stream Separation

**What's New:**
A new `NewStderrWriter()` constructor that writes rendered output to standard error (stderr), complementing the existing `NewStdoutWriter()`.

**Why This Matters:**
Following Unix conventions, stderr should be used for errors, warnings, and diagnostic messages, while stdout is reserved for normal program output. This separation is critical for:
- **CLI Tools**: Distinguish error messages from data results
- **Streaming Applications**: Prevent diagnostics from polluting data output
- **Shell Pipelines**: Enable separate capture of output and errors
- **Logging Systems**: Route diagnostics differently from results

**Example:**

```go
// Normal data output goes to stdout
dataOutput := output.NewOutput(
    output.WithFormat(output.JSON()),
    output.WithWriter(output.NewStdoutWriter()),
)

// Errors and diagnostics go to stderr
errorOutput := output.NewOutput(
    output.WithFormat(output.JSON()),
    output.WithWriter(output.NewStderrWriter()),
)

ctx := context.Background()

// Render user data to stdout
if err := dataOutput.Render(ctx, userDataDoc); err != nil {
    log.Fatal(err)
}

// Render error summary to stderr
if err := errorOutput.Render(ctx, errorDoc); err != nil {
    log.Fatal(err)
}
```

**Shell Usage:**
```bash
# Capture stdout and stderr separately
myapp > output.json 2> errors.log

# Pipe stdout while preserving stderr
myapp | jq . 2> errors.log

# Discard errors but keep output
myapp 2>/dev/null > results.txt
```

**Architecture:**
- Follows same design as `StdoutWriter` for API consistency
- Thread-safe concurrent write operations with mutex protection
- Context cancellation support for graceful termination
- Automatic newline addition for consistent output formatting
- `SetWriter()` method for custom `io.Writer` injection (useful for testing)

---

## 🎯 Use Cases

### CLI Tools with Data Output

```go
// Success path: results to stdout
results := output.New().
    Table("users", userData, output.WithKeys("name", "email")).
    Build()

stdout := output.NewOutput(
    output.WithFormat(output.JSON()),
    output.WithWriter(output.NewStdoutWriter()),
)

// Error path: diagnostics to stderr
errors := output.New().
    Table("errors", errorData, output.WithKeys("timestamp", "message", "severity")).
    Build()

stderr := output.NewOutput(
    output.WithFormat(output.JSON()),
    output.WithWriter(output.NewStderrWriter()),
)
```

### Logging and Diagnostics

```go
// Application logs go to stderr
logger := output.NewOutput(
    output.WithFormat(output.Table()),
    output.WithWriter(output.NewStderrWriter()),
)

// Query results go to stdout
queryOutput := output.NewOutput(
    output.WithFormat(output.CSV()),
    output.WithWriter(output.NewStdoutWriter()),
)
```

### Progress and Status Messages

```go
// Progress updates on stderr (won't interfere with piped output)
progress := output.NewOutput(
    output.WithFormat(output.Table()),
    output.WithWriter(output.NewStderrWriter()),
    output.WithProgress(output.NewPrettyProgress()),
)

// Final results on stdout
results := output.NewOutput(
    output.WithFormat(output.JSON()),
    output.WithWriter(output.NewStdoutWriter()),
)
```

---

## 📚 Documentation Updates

- **API Reference**: Added comprehensive documentation in `v2/docs/API.md`
  - StderrWriter constructor and methods (lines 1013-1014)
  - Stream separation guidelines (lines 1026-1037)
  - Usage examples with stdout/stderr patterns (lines 2180-2204)
- **Comprehensive Test Suite**: Full test coverage including:
  - Basic write operations
  - Concurrent write safety
  - Context cancellation handling
  - Error scenarios
  - Custom writer injection for testing

---

## 🔄 Migration Path

**No Breaking Changes** - This is a purely additive release.

**To Use:**
Simply call `output.NewStderrWriter()` where you need stderr output:

```go
// Old approach (stdout only)
out := output.NewOutput(
    output.WithFormat(output.JSON()),
    output.WithWriter(output.NewStdoutWriter()),
)

// New approach (add stderr for diagnostics)
diagnostics := output.NewOutput(
    output.WithFormat(output.JSON()),
    output.WithWriter(output.NewStderrWriter()),
)
```

---

## 📦 Installation

```bash
go get github.com/ArjenSchwarz/go-output/v2@v2.6.0
```

---

## 🔧 Technical Details

### Thread Safety
- All write operations protected by mutex
- Safe for concurrent use from multiple goroutines
- Context cancellation checked before writes

### Testing
The `StderrWriter` includes comprehensive tests:
- `TestNewStderrWriter`: Constructor validation
- `TestStderrWriterWrite`: Write operations with multiple formats
- `TestStderrWriterConcurrency`: Concurrent write safety
- `TestStderrWriterContextCancellation`: Graceful cancellation
- `TestStderrWriterWriteError`: Error handling

### API Consistency
Matches the `StdoutWriter` interface:
- `Write(ctx context.Context, format string, data []byte) error`
- `SetWriter(w io.Writer)` for testing with custom writers

---

## 📖 Additional Resources

- [CHANGELOG.md](CHANGELOG.md) - Complete version history
- [v2/docs/API.md](v2/docs/API.md) - Full API reference with StderrWriter documentation
- [README.md](README.md) - Quick start and feature overview
- [v2/examples/](v2/examples/) - Working code examples

---

## 🙏 Contributors

This release addresses the need for proper stream separation in CLI applications, a common pattern in Unix-style tools. Thank you to the community for feedback on output stream handling!

---

## ⬆️ Upgrade Notes

- **No code changes required** for existing applications
- **Backward compatible** with all v2.x releases
- **Drop-in addition** - use StderrWriter when you need stderr output
- All existing functionality remains unchanged

---

## What's Next

Check out the complete go-output v2 feature set:
- **Multiple Output Formats**: JSON, YAML, CSV, HTML, Tables, Markdown, DOT, Mermaid, Draw.io
- **Per-Content Transformations**: Filter, sort, limit data at the content level
- **Append Mode**: Add to existing files with format-aware handling
- **S3 Support**: Write directly to AWS S3 with append capability
- **Progress Indicators**: Visual feedback for long-running operations
- **Thread-Safe**: All operations safe for concurrent use
