# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

`goemail` is a production-ready email library for Go. Module path is `github.com/KARTIKrocks/goemail`, but the **package name is `email`** (imported as `email "github.com/KARTIKrocks/goemail"`). The core library has minimal dependencies (only `golang.org/x/sync` and `golang.org/x/time`); anything heavier lives in a separate sub-module under `providers/`.

## Multi-module layout

This is a **multi-module repo with five independent modules**, each with its own `go.mod` (there is no `go.work`; the `Makefile` coordinates them):

- Root (`.`) — the core `email` package. Dependency-light.
- `providers/sendgrid` — SendGrid v3 Web API adapter (no SendGrid SDK).
- `providers/mailgun` — Mailgun HTTP API adapter.
- `providers/ses` — AWS SES v2 adapter (pulls in `aws-sdk-go-v2`).
- `providers/otelmail` — OpenTelemetry tracing middleware (pulls in the OTel SDK).

Providers are separate modules so that core users never transitively pull in the AWS SDK, OTel, etc. The `Makefile` `MODULES` variable drives every per-module loop — when adding a module, update `MODULES`/`SUB_MODULES` there.

Sub-modules depend on the root module by version, and the committed `go.work` overrides that with the working tree — so a breaking change to the root fails the provider tests instead of passing CI against the last published version. No `go.mod` here carries a `replace` directive. Use `GOWORK=off` to reproduce a consumer's build. Releasing is manual: tag the root first, bump each provider's `require` to that tag, then tag the providers (see CONTRIBUTING.md).

## Commands

Run from the repo root. The `Makefile` loops these across all modules.

- `make all` — tidy, fmt, vet, lint, build, test. **Run this to verify work before reporting completion.**
- `make test` — `go test -count=1 ./...` across all modules.
- `make test-race` — tests with the race detector.
- `make ci` — what CI runs: `fmt-check vet lint test-race`.
- `make lint` / `make fix` — golangci-lint; `fix` also runs fmt + `--fix`.
- `make fmt` — `gofmt -s` + `goimports`. `make fmt-check` fails on unformatted/unordered imports (CI gate).
- `make coverage` — merged coverage report across all modules.
- `make bench` — benchmarks.
- `make examples` — build the `examples/*` programs into `bin/`.
- `make setup` — install pinned `golangci-lint` and `goimports`.

Run a single test (cd into the relevant module first if testing a provider):

```bash
go test -run TestName -v ./...
go test -run TestName/subtest -race ./...
cd providers/ses && go test -run TestName ./...   # provider test
```

## Architecture

Everything composes around one interface (`email.go`):

```go
type Sender interface {
    Send(ctx context.Context, email *Email) error
    Close() error
}
```

- **`Email`** (`email.go`) is a message built via a fluent builder (`NewEmail().SetFrom().AddTo()...`). Builder methods accumulate errors internally; `Validate()` surfaces them and scrubs header values for CRLF injection. Prefer `AddHeader` over writing `Headers` directly — it canonicalizes the key and rejects CRLF.
- **`Mailer`** (`mailer.go`) is the high-level entry point wrapping a `Sender`. It holds named templates and deep-copies (`cloneEmail`) every outgoing message to avoid shared mutable state across concurrent sends. Safe for concurrent use.
- **`SMTPSender`** (`smtp.go`) is the default `Sender`: TLS/STARTTLS, auth, retry with exponential backoff, and `golang.org/x/time/rate` rate limiting. Configured via `SMTPConfig`.
- **`MockSender`** (`mock.go`) records sent emails for tests.

These are all `Sender`s or wrap a `Sender`, and they stack:

- **Middleware** (`middleware.go`) — `type Middleware func(Sender) Sender`. `Chain(sender, mw...)` composes them; the first in the list is outermost. Built-ins: logging, metrics, recovery, hooks. `otelmail` is a middleware living in its own module.
- **Pool** (`pool.go`) — reuses SMTP connections for high-throughput sending; enabled via `SMTPConfig.PoolSize`. Has its own lifecycle/idle/eviction tuning fields.
- **AsyncSender** (`async.go`) — wraps a `Sender` with a buffered queue and background worker goroutines for non-blocking delivery. Returns `ErrQueueFull` / `ErrQueueClosed`.

Supporting pieces:

- **`template.go`** — Go `html/template` + `text/template` rendering, registered on the `Mailer` by name.
- **`mime.go`** — MIME multipart assembly and attachment encoding.
- **`dkim.go`** — DKIM signing (RSA-SHA256 and Ed25519-SHA256, RFC 6376/8463); attached via `SMTPConfig.DKIM`.
- **`sanitize.go`** — HTML sanitizer with a configurable `Policy` (`EmailPolicy` is the default); strips dangerous elements/attributes/protocols.
- **`webhook.go`** — parses provider delivery-event webhooks (delivered/bounced/deferred/opened/clicked…).
- **`logger.go` / `logger_slog.go`** — pluggable `Logger` interface with an slog adapter; bring your own logger.

A new provider implements `email.Sender` in its own sub-module and is passed to `NewMailer`. A new cross-cutting concern is usually a `Middleware`.

## Conventions

- Go 1.26. Errors are sentinel `var Err... = errors.New("email: ...")`; check with `errors.Is`.
- Keep the **root module dependency-light** — if a feature needs a third-party SDK, put it in a `providers/` sub-module.
- The docs site lives on its own `website` branch (not tracked on `main`), in `goemail-website/` — React 19 + TypeScript + Vite + Tailwind v4 + Shiki. It's unrelated to the Go library; build/deploy with `npm run deploy` (builds and publishes `dist/` to the `gh-pages` branch). Ignore it for Go work.
