.PHONY: help build test test-verbose test-coverage lint fmt format vet clean examples

# Default target
help:
	@echo "Available targets:"
	@echo "  make build         - Build the project"
	@echo "  make test          - Run tests"
	@echo "  make test-verbose  - Run tests with verbose output"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make lint          - Run linters (golangci-lint)"
	@echo "  make fmt           - Format code with gofmt"
	@echo "  make format        - Format code with golines (80 char lines)"
	@echo "  make vet           - Run go vet"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make examples      - Build all examples"

# Build the project
build:
	@echo "Building..."
	go build ./...

# Run tests
test:
	@echo "Running tests..."
	go test ./...

# Run tests with verbose output
test-verbose:
	@echo "Running tests (verbose)..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -cover ./...
	@echo ""
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run linters using golangci-lint
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found. Install it from https://golangci-lint.run/usage/install/"; \
		echo "Falling back to go vet..."; \
		$(MAKE) vet; \
	fi

# Format code with gofmt
fmt:
	@echo "Formatting code..."
	gofmt -s -w .

# Format code with golines (80 char lines, using goimports-reviser as base formatter)
format:
	@echo "Formatting code with golines..."
	@if command -v golines >/dev/null 2>&1; then \
		find . -name "*.go" -not -path "./vendor/*" -exec golines {} -w --base-formatter goimports-reviser --no-reformat-tags -m 80 -t 2 \;; \
	else \
		echo "golines not found. Install it with: go install github.com/segmentio/golines@latest"; \
		exit 1; \
	fi

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./...

# Build examples
examples:
	@echo "Building examples..."
	go build ./example/graphqldev
	go build ./example/realworld
	go build ./example/subscription

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f coverage.out coverage.html
	go clean ./...
