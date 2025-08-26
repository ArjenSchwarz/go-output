# Makefile for go-output v2 development
#
# This Makefile provides comprehensive developer tooling for the go-output v2 library.
# All commands operate on the v2 directory and example directories.
#
# Usage:
#   make help          - Show this help message
#   make check         - Run complete validation (fmt, lint, tests)
#   make test          - Run unit tests
#   make test-all      - Run all tests including integration tests

.PHONY: help
help: ## Show available make targets and their descriptions
	@echo "Available make targets:"
	@echo ""
	@echo "Testing targets:"
	@awk 'BEGIN {FS = ":.*##"} /^test.*:.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Code quality targets:"
	@awk 'BEGIN {FS = ":.*##"} /^(lint|fmt|modernize):.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Development targets:"
	@awk 'BEGIN {FS = ":.*##"} /^(mod-tidy|benchmark|clean):.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Composite targets:"
	@awk 'BEGIN {FS = ":.*##"} /^check:.*##/ { printf "  %-18s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@echo ""
	@echo "Usage: make <target>"
	@echo ""

# Default target
.DEFAULT_GOAL := help

# Testing targets
.PHONY: test
test: ## Run unit tests in v2 directory
	@echo "Running unit tests..."
	@cd v2 && go test ./...

.PHONY: test-integration
test-integration: ## Run integration tests with INTEGRATION=1 environment variable
	@echo "Running integration tests..."
	@cd v2 && INTEGRATION=1 go test ./...

.PHONY: test-all
test-all: ## Run both unit and integration tests
	@echo "Running all tests..."
	@cd v2 && go test ./...
	@echo "Running integration tests..."
	@cd v2 && INTEGRATION=1 go test ./...

.PHONY: test-coverage
test-coverage: ## Generate test coverage report and open in browser
	@echo "Generating coverage report..."
	@cd v2 && go test -coverprofile=coverage.out ./...
	@cd v2 && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: v2/coverage.html"
	@echo "Opening coverage report in browser..."
	@cd v2 && go tool cover -html=coverage.out

# Code quality targets
.PHONY: lint
lint: ## Run golangci-lint on v2 code
	@echo "Running golangci-lint..."
	@cd v2 && golangci-lint run

.PHONY: fmt
fmt: ## Format all Go code in v2 and example directories
	@echo "Formatting v2 code..."
	@cd v2 && go fmt ./...
	@echo "Formatting example directories..."
	@for dir in v2/examples/*/; do \
		if [ -f "$$dir/go.mod" ]; then \
			echo "  Formatting $$dir"; \
			(cd "$$dir" && go fmt ./...); \
		fi; \
	done

.PHONY: modernize
modernize: ## Apply modernize tool fixes to v2 and example directories
	@echo "Running modernize on v2 code..."
	@cd v2 && modernize -fix ./...
	@echo "Running modernize on example directories..."
	@for dir in v2/examples/*/; do \
		if [ -f "$$dir/go.mod" ]; then \
			echo "  Modernizing $$dir"; \
			(cd "$$dir" && modernize -fix ./...); \
		fi; \
	done
	@echo "Formatting after modernization..."
	@$(MAKE) fmt

# Development utility targets
.PHONY: mod-tidy
mod-tidy: ## Run go mod tidy on v2 and example directories
	@echo "Running go mod tidy on v2..."
	@cd v2 && go mod tidy
	@echo "Running go mod tidy on example directories..."
	@for dir in v2/examples/*/; do \
		if [ -f "$$dir/go.mod" ]; then \
			echo "  Tidying $$dir"; \
			(cd "$$dir" && go mod tidy); \
		fi; \
	done

.PHONY: benchmark
benchmark: ## Run performance benchmarks
	@echo "Running benchmarks..."
	@cd v2 && go test -bench=. -benchmem ./...

.PHONY: clean
clean: ## Remove generated files and test caches
	@echo "Cleaning generated files..."
	@cd v2 && rm -f coverage.out coverage.html
	@cd v2 && go clean -testcache
	@echo "Cleaning example directories..."
	@for dir in v2/examples/*/; do \
		if [ -f "$$dir/go.mod" ]; then \
			echo "  Cleaning $$dir"; \
			(cd "$$dir" && go clean -testcache); \
		fi; \
	done

# Composite targets
.PHONY: check
check: fmt lint test ## Run complete validation: format, lint, and test
	@echo "All checks completed successfully!"