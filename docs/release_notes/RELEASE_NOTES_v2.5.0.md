# Release Notes: go-output v2.5.0

**Release Date:** October 30, 2025

## Overview

Version 2.5.0 introduces a critical thread-safety improvement by converting format variables to constructor functions. This change enables safe parallel test execution and eliminates race conditions that occurred when multiple goroutines shared format instances.

---

## ⚠️ Breaking Changes

### Format Variables Converted to Functions

**What Changed:**
All format variables (`JSON`, `YAML`, `HTML`, etc.) are now functions that return fresh `Format` instances with independent renderer instances.

**Required Action:**
Add parentheses `()` to all format references:

```go
// Before (v2.4.x)
output.WithFormat(output.JSON)
output.WithFormats(output.Table, output.CSV)

// After (v2.5.0)
output.WithFormat(output.JSON())
output.WithFormats(output.Table(), output.CSV())
```

**Why This Change:**
- In v2.4.x and earlier, format variables were shared global instances
- Multiple goroutines using the same format caused race conditions
- Tests with `t.Parallel()` would fail with "concurrent map write" errors
- Now each format call returns a fresh renderer instance, ensuring thread safety

**Migration Guide:**
See the [Migration from v2.4.x to v2.5.0](v2/docs/MIGRATION.md#migration-from-v24x-to-v250) section in MIGRATION.md for detailed instructions.

---

## ✨ New Features & Improvements

### Thread-Safe Format Functions

**Benefit:** Applications using go-output can now safely run tests in parallel without race conditions.

**Formats Converted to Functions:**
- Core formats: `JSON()`, `YAML()`, `CSV()`, `HTML()`, `HTMLFragment()`, `Table()`, `Markdown()`, `DOT()`, `Mermaid()`, `DrawIO()`
- Table styles: `TableDefault()`, `TableBold()`, `TableColoredBright()`, `TableLight()`, `TableRounded()`

**Example:**
```go
func TestMyFeature(t *testing.T) {
    t.Parallel() // ✅ Now safe!

    out := output.NewOutput(
        output.WithFormat(output.JSON()), // Each call gets fresh instance
        output.WithWriter(output.NewStdoutWriter()),
    )
    // ... test code ...
}
```

---

## 🐛 Bug Fixes

### Race Conditions in Parallel Tests
- **Fixed:** Applications can now run tests with `t.Parallel()` without encountering "concurrent map write" errors
- **Fixed:** Eliminated shared mutable state by ensuring renderer instances are never shared between goroutines

---

## 📚 Documentation Updates

- Updated all internal code, tests, examples, and documentation to use new format functions
- Updated README.md examples with new format function syntax
- Updated v2/docs/MIGRATION.md with detailed v2.5.0 migration guide
- Updated examples in v2/examples/append_mode/ directory

---

## 🔄 Migration Path

1. **Search and Replace:** Add `()` to all format variable references
2. **Verify:** Run your tests to ensure everything compiles
3. **Test:** Enable `t.Parallel()` in your test suite to verify thread safety

**Automated Migration:**
The change is straightforward - a simple search and replace pattern:
- `output.JSON` → `output.JSON()`
- `output.YAML` → `output.YAML()`
- And so on for all format types

---

## 📦 Installation

```bash
go get github.com/ArjenSchwarz/go-output/v2@v2.5.0
```

---

## 🙏 Contributors

This release addresses user-reported issues with parallel test execution and demonstrates our commitment to thread-safe, production-ready code.

---

## 📖 Additional Resources

- [CHANGELOG.md](CHANGELOG.md) - Complete version history
- [v2/docs/MIGRATION.md](v2/docs/MIGRATION.md) - Migration guide from v2.4.x to v2.5.0
- [README.md](README.md) - Quick start and feature overview
- [v2/examples/](v2/examples/) - Working code examples

---

## ⚠️ Important Notes

- This is a **breaking change** requiring code modifications
- All format references must be updated to use function calls
- The change is simple but affects every format usage in your code
- Benefits include thread safety and the ability to run tests in parallel
- No other functionality has changed - this is purely a thread-safety improvement
