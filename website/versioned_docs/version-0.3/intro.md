---
id: intro
title: Overview
sidebar_label: Overview
description: A production-ready email package for Go with SMTP, templating, retries, connection pooling, async sending, middleware, DKIM signing, and provider adapters for SendGrid, Mailgun, and AWS SES.
slug: /
---

# goemail

Production-ready email package for Go with SMTP support, templating, retries,
rate limiting, connection pooling, async sending, middleware, DKIM signing,
and provider adapters for SendGrid, Mailgun, and AWS SES — composable,
well-tested, and dependency-light.

```bash
go get github.com/KARTIKrocks/goemail
```

## What you get

| | |
| --- | --- |
| **SMTP Support** | TLS / STARTTLS, authentication, and per-message context cancellation |
| **Templating** | Go `html/template` and `text/template` for HTML and plain-text bodies |
| **Attachments** | Send files with proper MIME encoding and content types |
| **Retry Logic** | Configurable exponential backoff for transient SMTP failures |
| **Rate Limiting** | Built-in token-bucket limiter to avoid overwhelming providers |
| **Connection Pooling** | Reuse SMTP connections for high-throughput sending |
| **Async Sending** | Background queue with configurable workers and buffer size |
| **Middleware Pipeline** | Composable logging, metrics, recovery, and lifecycle hooks |
| **Provider Adapters** | SendGrid, Mailgun, and AWS SES via HTTP APIs (separate modules) |
| **DKIM Signing** | RSA-SHA256 / Ed25519 signing per RFC 6376 + 8463 — zero extra deps |
| **Webhooks** | Receive and normalize delivery events from any provider |
| **Pluggable Logger** | Bring your own logger — slog, zap, logrus, zerolog, anything |
| **Mock Sender** | Drop-in test double with assertions on sent messages |
| **Header Safety** | Automatic header-injection protection and address validation |

## Two layers: Sender and Mailer

goemail is built around a single `Sender` interface. Concrete senders include
`NewSMTPSender`, `NewMockSender`, and provider adapters (`sendgrid.New`,
`mailgun.New`, `ses.New`). A `Mailer` wraps any sender with helpers for
plain-text, HTML, templates, and batch sending.

Middleware (logging, metrics, hooks, recovery) is applied to senders via
`email.Chain`, so you can compose any sender with any middleware without
touching the `Mailer`.

## Where to go next

- **[Getting Started](./getting-started.md)** — install and send your first email
- **[Mailer](./mailer.md)** — the high-level convenience type
- **[SMTP](./smtp.md)** — configure the built-in SMTP sender
- **[API Reference](https://pkg.go.dev/github.com/KARTIKrocks/goemail)** — full
  generated godoc on pkg.go.dev

## Documentation layout

These guides explain concepts, patterns, and configuration. For exact type
signatures, method sets, and struct fields, use
[pkg.go.dev](https://pkg.go.dev/github.com/KARTIKrocks/goemail) — it is
generated from the source and is always authoritative.

## Requirements

Go 1.22 or later. The core module has no third-party dependencies beyond
`golang.org/x/sync` and `golang.org/x/time`.
