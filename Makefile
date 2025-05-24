# Variables
BINARY_NAME=lnk
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME)"

# Go related variables
GOBASE=$(shell pwd)
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

# Colors for pretty output
RED=\033[0;31m
GREEN=\033[0;32m
YELLOW=\033[0;33m
BLUE=\033[0;34m
NC=\033[0m # No Color

.PHONY: help build test clean install uninstall fmt lint vet tidy run dev cross-compile release goreleaser-check goreleaser-snapshot

## help: Show this help message
help:
	@echo "$(BLUE)Lnk CLI - Available targets:$(NC)"
	@echo ""
	@echo "$(GREEN)Development:$(NC)"
	@echo "  build       Build the binary"
	@echo "  test        Run tests"
	@echo "  test-v      Run tests with verbose output"
	@echo "  test-cover  Run tests with coverage"
	@echo "  run         Run the application"
	@echo "  dev         Development mode with file watching"
	@echo ""
	@echo "$(GREEN)Code Quality:$(NC)"
	@echo "  fmt         Format Go code"
	@echo "  lint        Run golangci-lint"
	@echo "  vet         Run go vet"
	@echo "  tidy        Tidy Go modules"
	@echo "  check       Run all quality checks (fmt, vet, lint, test)"
	@echo ""
	@echo "$(GREEN)Installation:$(NC)"
	@echo "  install     Install binary to /usr/local/bin"
	@echo "  uninstall   Remove binary from /usr/local/bin"
	@echo ""
	@echo "$(GREEN)Release:$(NC)"
	@echo "  cross-compile       Build for multiple platforms (legacy)"
	@echo "  release             Create release builds (legacy)"
	@echo "  goreleaser-check    Validate .goreleaser.yml config"
	@echo "  goreleaser-snapshot Build snapshot release with GoReleaser"
	@echo ""
	@echo "$(GREEN)Utilities:$(NC)"
	@echo "  clean       Clean build artifacts"
	@echo "  deps        Install development dependencies"

## build: Build the binary
build:
	@echo "$(BLUE)Building $(BINARY_NAME)...$(NC)"
	@go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "$(GREEN)✓ Build complete: $(BINARY_NAME)$(NC)"

## test: Run tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	@go test ./test
	@echo "$(GREEN)✓ Tests passed$(NC)"

## test-v: Run tests with verbose output
test-v:
	@echo "$(BLUE)Running tests (verbose)...$(NC)"
	@go test -v ./test

## test-cover: Run tests with coverage
test-cover:
	@echo "$(BLUE)Running tests with coverage...$(NC)"
	@go test -v -cover ./test
	@go test -coverprofile=coverage.out ./test
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)✓ Coverage report generated: coverage.html$(NC)"

## run: Run the application
run: build
	@echo "$(BLUE)Running $(BINARY_NAME)...$(NC)"
	@./$(BINARY_NAME)

## dev: Development mode with file watching (requires entr)
dev:
	@echo "$(YELLOW)Development mode - watching for changes...$(NC)"
	@echo "$(YELLOW)Install 'entr' if not available: brew install entr$(NC)"
	@find . -name "*.go" | entr -r make run

## fmt: Format Go code
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)✓ Code formatted$(NC)"

## lint: Run golangci-lint
lint:
	@echo "$(BLUE)Running linter...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
		echo "$(GREEN)✓ Linting complete$(NC)"; \
	else \
		echo "$(YELLOW)⚠ golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest$(NC)"; \
	fi

## vet: Run go vet
vet:
	@echo "$(BLUE)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)✓ Vet check passed$(NC)"

## tidy: Tidy Go modules
tidy:
	@echo "$(BLUE)Tidying modules...$(NC)"
	@go mod tidy
	@echo "$(GREEN)✓ Modules tidied$(NC)"

## check: Run all quality checks
check: fmt vet lint test
	@echo "$(GREEN)✓ All quality checks passed$(NC)"

## install: Install binary to /usr/local/bin
install: build
	@echo "$(BLUE)Installing $(BINARY_NAME) to /usr/local/bin...$(NC)"
	@sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "$(GREEN)✓ $(BINARY_NAME) installed$(NC)"

## uninstall: Remove binary from /usr/local/bin
uninstall:
	@echo "$(BLUE)Uninstalling $(BINARY_NAME)...$(NC)"
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(GREEN)✓ $(BINARY_NAME) uninstalled$(NC)"

## cross-compile: Build for multiple platforms
cross-compile: clean
	@echo "$(BLUE)Cross-compiling for multiple platforms...$(NC)"
	@mkdir -p dist
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64 .
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64 .
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64 .
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe .
	@echo "$(GREEN)✓ Cross-compilation complete. Binaries in dist/$(NC)"

## release: Create release builds with checksums
release: cross-compile
	@echo "$(BLUE)Creating release artifacts...$(NC)"
	@cd dist && sha256sum * > checksums.txt
	@echo "$(GREEN)✓ Release artifacts created in dist/$(NC)"

## clean: Clean build artifacts
clean:
	@echo "$(BLUE)Cleaning...$(NC)"
	@rm -f $(BINARY_NAME)
	@rm -rf dist/
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)✓ Clean complete$(NC)"

## deps: Install development dependencies
deps:
	@echo "$(BLUE)Installing development dependencies...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@if ! command -v goreleaser >/dev/null 2>&1; then \
		echo "$(BLUE)Installing GoReleaser...$(NC)"; \
		go install github.com/goreleaser/goreleaser@latest; \
	fi
	@echo "$(GREEN)✓ Dependencies installed$(NC)"

## goreleaser-check: Validate GoReleaser configuration
goreleaser-check:
	@echo "$(BLUE)Validating GoReleaser configuration...$(NC)"
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser check; \
		echo "$(GREEN)✓ GoReleaser configuration is valid$(NC)"; \
	else \
		echo "$(YELLOW)⚠ GoReleaser not found. Install with: make deps$(NC)"; \
	fi

## goreleaser-snapshot: Build snapshot release with GoReleaser
goreleaser-snapshot: goreleaser-check
	@echo "$(BLUE)Building snapshot release with GoReleaser...$(NC)"
	@goreleaser build --snapshot --clean
	@echo "$(GREEN)✓ Snapshot release built in dist/$(NC)"

# Default target
all: check build 