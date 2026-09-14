<!-- The centred logo block opens the file, so there is no h1 on line 1. -->
<!-- markdownlint-disable-next-line MD041 -->
<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="website/static/img/logo-dark.svg">
    <img src="website/static/img/logo.svg" alt="goemail" width="104" height="104">
  </picture>
</p>

<h1 align="center">goemail</h1>

<p align="center">
  Production-ready email package for Go with SMTP support, templating,
  retries, connection pooling, DKIM signing, and provider adapters for
  SendGrid, Mailgun, and AWS SES.
</p>

<p align="center">
  <a href="https://pkg.go.dev/github.com/KARTIKrocks/goemail"><img src="https://pkg.go.dev/badge/github.com/KARTIKrocks/goemail.svg" alt="Go Reference"></a>
  <a href="https://goreportcard.com/report/github.com/KARTIKrocks/goemail"><img src="https://goreportcard.com/badge/github.com/KARTIKrocks/goemail" alt="Go Report Card"></a>
  <a href="go.mod"><img src="https://img.shields.io/github/go-mod/go-version/KARTIKrocks/goemail" alt="Go Version"></a>
  <a href="https://github.com/KARTIKrocks/goemail/actions/workflows/ci.yml"><img src="https://github.com/KARTIKrocks/goemail/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="https://github.com/KARTIKrocks/goemail/releases"><img src="https://img.shields.io/github/v/tag/KARTIKrocks/goemail" alt="GitHub tag"></a>
  <a href="https://opensource.org/licenses/MIT"><img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT"></a>
  <a href="https://codecov.io/gh/KARTIKrocks/goemail"><img src="https://codecov.io/gh/KARTIKrocks/goemail/branch/main/graph/badge.svg" alt="codecov"></a>
</p>

<p align="center">
  <b><a href="https://kartikrocks.github.io/goemail/">Documentation</a></b> ·
  <b><a href="https://pkg.go.dev/github.com/KARTIKrocks/goemail">API Reference</a></b> ·
  <b><a href="CHANGELOG.md">Changelog</a></b>
</p>

## Why goemail?

`net/smtp` gets you a connection and a `DATA` command. Everything past that —
the parts that turn "I can send one email" into "I can run this in
production" — is what goemail provides:

| Capability | goemail | Raw `net/smtp` |
| --- | --- | --- |
| Retry with exponential backoff | ✓ | You build it |
| Connection pooling for high throughput | ✓ | You build it |
| Rate limiting | ✓ | You build it |
| DKIM signing (RSA-SHA256 / Ed25519-SHA256) | ✓ | You build it |
| HTML templates, attachments, batch sending | ✓ | You build it |
| Provider HTTP adapters (SendGrid, Mailgun, AWS SES) | ✓ | You build it |
| Middleware (logging, metrics, hooks, recovery) | ✓ | You build it |
| Header injection protection, address validation | ✓ | You build it |
| Async sending with a buffered worker queue | ✓ | You build it |

goemail isn't a replacement for `net/smtp` — the default `Sender` is built on
top of it. It's the reliability and delivery layer around raw SMTP that most
services sending email end up writing themselves, packaged once and kept
production-safe by default.

## ✨ Features

- 📤 **SMTP Support** - TLS/STARTTLS, authentication
- 📝 **Templating** - Go templates for HTML and plain text emails
- 📎 **Attachments** - Send files with proper MIME encoding
- 🔄 **Retry Logic** - Exponential backoff with configurable attempts
- ⚡ **Rate Limiting** - Built-in rate limiting to prevent overwhelming servers
- 🔍 **Logging Interface** - Bring your own logger (slog, zap, logrus, etc.)
- 🧪 **Testing** - Mock sender for easy testing
- 🎯 **Builder API** - Fluent, chainable API for constructing emails
- 🔒 **Security** - Email header injection protection, address validation
- 🎨 **Batch Sending** - Send multiple emails concurrently with limits
- 🔗 **Connection Pooling** - Reuse SMTP connections for high-throughput sending
- 🔌 **Middleware Pipeline** - Composable middleware for logging, metrics, recovery, and hooks
- 🔀 **Async Sending** - Background queue worker with configurable workers and buffer
- 🌐 **Provider Adapters** - SendGrid, Mailgun, and AWS SES via HTTP APIs (providers modules)
- ✍️ **DKIM Signing** - Sign outgoing emails with RSA-SHA256 or Ed25519-SHA256 (RFC 6376/8463)
- 📊 **Context Support** - Full context.Context integration for timeouts and cancellation

## 📦 Installation

```bash
go get github.com/KARTIKrocks/goemail
```

## 🚀 Quick Start

### Basic Email

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/KARTIKrocks/goemail"
)

