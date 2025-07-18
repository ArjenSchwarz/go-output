# Technology Stack

## Language & Version
- **Go 1.21+** - Modern Go with generics support
- Package name: `github.com/ArjenSchwarz/go-output`

## Key Dependencies
- `github.com/jedib0t/go-pretty/v6` - Table formatting and styling
- `github.com/emicklei/dot` - DOT graph generation
- `github.com/aws/aws-sdk-go-v2/service/s3` - S3 integration
- `gopkg.in/yaml.v3` - YAML processing
- `github.com/gosimple/slug` - URL-safe string generation
- `github.com/fatih/color` - Terminal colors

## Build System
Standard Go toolchain:
```bash
# Build
go build

# Test
go test ./...

# Run examples
cd examples && go run basic_usage.go

# Install as dependency
go get github.com/ArjenSchwarz/go-output
```

## Code Style & Standards
- Use `gofmt` for formatting
- Follow standard Go naming conventions
- Add comments for exported functions and types
- Keep functions focused and testable
- Use `log.Fatal()` for critical errors
- Maintain test coverage for new features

## Testing Approach
- Unit tests for core functionality
- Integration tests for complex features
- Test files follow `*_test.go` pattern
- Use table-driven tests where appropriate
- Mock external dependencies (S3, file system)

## Architecture Patterns
- Factory pattern for progress indicators
- Builder pattern for configuration
- Interface-based design for extensibility
- Concurrent-safe progress implementations