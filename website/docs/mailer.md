---
id: mailer
title: Mailer
description: The high-level convenience type that wraps a Sender with helpers for plain text, HTML, templates, and batch sending.
---

# Mailer

`Mailer` is the high-level convenience type. It wraps a `Sender` with
helpers for plain text, HTML, templates, and batch sending. The default
`From` address you pass to `NewMailer` is used whenever an email does not
specify its own sender.

## Creating a Mailer

```go
sender, err := email.NewSMTPSender(config)
if err != nil {
    log.Fatal(err)
}

mailer := email.NewMailer(sender, "no-reply@example.com")
defer mailer.Close() // closes the underlying sender
```

For senders wrapped in middleware, use `NewMailerWithOptions`:

```go
mailer := email.NewMailerWithOptions(sender, "no-reply@example.com",
    email.WithMiddleware(
        email.WithRecovery(),
        email.WithLogging(logger),
    ),
)
```

## Send & SendHTML

For one-off messages, the `Send` and `SendHTML` helpers skip the builder
and accept `(ctx, to, subject, body)` directly.

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Plain text
err := mailer.Send(ctx,
    []string{"alice@example.com", "bob@example.com"},
    "Daily report",
    "Today's numbers are looking good.",
)

// HTML
err = mailer.SendHTML(ctx,
    []string{"alice@example.com"},
    "Welcome!",
    "<h1>Welcome to MyApp</h1><p>Click <a href=\"https://example.com\">here</a> to get started.</p>",
)
```

## SendTemplate

Render a registered template with arbitrary data. See [Templates](./templates.md)
for how to register them.

```go
data := map[string]any{
    "Name":       "Alice",
    "VerifyLink": "https://example.com/verify/abc123",
}

err := mailer.SendTemplate(ctx,
    []string{"alice@example.com"},
    "welcome",
    data,
)
```

## Batch Sending

`SendBatch` sends many emails concurrently with a bounded worker pool.
Failures are aggregated and returned as a single joined error — successful
sends still go through.

```go
emails := []*email.Email{
    email.NewEmail().SetFrom("no-reply@example.com").AddTo("alice@example.com").
        SetSubject("Hi Alice").SetBody("Hello"),
    email.NewEmail().SetFrom("no-reply@example.com").AddTo("bob@example.com").
        SetSubject("Hi Bob").SetBody("Hello"),
    email.NewEmail().SetFrom("no-reply@example.com").AddTo("carol@example.com").
        SetSubject("Hi Carol").SetBody("Hello"),
}

// Concurrency limit of 5
if err := mailer.SendBatch(ctx, emails, 5); err != nil {
    log.Printf("some sends failed: %v", err)
}
```

## Close

Always `defer mailer.Close()`. It releases pooled SMTP connections, drains
the async queue (if wrapping `AsyncSender`), and flushes any internal
buffers.

```go
mailer := email.NewMailer(sender, config.From)
defer mailer.Close()
```
