# Contribution Guide

go-output is a Go library that provides a unified interface to convert your structured data into various output formats including JSON, YAML, CSV, HTML, console tables, Markdown, DOT graphs, Mermaid diagrams, and Draw.io files.

## Project layout

- most code is in the format package.
- more complicated outputs (drawio and mermaid) have their own packages.

## Documentation

📖 **[Complete Documentation](DOCUMENTATION.md)** - Comprehensive guide covering all features, configuration options, and API reference

🚀 **[Getting Started Guide](GETTING_STARTED.md)** - Quick introduction and setup instructions

💡 **[Examples](examples/)** - Working code examples demonstrating all features

## Dependencies

This library uses several external packages:
- [go-pretty](https://github.com/jedib0t/go-pretty) - Table formatting and styling
- [dot](https://github.com/emicklei/dot) - DOT graph generation
- [aws-sdk-go-v2](https://github.com/aws/aws-sdk-go-v2) - S3 integration
- [yaml.v3](https://gopkg.in/yaml.v3) - YAML processing
- [slug](https://github.com/gosimple/slug) - URL-safe string generation
- [color](https://github.com/fatih/color) - Terminal colors

## Local validation

Before opening a pull request run the following commands:

1. `gofmt`
2. `go test ./... -v`
3. `golangci-lint run`
4. Optionally `go build -o fog` to confirm the project builds.

## Code Style

- Use `gofmt` for formatting
- Follow Go best practices and maintain existing code style
- Follow standard Go naming conventions
- Add comments for exported functions and types
- Keep functions focused and testable

## Test instructions

1. Add or update tests for any code you change, even if nobody asked.
2. Tests should be complete and cover both failure and success states.
3. Tests should NOT recreate functions from the files that are being tested. Instead, the original function can be updated to make it possible to provide mock objects.
4. Include clear documentation of what the tests cover

## Code changes

1. If there are existing comments that are intended to clarify the code, leave these intact or update them accordingly. Do not delete the comments unless the related code is deleted.
2. Add comments where they add value to understanding the code. Ensure these comments are clear aid in this understanding. The comments should explain why the code does what it does, not how it does it.
3. Prefer simplicity and easy to understand code over complex solutions.
4. Try to make changes as localised as possible, but functions that are likely useful in multiple places should go in a helper file in the same directory.
5. If the changes impact the way existing functionality works, this should be reflected in the README and if required in the docs folder.
6. Documentation is aimed at aiding understanding. Don't try to be too concise.

## New functionality

1. If new functionality is created, ensure that the README file is updated to include this.
2. Documentation is aimed at aiding understanding. Don't try to be too concise.

## Pull request requirements

- PR titles should clarify if they're bug fixes, features, refactors, or documentation based in the format `[bug|feature|refactor|doc] <Title>` and should reference the relevant issue when applicable.

## Configuration examples

See [`example-fog.yaml`](example-fog.yaml) for an annotated example configuration and [`fog.yaml`](fog.yaml) for an example used by tests.