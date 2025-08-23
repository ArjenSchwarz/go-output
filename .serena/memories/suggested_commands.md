# Suggested Commands for go-output Development

## Essential Development Commands

### Testing
```bash
# Run all tests (primary command)
go test ./...

# Run specific test
go test -v -run TestSchemaKeyOrderPreservation

# Run tests with coverage
go test -cover ./...
```

### Code Quality and Formatting
```bash
# Format code (converts interface{} to any automatically)
go fmt ./...

# Run linter (configured with .golangci.yml)
golangci-lint run
```

### Module Management
```bash
# Update dependencies
go mod tidy

# Verify dependencies
go mod verify
```

### Git Commands (Darwin system)
```bash
# Basic git operations
git status
git diff
git add .
git commit -m "message"
git branch --show-current
git log --oneline -5
```

### System Utilities (macOS/Darwin)
```bash
# File operations
ls -la
find . -name "*.go"
grep -r "pattern" .
cat filename
mv source dest
cp source dest
rm filename

# Directory navigation
pwd
cd path/to/directory
```

## Project-Specific Commands

### Working in v2 Directory
```bash
cd v2  # Primary development happens in v2/
go test ./...  # Test v2 specifically
```

### Running Examples
```bash
cd v2/examples
go run basic_usage/main.go
```

## Task Completion Commands
After finishing any Go development task:
1. `go fmt ./...` (required by CLAUDE.md)
2. `go test ./...` (required by CLAUDE.md)
3. `golangci-lint run` (if available)