---
id: getting-started
title: Getting Started
description: Install goemail and send your first email over SMTP.
---

# Getting Started

## Installation

Requires **Go 1.26+**. The core module has no third-party dependencies
beyond `golang.org/x/sync` and `golang.org/x/time`.

```bash
go get github.com/KARTIKrocks/goemail
```

## Quick Start

A minimal program that sends a plain-text email over SMTP with a context
timeout:

```go
package main

import (
    "context"
    "log"
    "time"

    email "github.com/KARTIKrocks/goemail"
)

func main() {
    config := email.SMTPConfig{
        Host:     "smtp.gmail.com",
        Port:     587,
        Username: "your-email@gmail.com",
        Password: "your-app-password",
        From:     "your-email@gmail.com",
        UseTLS:   true,
    }

    sender, err := email.NewSMTPSender(config)
    if err != nil {
        log.Fatal(err)
    }
    mailer := email.NewMailer(sender, config.From)
    defer mailer.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    err = mailer.Send(ctx,
        []string{"recipient@example.com"},
        "Hello from goemail",
        "This is a test email.",
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

## Two layers: Sender and Mailer

goemail is built around a single `Sender` interface. Concrete senders
include `NewSMTPSender`, `NewMockSender`, and provider adapters
(`sendgrid.New`, `mailgun.New`, `ses.New`). A `Mailer` wraps any sender with
helpers for plain-text, HTML, templates, and batch sending.

Middleware (logging, metrics, hooks, recovery) is applied to senders via
`email.Chain`, so you can compose any sender with any middleware without
touching the `Mailer`.
