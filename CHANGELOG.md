# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-04-19

Initial public release.

### Added

#### Core
- `Email` type with fluent builder API (`NewEmail`, `SetFrom`, `AddTo`, `AddCc`,
  `AddBcc`, `SetReplyTo`, `SetSubject`, `SetBody`, `SetHTMLBody`,
  `AddAttachment`, `AddHeader`, `Build`, `Validate`).
- `Sender` interface (`Send`, `Close`) as the core abstraction for all
  delivery backends.
- `Mailer` high-level wrapper with `Send`, `SendHTML`, `SendTemplate`,
  `SendEmail`, and `SendBatch` (concurrent delivery with configurable limit).
- `NewMailerWithOptions` with the `MailerOption` functional-options pattern
  and `WithMiddleware` helper.
- `Error` type exposing `Op`, `From`, `To`, and the wrapped underlying error;
  supports `errors.Is`/`errors.As`.
- Sentinel errors: `ErrNoRecipients`, `ErrNoSender`, `ErrNoSubject`,
  `ErrNoBody`, `ErrInvalidHeader`, `ErrPoolClosed`, `ErrPoolTimeout`,
  `ErrQueueFull`, `ErrQueueClosed`, `ErrPanicked`.
- Email address validation via `net/mail` on From, To, Cc, Bcc, and Reply-To.
- CRLF injection protection on all header keys and values, including Subject
  and custom headers.
- `AddHeader` canonicalizes keys with `textproto.CanonicalMIMEHeaderKey` to
  prevent duplicate-case headers.
- `AddAttachment` defensively copies the caller's byte slice.

#### SMTP
- `SMTPSender` with `SMTPConfig` (Host, Port, Username, Password, From, UseTLS,
  Timeout, MaxRetries, RetryDelay, RetryBackoff, RateLimit).
- STARTTLS support with `ServerName` set for hostname verification.
- Exponential backoff retry logic with a 5-minute per-attempt ceiling.
- Non-retry fast-path for deterministic failures (validation errors, context
  cancellation, missing recipients/sender/subject/body, and permanent SMTP
  5xx replies per RFC 5321).
- Token-bucket rate limiting via `golang.org/x/time/rate` (configurable or
  disabled with a negative value).
- Per-connection deadline applied to dial, STARTTLS, auth, MAIL/RCPT/DATA so a
  slow server cannot hang the send indefinitely.
- Graceful shutdown of the SMTP client on error paths via a single deferred
  `client.Close`.

#### Connection Pooling
- Optional SMTP connection pool activated by setting `PoolSize > 0`.
- Configurable `MaxIdleConns`, `PoolMaxLifetime`, `PoolMaxIdleTime`,
  `MaxMessages` (messages per connection before rotation), `PoolWaitTimeout`.
- Health-check probe (`RSET`) on every checkout and on connections received
  from the wait queue.
- LIFO idle-connection reuse and a FIFO waiter queue honoring `PoolWaitTimeout`
  as a real ceiling across health-check retries.
- Background cleaner goroutine that evicts expired and idle-timed-out
  connections.
- `Close()` drains idle connections and signals waiters with `ErrPoolClosed`.

#### Async Delivery
- `AsyncSender` wrapping any `Sender` with a buffered work queue and
  configurable worker goroutines.
- Functional options: `WithQueueSize`, `WithWorkers`, `WithAsyncLogger`,
  `WithErrorHandler`.
- `Send` (fire-and-forget) detaches caller cancellation via
  `context.WithoutCancel` while preserving request-scoped values.
- `SendWait` blocks until delivery completes or the context is cancelled.
- Graceful shutdown via `Close` (`sync.Once`): drains the queue, waits for
  workers, then closes the underlying sender.

#### Templating
- `Template` with `SetSubject`, `SetTextTemplate`, `SetHTMLTemplate`, and
  `Render(data)`. Subject uses `text/template`; HTML uses `html/template` with
  auto-escaping.
- Optional HTML sanitization on rendered output via `WithSanitization` and
  `WithSanitizationPolicy`.
- Template loaders: `LoadTemplateFromFile`, `LoadTemplateFromFS`,
  `LoadTemplatesFromDir`, `LoadTemplatesFromFS`. Merges files with the same
  base name (`welcome.html` + `welcome.txt` + `welcome.subject` → one
  `Template`).

#### Middleware Pipeline
- `Middleware` type and `Chain(sender, middlewares...)` helper (first middleware
  is outermost).
- `WithLogging(logger)` — logs send attempts, successes, and failures.
- `WithRecovery()` — converts panics into errors wrapping `ErrPanicked`.
- `WithHooks(SendHooks)` — user-defined `OnSend`, `OnSuccess`, `OnFailure`
  callbacks.
- `WithMetrics(collector)` — records send counters and latency via the
  `MetricsCollector` interface.
