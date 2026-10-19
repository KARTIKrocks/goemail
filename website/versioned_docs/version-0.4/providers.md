---
id: providers
title: Provider Adapters
description: Send mail through SendGrid, Mailgun, AWS SES, or add OpenTelemetry tracing, as separate Go modules.
---

# Provider Adapters

Provider adapters send mail through an HTTP API instead of SMTP — usually
faster, more reliable, and with richer delivery feedback. Each adapter
lives in its own Go module under `providers/` so the core library stays
dependency-free. Every adapter implements the standard `email.Sender`
interface, so you can swap providers, wrap them in middleware, or feed
them to `AsyncSender` with no other code changes.

## SendGrid

```bash
go get github.com/KARTIKrocks/goemail/providers/sendgrid
```

```go
import (
    email "github.com/KARTIKrocks/goemail"
    "github.com/KARTIKrocks/goemail/providers/sendgrid"
)

sender, err := sendgrid.New(sendgrid.Config{
    APIKey: os.Getenv("SENDGRID_API_KEY"),
})
if err != nil {
    log.Fatal(err)
}

mailer := email.NewMailer(sender, "no-reply@example.com")
```

## Mailgun

```bash
go get github.com/KARTIKrocks/goemail/providers/mailgun
```

```go
import "github.com/KARTIKrocks/goemail/providers/mailgun"

sender, err := mailgun.New(mailgun.Config{
    Domain: "mg.example.com",
    APIKey: os.Getenv("MAILGUN_API_KEY"),
    // BaseURL: "https://api.eu.mailgun.net", // for EU accounts
})
```

## AWS SES

The SES adapter uses the AWS SDK v2 SES v2 client. AWS credentials are
picked up from the ambient environment (env vars, EC2 instance role, IAM
role for service accounts, etc.).

```bash
go get github.com/KARTIKrocks/goemail/providers/ses
```

```go
import "github.com/KARTIKrocks/goemail/providers/ses"

sender, err := ses.New(context.Background(), ses.Config{
    Region: "us-east-1",
})
```

## OpenTelemetry

`otelmail` is a middleware, not a sender — it adds a distributed-tracing
span around every send. Each span carries `email.from`, `email.to`,
`email.subject`, and `email.recipients.count` attributes; failures record
the error and set status to `Error`.

```bash
go get github.com/KARTIKrocks/goemail/providers/otelmail
```

```go
import (
    email "github.com/KARTIKrocks/goemail"
    "github.com/KARTIKrocks/goemail/providers/otelmail"
)

wrapped := email.Chain(sender,
    otelmail.WithTracing(),       // creates a span per Send
    email.WithLogging(logger),
)

// Optional: customize the tracer or span name
// otelmail.WithTracing(otelmail.WithTracerName("myapp.email"))
```
