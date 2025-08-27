# go-output v2 Documentation

This directory contains comprehensive documentation for go-output v2.

## Getting Started

- **[GETTING-STARTED.md](GETTING-STARTED.md)** - Quick introduction to v2 features and concepts
- **[DOCUMENTATION.md](DOCUMENTATION.md)** - Complete library documentation and API reference

## Migration from v1

- **[MIGRATION.md](MIGRATION.md)** - Complete step-by-step migration instructions
- **[V1_V2_MIGRATION_EXAMPLES.md](V1_V2_MIGRATION_EXAMPLES.md)** - Real before/after code examples
- **[BREAKING_CHANGES.md](BREAKING_CHANGES.md)** - Detailed list of all breaking changes
- **[MIGRATION_QUICK_REFERENCE.md](MIGRATION_QUICK_REFERENCE.md)** - Common patterns lookup table

## API Reference

- **[API.md](API.md)** - Comprehensive interface reference with examples

## Examples

See the [../examples/](../examples/) directory for working code examples demonstrating all major features.

## Development

### Quick Start for Contributors

```bash
# Clone and set up the project (from the repository root)
# Note: Makefile is in the repository root, not in v2/

# Run the pre-commit check (format, lint, test)
make check

# Run only unit tests
make test

# Run integration tests (if needed)  
make test-integration

# Generate coverage report
make test-coverage
```

### Documentation

- **[../CLAUDE.md](../CLAUDE.md)** - Complete development guide including:
  - Makefile targets and commands
  - Testing philosophy and organization  
  - Code architecture and patterns
  - Modernization details (Go 1.24+ features)

### Testing Strategy

The v2 codebase follows modern Go 2025 best practices:

- **Map-based Table Tests**: Improved test isolation using `map[string]struct` patterns
- **Integration Test Separation**: Uses `INTEGRATION=1` environment variable
- **File Organization**: Large test files split by logical functionality for maintainability
- **Test Conversion Tools**: Available in `test-conversion/` directory for modernizing test patterns

Run `make help` to see all available development commands.