- `WithSanitization` / `WithSanitizationPolicy` — sanitizes `HTMLBody` before
  delivery as a safety net.

#### Webhooks
- Provider-agnostic `WebhookEvent` normalized type with `EventType`
  constants (`delivered`, `bounced`, `deferred`, `opened`, `clicked`,
  `complained`, `unsubscribed`, `dropped`).
- `WebhookParser` and `WebhookHandler` interfaces for provider-specific
  parsing and event handling.
- `WebhookReceiver` `http.Handler` with `WithWebhookLogger` and
  `WithEventFilter` options. Dispatches every event in a batch even when one
  fails (handlers must be idempotent).

#### DKIM Signing (RFC 6376 / RFC 8463)
- `DKIMConfig` with Domain, Selector, PrivateKey, optional `SignedHeaders`,
  `Expiration`, and per-section (header/body) canonicalization choice.
- `SignMessage(msg, config)` produces a DKIM-Signature header prepended to the
  message.
- `BuildRawMessage` and `BuildRawMessageWithDKIM` for raw-message providers
  (e.g. AWS SES).
- `ParseDKIMPrivateKey(pem)` accepts PKCS#8 (RSA or Ed25519) and PKCS#1 (RSA).
- Automatic signing of every SMTP send when `SMTPConfig.DKIM` is non-nil.
- Supports simple and relaxed canonicalization for both headers and body.
- RSA-SHA256 and Ed25519-SHA256 algorithms; `From` header always signed per
  RFC 6376 §5.4.

#### MIME
- Proper multipart encoding: `multipart/mixed` (with attachments),
  `multipart/alternative` (plain + HTML without attachments), and single-part
  fallbacks.
- Quoted-printable body encoding and base64 attachment encoding with
  76-character line wrapping per RFC 2045.
- RFC 2047 Q-encoding for non-ASCII subjects and display names.
- RFC 2231 `filename*=` encoding for non-ASCII attachment filenames.
- Filename sanitization strips CR/LF/NUL and path separators from
  `Content-Disposition`.
- Deterministic custom-header ordering for reproducible output.
- Case-insensitive `Message-ID` override; auto-generated `Message-ID` uses the
  sender's domain and a `crypto/rand` identifier.

#### HTML Sanitization
- `Policy` with builder methods `AllowElements`, `AllowAttributes`,
  `AllowGlobalAttributes`, `AllowURLProtocols`, `StripElements`.
- `EmailPolicy()` ships a sensible default allowlist covering common email
  clients (Gmail, Outlook, Apple Mail, Yahoo).
- `SanitizeHTML` and `SanitizeHTMLWithPolicy` entry points, plus
  `SanitizeFuncMap` for use inside `html/template`.
- URL-protocol allowlisting with HTML-entity decoding, WHATWG URL
  normalization (strip tab/LF/CR/NUL), and control-character stripping to
  catch `java&#9;script:` and similar obfuscations.
- CSS value filtering: rejects `expression`, `javascript:`, `vbscript:`,
  `-moz-binding`, `behavior:`, and `url(...)` references with disallowed
  protocols.

#### Logging
- `Logger` interface (`Debug`, `Info`, `Warn`, `Error`, `With`).
- `NoOpLogger` default implementation.
- `SlogLogger` adapter for the standard library `log/slog`.

#### Testing Support
- `MockSender` with `GetSentEmails`, `GetLastEmail`, `GetEmailCount`,
  `GetEmailsTo`, `GetEmailsBySubject`, `Reset`, and `SetSendFunc` for
  injecting error scenarios.

#### Provider Adapters (separate Go modules)
- `providers/sendgrid` — SendGrid v3 Web API (`/v3/mail/send`), handles
  personalizations, Cc/Bcc, Reply-To, headers, and base64 attachments.
- `providers/mailgun` — Mailgun v3 Messages API with multipart form upload;
  supports US and EU base URLs. Attachment `Content-Type` is preserved on
  each form part.
- `providers/ses` — AWS SES v2 using raw MIME messages (via
  `BuildRawMessage`), optional configuration set.
- `providers/otelmail` — OpenTelemetry tracing middleware that produces a
  client-kind span per send, with configurable tracer and span names.

#### Examples
- `examples/basic`, `examples/template`, `examples/attachment`,
  `examples/batch`, `examples/middleware`, `examples/pool`,
  `examples/testing`.

#### Tooling
- `Makefile` with `all`, `test`, `vet`, `lint`, `build`, `bench`, `fmt`,
  `cover`, `ci`, and `examples` targets.
- `golangci-lint` configuration.
- GitHub Actions CI, Codecov integration, editorconfig, and contribution
  guidelines.

### Requirements
- Go 1.26 or newer.

[Unreleased]: https://github.com/KARTIKrocks/goemail/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/KARTIKrocks/goemail/releases/tag/v0.1.0
