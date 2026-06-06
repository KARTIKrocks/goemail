GOLANGCI_LINT_VERSION := v2.11.4

.PHONY: all help setup deps test test-v vet lint lint-fix build bench fmt cover clean examples ci

all: fmt vet lint build test

## Show available targets
help:
	@echo "Available targets:"
	@echo "  all           - Format, vet, lint, test, build"
	@echo "  setup         - Install development tools"
	@echo "  deps          - Download module dependencies"
	@echo "  test          - Run tests with race detector"
	@echo "  test-v        - Run tests with verbose output"
	@echo "  vet           - Run go vet"
	@echo "  lint          - Run golangci-lint"
	@echo "  lint-fix      - Run golangci-lint with --fix to auto-apply fixes"
	@echo "  build         - Build all packages"
	@echo "  bench         - Run benchmarks"
	@echo "  fmt           - Format code"
	@echo "  cover         - Run tests with coverage report"
	@echo "  clean         - Remove build artifacts"
	@echo "  examples      - Build all examples"
	@echo "  ci            - Run CI pipeline (vet, lint, test, build)"

## Install development tools (skips if already present)
setup:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	}
	@command -v goimports >/dev/null 2>&1 || { \
		echo "Installing goimports..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	}

## Download module dependencies
deps:
	go mod download

## Run all tests with race detector
test:
	go test -race -count=1 ./...

## Run tests with verbose output
test-v:
	go test -race -v -count=1 ./...

## Run go vet
vet:
	go vet ./...

## Run golangci-lint
lint: setup
	golangci-lint run ./...

## Run golangci-lint with --fix to auto-apply fixes where supported
lint-fix: setup
	golangci-lint run --fix ./...

## Build all packages
build:
	go build ./...

## Run benchmarks
bench:
	go test -bench=. -benchmem ./...

## Format code
fmt:
	gofmt -s -w .
	goimports -w .

## Run tests with coverage report (codecov-compatible output)
cover:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./...
	go tool cover -html=coverage.txt -o coverage.html
	@echo "Coverage report: coverage.html"

## Remove build artifacts
clean:
	rm -f coverage.txt coverage.html
	rm -rf dist/ build/ bin/

## Build all examples
examples:
	@echo "Building examples..."
	@mkdir -p bin
	go build -o bin/basic examples/basic/main.go
	go build -o bin/template examples/template/main.go
	go build -o bin/attachment examples/attachment/main.go
	go build -o bin/batch examples/batch/main.go
	@echo "Examples built in bin/"

## CI pipeline: vet, lint, test, build
ci: vet lint test build
	@echo "All CI checks passed!"
