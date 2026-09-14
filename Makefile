GOLANGCI_LINT_VERSION := v2.13.2
GOIMPORTS_VERSION := v0.50.0
GOVULNCHECK_VERSION := v1.8.0

MODULES = . ./providers/mailgun ./providers/otelmail ./providers/sendgrid ./providers/ses
SUB_MODULES = ./providers/mailgun ./providers/otelmail ./providers/sendgrid ./providers/ses

# The Markdown linter. Versioned in website/package.json rather than pinned
# here, so Dependabot keeps it current along with the rest of the docs toolchain.
MARKDOWNLINT := website/node_modules/.bin/markdownlint-cli2

.PHONY: all help setup setup-golangci-lint setup-goimports setup-govulncheck deps ci test test-v test-race coverage lint lint-fix lint-docs lint-docs-fix fix fmt fmt-check vet tidy tidy-check vuln print-golangci-lint-version build bench examples clean

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
	@echo "  lint-docs     - Lint every Markdown file in the repo"
	@echo "  lint-docs-fix - Lint Markdown and apply automatic fixes"
	@echo "  fix           - fmt + lint-fix"
	@echo "  fmt           - Format code (gofmt -s + goimports)"
	@echo "  fmt-check     - Verify formatting without modifying files"
	@echo "  tidy          - Run go mod tidy (all modules)"
	@echo "  tidy-check    - Fail if go.mod/go.sum are not tidy (all modules)"
	@echo "  vuln          - Run govulncheck (all modules)"
	@echo "  build         - Build all packages (all modules)"
	@echo "  bench         - Run benchmarks (all modules)"
	@echo "  examples      - Build all examples"
	@echo "  clean         - Remove build/coverage artifacts"

## Install all development tools (skips each if already present)
setup: setup-golangci-lint setup-goimports setup-govulncheck

## Install golangci-lint, skipping if already present
setup-golangci-lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)..."; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	}

## Install goimports, skipping if already present
setup-goimports:
	@command -v goimports >/dev/null 2>&1 || { \
		echo "Installing goimports $(GOIMPORTS_VERSION)..."; \
		go install golang.org/x/tools/cmd/goimports@$(GOIMPORTS_VERSION); \
	}

## Install govulncheck, skipping if already present
setup-govulncheck:
	@command -v govulncheck >/dev/null 2>&1 || { \
		echo "Installing govulncheck $(GOVULNCHECK_VERSION)..."; \
		go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION); \
	}

## Download module dependencies across all modules
deps:
	@for mod in $(MODULES); do \
		echo "==> Downloading deps $$mod"; \
		(cd $$mod && go mod download) || exit 1; \
	done

## CI: run lint, vet, tests with race detector, and the vulnerability/tidy
## gates (used in CI pipelines)
ci: fmt-check vet lint test-race tidy-check vuln lint-docs

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
lint: setup-golangci-lint
	@for mod in $(MODULES); do \
		echo "==> Linting $$mod"; \
		(cd $$mod && golangci-lint run --timeout=5m ./...) || exit 1; \
	done

## Run golangci-lint with auto-fix (root module)
lint-fix: setup-golangci-lint
	golangci-lint run --fix ./...

## Lint every Markdown file in the repo — the docs site, the README, and the
## contributor/security policies. Config and rationale live in
## .markdownlint-cli2.jsonc. Needs Node; the binary comes from website/, which
## is the only npm project here.
lint-docs: $(MARKDOWNLINT)
	@website/node_modules/.bin/markdownlint-cli2

## Lint Markdown and apply the fixes it can make automatically.
lint-docs-fix: $(MARKDOWNLINT)
	@website/node_modules/.bin/markdownlint-cli2 --fix

$(MARKDOWNLINT):
	@echo "Installing website dependencies (needed for the Markdown linter)..."
	@cd website && npm ci

## Fix code formatting and linting issues
fix: fmt lint-fix

## Format code
fmt: setup-goimports
	@gofmt -s -w .
	@goimports -w .

## Check formatting without modifying files (used in CI)
fmt-check: setup-goimports
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

## Fail if any go.mod/go.sum is not tidy, without leaving the change behind.
## Suitable for CI, where a stale go.sum should block the merge.
tidy-check:
	@status=$$(git status --porcelain -- '*go.mod' '*go.sum'); \
	if [ -n "$$status" ]; then \
		echo "go.mod/go.sum already modified; commit or stash before running tidy-check"; \
		exit 1; \
	fi; \
	$(MAKE) --no-print-directory tidy; tidy_status=$$?; \
	changed=$$(git diff --name-only -- '*go.mod' '*go.sum'); \
	if [ $$tidy_status -ne 0 ]; then \
		[ -n "$$changed" ] && git checkout -- $$changed; \
		exit $$tidy_status; \
	fi; \
	if [ -n "$$changed" ]; then \
		echo "go.mod/go.sum are not tidy — run 'make tidy' and commit:"; \
		git diff --stat -- '*go.mod' '*go.sum'; \
		git checkout -- $$changed; \
		exit 1; \
	fi; \
	echo "all modules tidy"

## Scan all modules for known vulnerabilities, filtered to advisories the code
## actually reaches. Mirrors the `vuln` CI job, which gates merges.
##
## Needs network access — the advisory database is fetched on every run. This
## also scans the standard library of whichever Go toolchain you have
## installed, so it can fail locally on a green branch when your Go is a patch
## release behind the one CI pins. That is a real finding about your machine,
## not a false positive.
vuln: setup-govulncheck
	@for mod in $(MODULES); do \
		echo "==> Scanning $$mod"; \
		(cd $$mod && govulncheck ./...) || exit 1; \
	done

## Print the pinned linter version. CI resolves golangci-lint-action's version
## input from this rather than hardcoding a second copy of the number, so the
## workflow and this file cannot drift apart.
print-golangci-lint-version:
	@echo $(GOLANGCI_LINT_VERSION)

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
	go build -o bin/middleware examples/middleware/main.go
	go build -o bin/pool examples/pool/main.go
	@echo "Examples built in bin/"

## Remove build and coverage artifacts
clean:
	@rm -f coverage*.out coverage.txt coverage.html
	@rm -rf dist/ build/ bin/

