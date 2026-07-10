GOLANGCI_LINT_VERSION := v2.12.2
GOIMPORTS_VERSION := v0.45.0

MODULES = . ./providers/mailgun ./providers/otelmail ./providers/sendgrid ./providers/ses
SUB_MODULES = ./providers/mailgun ./providers/otelmail ./providers/sendgrid ./providers/ses

.PHONY: all help setup deps ci test test-v test-race coverage lint lint-fix fix fmt fmt-check vet tidy build bench examples clean

all: tidy fmt vet lint build test

## Show available targets
help:
	@echo "Available targets:"
	@echo "  all           - Tidy, format, vet, lint, build, test (all modules)"
	@echo "  setup         - Install development tools"
	@echo "  deps          - Download module dependencies (all modules)"
	@echo "  ci            - CI pipeline (fmt-check, vet, lint, test-race)"
	@echo "  test          - Run tests across all modules"
	@echo "  test-v        - Run tests with verbose output (all modules)"
	@echo "  test-race     - Run tests with race detector (all modules)"
	@echo "  coverage      - Run tests with merged coverage report (all modules)"
	@echo "  vet           - Run go vet (all modules)"
	@echo "  lint          - Run golangci-lint (all modules)"
	@echo "  lint-fix      - Run golangci-lint with --fix (root module)"
	@echo "  fix           - fmt + lint-fix"
	@echo "  fmt           - Format code (gofmt -s + goimports)"
	@echo "  fmt-check     - Verify formatting without modifying files"
	@echo "  tidy          - Run go mod tidy (all modules)"
	@echo "  build         - Build all packages (all modules)"
	@echo "  bench         - Run benchmarks (all modules)"
	@echo "  examples      - Build all examples"
	@echo "  clean         - Remove build/coverage artifacts"

## Install development tools (skips if already present)
setup:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	}
	@command -v goimports >/dev/null 2>&1 || { \
		echo "Installing goimports $(GOIMPORTS_VERSION)..."; \
		go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION); \
	}

## Download module dependencies across all modules
deps:
	@for mod in $(MODULES); do \
		echo "==> Downloading deps $$mod"; \
		(cd $$mod && go mod download) || exit 1; \
	done

## CI: run lint and tests with race detector (used in CI pipelines)
ci: fmt-check vet lint test-race

## Build all modules
build:
	@for mod in $(MODULES); do \
		echo "==> Building $$mod"; \
		(cd $$mod && go build ./...) || exit 1; \
	done

## Run tests across all modules
test:
	@for mod in $(MODULES); do \
		echo "==> Testing $$mod"; \
		(cd $$mod && go test -count=1 ./...) || exit 1; \
	done

## Run tests with verbose output across all modules
test-v:
	@for mod in $(MODULES); do \
		echo "==> Testing (verbose) $$mod"; \
		(cd $$mod && go test -v -count=1 ./...) || exit 1; \
	done

## Run tests with race detector across all modules
test-race:
	@for mod in $(MODULES); do \
		echo "==> Testing (race) $$mod"; \
		(cd $$mod && go test -race -count=1 ./...) || exit 1; \
	done

## Run tests with coverage and generate a merged report
coverage:
	@go test -race -coverprofile=coverage-core.out -covermode=atomic ./...
	@cd providers/mailgun  && go test -race -coverprofile=../../coverage-mailgun.out  -covermode=atomic ./...
	@cd providers/otelmail && go test -race -coverprofile=../../coverage-otelmail.out -covermode=atomic ./...
	@cd providers/sendgrid && go test -race -coverprofile=../../coverage-sendgrid.out -covermode=atomic ./...
	@cd providers/ses      && go test -race -coverprofile=../../coverage-ses.out      -covermode=atomic ./...
	@cat coverage-core.out > coverage.out
	@tail -n +2 coverage-mailgun.out  >> coverage.out
	@tail -n +2 coverage-otelmail.out >> coverage.out
	@tail -n +2 coverage-sendgrid.out >> coverage.out
	@tail -n +2 coverage-ses.out      >> coverage.out
	@go tool cover -func=coverage.out | tail -1
	@echo "Full report: go tool cover -html=coverage.out"

## Run linter across all modules
lint: setup
	@for mod in $(MODULES); do \
		echo "==> Linting $$mod"; \
		(cd $$mod && golangci-lint run --timeout=5m ./...) || exit 1; \
	done

## Run golangci-lint with auto-fix (root module)
lint-fix: setup
	golangci-lint run --fix ./...

## Fix code formatting and linting issues
fix: fmt lint-fix

## Format code
fmt: setup
	@gofmt -s -w .
	@goimports -w .

## Check formatting without modifying files (used in CI)
fmt-check: setup
	@test -z "$$(gofmt -s -l . | tee /dev/stderr)" || { echo "Unformatted files found. Run 'make fmt'."; exit 1; }
	@test -z "$$(goimports -l . | tee /dev/stderr)" || { echo "Unordered imports found. Run 'make fmt'."; exit 1; }

## Run go vet across all modules
vet:
	@for mod in $(MODULES); do \
		echo "==> Vetting $$mod"; \
		(cd $$mod && go vet ./...) || exit 1; \
	done

## Run go mod tidy across all modules
tidy:
	@for mod in $(MODULES); do \
		echo "==> Tidying $$mod"; \
		(cd $$mod && go mod tidy) || exit 1; \
	done

## Run benchmarks across all modules
bench:
	@for mod in $(MODULES); do \
		echo "==> Benchmarking $$mod"; \
		(cd $$mod && go test -bench=. -benchmem ./...) || exit 1; \
	done

## Build all examples
examples:
	@echo "Building examples..."
	@mkdir -p bin
	go build -o bin/basic examples/basic/main.go
	go build -o bin/template examples/template/main.go
	go build -o bin/attachment examples/attachment/main.go
	go build -o bin/batch examples/batch/main.go
	@echo "Examples built in bin/"

## Remove build and coverage artifacts
clean:
	@rm -f coverage*.out coverage.txt coverage.html
	@rm -rf dist/ build/ bin/

