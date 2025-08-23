# Task Completion Requirements

## Mandatory Steps After Code Changes

When working with Go files in this project, ALWAYS follow these steps after making changes:

### 1. Format Code (Required)
```bash
go fmt ./...
```
- Automatically converts `interface{}` to `any`
- Ensures consistent formatting
- Required by project CLAUDE.md

### 2. Run Tests (Required)
```bash
go test ./...
```
- Ensures all functionality still works
- Required by project CLAUDE.md
- Must pass before considering task complete

### 3. Run Linter (Recommended)
```bash
golangci-lint run
```
- Configured with .golangci.yml
- Enforces modern Go practices
- Catches common issues

## Development Workflow

### Before Starting Work
1. Check current branch: `git branch --show-current`
2. Ensure you're in correct directory (usually v2/)
3. Run initial tests to ensure starting point is clean

### During Development
- Write tests first when implementing new features
- Ensure thread-safety for all operations
- Preserve key order for table operations
- Follow immutability patterns

### After Completing Tasks
1. **Always** run `go fmt ./...`
2. **Always** run `go test ./...` 
3. Fix any failing tests before proceeding
4. Run linter if available
5. Only commit when explicitly requested by user

## Quality Standards
- All tests must pass
- Code must be properly formatted
- No breaking changes to existing APIs
- Thread-safe operations required
- Key order preservation maintained
- Error handling with proper context

## Directory Structure
- Main v1 code in root directory
- **v2 development** in v2/ subdirectory
- Feature development in agents/{feature-name}/ directories
- Focus development efforts on v2 unless specified otherwise