func main() {
    // Configure SMTP
    config := email.SMTPConfig{
        Host:     "smtp.gmail.com",
        Port:     587,
        Username: "your-email@gmail.com",
        Password: "your-app-password",
        From:     "your-email@gmail.com",
        UseTLS:   true,
    }

    // Create sender and mailer
    sender, err := email.NewSMTPSender(config)
    if err != nil {
        log.Fatal(err)
    }
    mailer := email.NewMailer(sender, config.From)
    defer mailer.Close()

    // Send email
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    err = mailer.Send(ctx,
        []string{"recipient@example.com"},
        "Hello!",
        "This is a test email.",
    )

    if err != nil {
        log.Fatal(err)
    }
}
```

### HTML Email with Template

```go
// Create template
tmpl := email.NewTemplate("welcome")
tmpl.SetSubject("Welcome {{.Name}}!")

tmpl.SetHTMLTemplate(`
<!DOCTYPE html>
<html>
<body>
    <h1>Hello {{.Name}}!</h1>
    <p>Welcome to our service. Click below to get started:</p>
    <a href="{{.VerifyLink}}">Verify Email</a>
</body>
</html>
`)

// Register template
mailer.RegisterTemplate("welcome", tmpl)

// Send using template
data := map[string]any{
    "Name":       "John Doe",
    "VerifyLink": "https://example.com/verify/abc123",
}

err := mailer.SendTemplate(ctx, []string{"john@example.com"}, "welcome", data)
```

### With Attachments

```go
// Read file
pdfData, _ := os.ReadFile("invoice.pdf")

email := email.NewEmail().
    SetFrom("billing@example.com").
    AddTo("customer@example.com").
    SetSubject("Your Invoice").
    SetBody("Please find your invoice attached.").
    AddAttachment("invoice.pdf", "application/pdf", pdfData)

err := mailer.SendEmail(ctx, email)
```

## 📖 Documentation

Full guides live at **[kartikrocks.github.io/goemail](https://kartikrocks.github.io/goemail/)**:

| Guide | Covers |
| --- | --- |
| [Getting Started](https://kartikrocks.github.io/goemail/docs/getting-started) | Install and send your first email |
| [SMTP & Configuration](https://kartikrocks.github.io/goemail/docs/smtp) | `SMTPConfig`, provider relay settings (Gmail, SendGrid, SES, Mailgun), connection pooling |
| [Mailer](https://kartikrocks.github.io/goemail/docs/mailer) | `Send`, `SendHTML`, `SendTemplate`, batch sending, `Close` |
| [Email Builder](https://kartikrocks.github.io/goemail/docs/email-builder) | The fluent `Email` builder API |
| [Templates](https://kartikrocks.github.io/goemail/docs/templates) | Registering and rendering HTML/text templates |
| [Middleware](https://kartikrocks.github.io/goemail/docs/middleware) | Logging, metrics, recovery, and hooks middleware |
| [Async Sending](https://kartikrocks.github.io/goemail/docs/async) | Buffered worker queue for non-blocking sends |
| [Reliability](https://kartikrocks.github.io/goemail/docs/reliability) | Retry logic, rate limiting, context timeouts |
| [DKIM Signing](https://kartikrocks.github.io/goemail/docs/dkim) | RSA-SHA256 / Ed25519-SHA256 signing (RFC 6376/8463) |
| [Provider Adapters](https://kartikrocks.github.io/goemail/docs/providers) | SendGrid, Mailgun, and AWS SES via HTTP APIs, plus the OpenTelemetry tracing middleware |
| [Webhooks](https://kartikrocks.github.io/goemail/docs/webhooks) | Parsing provider delivery-event webhooks |
| [Metrics](https://kartikrocks.github.io/goemail/docs/metrics) | The `MetricsCollector` interface and a worked Prometheus example |
| [Logging](https://kartikrocks.github.io/goemail/docs/logging) | Bringing your own logger |
| [Security](https://kartikrocks.github.io/goemail/docs/security) | Credential handling, header injection protection, address validation, App Passwords |
| [Testing](https://kartikrocks.github.io/goemail/docs/testing) | Unit-testing code that sends email with `MockSender` |

Exact type signatures are generated from source on
[pkg.go.dev](https://pkg.go.dev/github.com/KARTIKrocks/goemail).

Runnable programs are in [`examples/`](examples/) — basic, template,
attachment, batch, pool, middleware, and testing.

## 🔒 Security

goemail signs and sends outbound mail with credentials and key material you
provide, so header/CRLF injection, credential handling, and DKIM signing are
treated as part of the API, not an afterthought. `v0.2.0` ([CHANGELOG.md](CHANGELOG.md))
enforced a minimum of TLS 1.2 on STARTTLS connections and closed several
CRLF/header-injection gaps that `v0.1.0` did not catch.

Every push and pull request to `main` is scanned by
[CodeQL](https://github.com/KARTIKrocks/goemail/actions/workflows/codeql.yml),
with a full re-scan weekly to catch newly published query patterns against
unchanged code. `govulncheck` gates every merge on advisories that are
reachable from this code's call graph. Both run separately against each of
the five modules — the root package and the four `providers/*` sub-modules —
because a scan started from the root stops at the nested `go.mod` boundaries
and would miss the providers' own dependency trees. Dependabot tracks updates
across all five modules, the docs site, and the GitHub Actions themselves.

See [SECURITY.md](SECURITY.md) for supported versions, what is in scope, and
how to report a vulnerability privately.

## 📝 License

[MIT](LICENSE)

## 🤝 Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.
