# 📧 goemail

[![Go Reference](https://pkg.go.dev/badge/github.com/KARTIKrocks/goemail.svg)](https://pkg.go.dev/github.com/KARTIKrocks/goemail)
[![Go Report Card](https://goreportcard.com/badge/github.com/KARTIKrocks/goemail)](https://goreportcard.com/report/github.com/KARTIKrocks/goemail)
[![Go Version](https://img.shields.io/github/go-mod/go-version/KARTIKrocks/goemail)](go.mod)
[![CI](https://github.com/KARTIKrocks/goemail/actions/workflows/ci.yml/badge.svg)](https://github.com/KARTIKrocks/goemail/actions/workflows/ci.yml)
[![GitHub tag](https://img.shields.io/github/v/tag/KARTIKrocks/goemail)](https://github.com/KARTIKrocks/goemail/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![codecov](https://codecov.io/gh/KARTIKrocks/goemail/branch/main/graph/badge.svg)](https://codecov.io/gh/KARTIKrocks/goemail)

Production-ready email package for Go with SMTP support, templating, and retry logic.

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

### Configuration

```go
type SMTPConfig struct {
    Host            string        // SMTP server hostname
    Port            int           // SMTP server port (typically 587 for TLS, 465 for SSL)
    Username        string        // SMTP username
    Password        string        // SMTP password
    From            string        // Default sender email
    UseTLS          bool          // Use STARTTLS
    Timeout         time.Duration // Connection timeout (default: 30s)
    MaxRetries      int           // Max retry attempts (default: 3)
    RetryDelay      time.Duration // Initial retry delay (default: 1s)
    RetryBackoff    float64       // Backoff multiplier (default: 2.0)
    RateLimit       int           // Max emails per second (default: 10)
    PoolSize        int           // Max pooled connections (0 = disabled)
    MaxIdleConns    int           // Max idle connections (default: 2)
    PoolMaxLifetime time.Duration // Max connection lifetime (default: 30m)
    PoolMaxIdleTime time.Duration // Max idle time before eviction (default: 5m)
    MaxMessages     int           // Max messages per connection (default: 100)
    PoolWaitTimeout time.Duration // Max wait when pool full (default: 5s)
    Logger          Logger        // Optional logger interface
    DKIM            *DKIMConfig   // Optional DKIM signing configuration
}
```

### Common SMTP Providers

#### Gmail

```go
config := email.SMTPConfig{
    Host:     "smtp.gmail.com",
    Port:     587,
    Username: "your-email@gmail.com",
    Password: "your-app-password", // Use App Password, not regular password
    UseTLS:   true,
}
```

**Note:** Enable 2FA and create an App Password at https://myaccount.google.com/apppasswords

#### SendGrid

```go
config := email.SMTPConfig{
    Host:     "smtp.sendgrid.net",
    Port:     587,
    Username: "apikey",
    Password: "your-sendgrid-api-key",
    UseTLS:   true,
}
```

#### AWS SES

```go
config := email.SMTPConfig{
    Host:     "email-smtp.us-east-1.amazonaws.com",
    Port:     587,
    Username: "your-ses-smtp-username",
    Password: "your-ses-smtp-password",
    UseTLS:   true,
}
```

#### Mailgun

```go
config := email.SMTPConfig{
    Host:     "smtp.mailgun.org",
    Port:     587,
    Username: "postmaster@your-domain.mailgun.org",
    Password: "your-mailgun-smtp-password",
    UseTLS:   true,
}
```

### Email Builder API

```go
msg := email.NewEmail().
    SetFrom("sender@example.com").
    AddTo("user1@example.com", "user2@example.com").
    AddCc("manager@example.com").
    AddBcc("archive@example.com").
    SetReplyTo("support@example.com").
    SetSubject("Important Update").
    SetBody("Plain text body").
    SetHTMLBody("<h1>HTML body</h1>").
    AddHeader("X-Priority", "1").
    AddAttachment("report.pdf", "application/pdf", pdfData)

// Validate
built, err := msg.Build()
if err != nil {
    log.Fatal(err)
}

if err := sender.Send(ctx, built); err != nil {
    log.Fatal(err)
}
```

### Batch Sending

```go
emails := []*email.Email{
    email.NewEmail().SetFrom("no-reply@example.com").AddTo("user1@example.com").SetSubject("Hi").SetBody("Hello"),
    email.NewEmail().SetFrom("no-reply@example.com").AddTo("user2@example.com").SetSubject("Hi").SetBody("Hello"),
    email.NewEmail().SetFrom("no-reply@example.com").AddTo("user3@example.com").SetSubject("Hi").SetBody("Hello"),
}

// Send with concurrency limit of 5
err := mailer.SendBatch(ctx, emails, 5)
```

### Connection Pooling

For high-throughput sending, enable SMTP connection pooling to reuse established connections and avoid per-email TCP + TLS + AUTH overhead:

```go
config := email.SMTPConfig{
    Host:     "smtp.gmail.com",
    Port:     587,
    Username: "your-email@gmail.com",
    Password: "your-app-password",
    UseTLS:   true,
    PoolSize: 5,  // Enable pooling with max 5 connections
}

sender, err := email.NewSMTPSender(config)
if err != nil {
    log.Fatal(err)
}
defer sender.Close() // Important: closes all pooled connections

mailer := email.NewMailer(sender, config.From)

// Send many emails — connections are reused automatically
for _, recipient := range recipients {
    err := mailer.Send(ctx, []string{recipient}, "Hello!", "Message body")
    if err != nil {
        log.Printf("send to %s failed: %v", recipient, err)
    }
}
```

Pool configuration options:

| Field             | Default      | Description                                 |
| ----------------- | ------------ | ------------------------------------------- |
| `PoolSize`        | 0 (disabled) | Max open connections                        |
| `MaxIdleConns`    | 2            | Max idle connections in pool                |
| `PoolMaxLifetime` | 30m          | Max connection lifetime                     |
| `PoolMaxIdleTime` | 5m           | Max idle time before eviction               |
| `MaxMessages`     | 100          | Max messages per connection before rotation |
| `PoolWaitTimeout` | 5s           | Max wait when pool is exhausted             |

### Middleware Pipeline

Add cross-cutting concerns like logging, metrics, and panic recovery using composable middleware:

```go
// Create your sender (SMTP, mock, etc.)
sender, _ := email.NewSMTPSender(config)

// Wrap with middleware using Chain
wrapped := email.Chain(sender,
    email.WithRecovery(),              // Catch panics
    email.WithLogging(logger),         // Log sends
    email.WithHooks(email.SendHooks{   // Lifecycle callbacks
        OnSuccess: func(ctx context.Context, e *email.Email, d time.Duration) {
            fmt.Printf("sent to %v in %s\n", e.To, d)
        },
    }),
    email.WithMetrics(myCollector),    // Record metrics
)

mailer := email.NewMailer(wrapped, config.From)
```

Or use `NewMailerWithOptions` for a more compact setup:

```go
mailer := email.NewMailerWithOptions(sender, config.From,
    email.WithMiddleware(
        email.WithRecovery(),
        email.WithLogging(logger),
        email.WithMetrics(myCollector),
    ),
)
```

**Built-in middlewares:**

| Middleware                      | Description                                    |
| ------------------------------- | ---------------------------------------------- |
| `WithLogging(Logger)`           | Logs send start, success/failure with duration |
| `WithRecovery()`                | Catches panics, returns `ErrPanicked` error    |
| `WithHooks(SendHooks)`          | `OnSend`, `OnSuccess`, `OnFailure` callbacks   |
| `WithMetrics(MetricsCollector)` | Counters + duration via pluggable interface    |

**Custom middleware:** Implement the `Middleware` type (`func(Sender) Sender`) to add your own behavior (tracing, throttling, etc.).

### OpenTelemetry Tracing

Add distributed tracing to email sends with the `providers/otelmail` submodule. It lives in a separate Go module so the core library stays dependency-free.

```bash
go get github.com/KARTIKrocks/goemail/providers/otelmail
```

```go
import "github.com/KARTIKrocks/goemail/providers/otelmail"

// Wrap your sender with OTel tracing
wrapped := email.Chain(sender,
    otelmail.WithTracing(),          // creates a span per Send
    email.WithLogging(logger),
)
```

Each span includes attributes: `email.from`, `email.to`, `email.subject`, `email.recipients.count`. On failure the span records the error and sets status to `Error`.

Options:

- `otelmail.WithTracerProvider(tp)` — use a custom `TracerProvider` (default: global)
- `otelmail.WithTracerName(name)` — custom tracer name
- `otelmail.WithSpanName(name)` — custom span name (default: `"email.send"`)

### Async Sending

For non-blocking email delivery, wrap any `Sender` with `AsyncSender`. Emails are validated eagerly and queued for background workers:

```go
sender, _ := email.NewSMTPSender(config)

async := email.NewAsyncSender(sender,
    email.WithQueueSize(200),   // Buffer up to 200 emails
    email.WithWorkers(3),       // 3 background workers
    email.WithErrorHandler(func(ctx context.Context, e *email.Email, err error) {
        log.Printf("failed to send to %v: %v", e.To, err)
    }),
)
defer async.Close() // Drains queue, waits for workers, closes underlying sender

// Fire-and-forget — returns immediately
err := async.Send(ctx, myEmail)

// Or block until delivered
err = async.SendWait(ctx, myEmail)
```

`AsyncSender` implements the `Sender` interface, so it composes with middleware and `Mailer`:

```go
wrapped := email.Chain(async, email.WithLogging(logger))
mailer := email.NewMailer(wrapped, "from@example.com")
```

### Provider Adapters

Send emails via HTTP APIs instead of SMTP. Each adapter is a separate Go module under `providers/` with no extra dependencies on the core library.

#### SendGrid

```bash
go get github.com/KARTIKrocks/goemail/providers/sendgrid
```

```go
import "github.com/KARTIKrocks/goemail/providers/sendgrid"

sender, err := sendgrid.New(sendgrid.Config{
    APIKey: os.Getenv("SENDGRID_API_KEY"),
})
```

#### Mailgun

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

#### AWS SES

```bash
go get github.com/KARTIKrocks/goemail/providers/ses
```

```go
import "github.com/KARTIKrocks/goemail/providers/ses"

sender, err := ses.New(context.Background(), ses.Config{
    Region: "us-east-1",
})
```

All adapters implement `email.Sender` — use them with `Mailer`, middleware, `AsyncSender`, or directly.

### DKIM Signing

Sign outgoing emails with DKIM-Signature headers for improved deliverability and authentication. Supports RSA-SHA256 and Ed25519-SHA256 algorithms with zero additional dependencies.

#### With SMTP

```go
// Load your DKIM private key
pemData, _ := os.ReadFile("dkim-private.pem")
privateKey, err := email.ParseDKIMPrivateKey(pemData)
if err != nil {
    log.Fatal(err)
}

config := email.SMTPConfig{
    Host:     "smtp.example.com",
    Port:     587,
    Username: "user@example.com",
    Password: "password",
    UseTLS:   true,
    DKIM: &email.DKIMConfig{
        Domain:     "example.com",
        Selector:   "default",       // DNS record: default._domainkey.example.com
        PrivateKey: privateKey,
        // Optional:
        // HeaderCanonicalization: email.CanonicalizationRelaxed, // default
        // BodyCanonicalization:   email.CanonicalizationRelaxed, // default
        // Expiration:             24 * time.Hour,                // signature validity
    },
}

sender, _ := email.NewSMTPSender(config)
// All emails sent through this sender are automatically DKIM-signed
```

#### With Raw Messages (SES, custom providers)

```go
e := email.NewEmail().
    SetFrom("sender@example.com").
    AddTo("recipient@example.com").
    SetSubject("Signed Email").
    SetBody("This email is DKIM-signed.")

dkimConfig := &email.DKIMConfig{
    Domain:     "example.com",
    Selector:   "default",
    PrivateKey: privateKey,
}

msg, err := email.BuildRawMessageWithDKIM(e, dkimConfig)
// msg now contains the DKIM-Signature header
```

#### Standalone Signing

```go
// Sign any raw RFC 2822 message
signedMsg, err := email.SignMessage(rawMessage, dkimConfig)
```

### Templates

#### Creating Templates

```go
// HTML + Text template
tmpl := email.NewTemplate("newsletter")
tmpl.SetSubject("Newsletter - {{.Month}} {{.Year}}")

tmpl.SetTextTemplate("Hello {{.Name}}, check out our newsletter at {{.Link}}")

tmpl.SetHTMLTemplate(`
<!DOCTYPE html>
<html>
<body>
    <h2>{{.Title}}</h2>
    <p>Hello {{.Name}},</p>
    <div>{{.Content}}</div>
    <a href="{{.Link}}">Read More</a>
</body>
</html>
`)

// Register
mailer.RegisterTemplate("newsletter", tmpl)

// Send
data := map[string]any{
    "Name":    "Jane",
    "Month":   "January",
    "Year":    2024,
    "Title":   "Monthly Update",
    "Content": "Here's what's new...",
    "Link":    "https://example.com/newsletter",
}

err := mailer.SendTemplate(ctx, []string{"jane@example.com"}, "newsletter", data)
```

#### Loading from Files

```go
// Load HTML template from file
tmpl, err := email.LoadTemplateFromFile("welcome", "templates/welcome.html")
if err != nil {
    log.Fatal(err)
}

mailer.RegisterTemplate("welcome", tmpl)
```

### Logging

The package uses a simple `Logger` interface, allowing you to integrate with any logging library.

#### Using slog (Go 1.21+)

```go
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

config := email.SMTPConfig{
    Host:   "smtp.gmail.com",
    Logger: email.NewSlogLogger(logger),
}
```

#### Custom Logger

Implement the `Logger` interface to integrate any logging library (zap, logrus, zerolog, etc.).

```go
type MyLogger struct{}

func (l MyLogger) Debug(msg string, keysAndValues ...any) {
    // Your implementation
}

func (l MyLogger) Info(msg string, keysAndValues ...any) {
    // Your implementation
}

func (l MyLogger) Warn(msg string, keysAndValues ...any) {
    // Your implementation
}

func (l MyLogger) Error(msg string, keysAndValues ...any) {
    // Your implementation
}

func (l MyLogger) With(keysAndValues ...any) email.Logger {
    return l
}

config := email.SMTPConfig{
    Host:   "smtp.gmail.com",
    Logger: MyLogger{},
}
```

### Testing

```go
func TestSendWelcomeEmail(t *testing.T) {
    // Create mock sender
    mock := email.NewMockSender()
    mailer := email.NewMailer(mock, "test@example.com")

    // Send email
    ctx := context.Background()
    err := mailer.Send(ctx,
        []string{"user@example.com"},
        "Welcome",
        "Welcome to our service!",
    )

    if err != nil {
        t.Fatalf("send failed: %v", err)
    }

    // Verify
    if mock.GetEmailCount() != 1 {
        t.Errorf("expected 1 email, got %d", mock.GetEmailCount())
    }

    email := mock.GetLastEmail()
    if email.Subject != "Welcome" {
        t.Errorf("expected subject 'Welcome', got '%s'", email.Subject)
    }

    if email.To[0] != "user@example.com" {
        t.Errorf("expected to 'user@example.com', got '%s'", email.To[0])
    }
}
```

## 🔒 Security Best Practices

### 1. Use Environment Variables

```go
config := email.SMTPConfig{
    Host:     os.Getenv("SMTP_HOST"),
    Port:     getEnvInt("SMTP_PORT", 587),
    Username: os.Getenv("SMTP_USERNAME"),
    Password: os.Getenv("SMTP_PASSWORD"),
    UseTLS:   true,
}

func getEnvInt(key string, defaultVal int) int {
    if val := os.Getenv(key); val != "" {
        if i, err := strconv.Atoi(val); err == nil {
            return i
        }
    }
    return defaultVal
}
```

### 2. Validate Emails

The package automatically validates email addresses, but you can also manually validate:

```go
email := email.NewEmail().
    SetFrom("sender@example.com").
    AddTo("recipient@example.com").
    SetSubject("Test").
    SetBody("Body")

// Build validates the email
builtEmail, err := email.Build()
if err != nil {
    // Handle validation error
}
```

### 3. Use App Passwords (Gmail)

Don't use your regular Gmail password. Generate an App Password:

1. Enable 2-Step Verification
2. Go to https://myaccount.google.com/apppasswords
3. Select "Mail" and generate password
4. Use the generated password in your config

## 📊 Advanced Features

### Rate Limiting

```go
config := email.SMTPConfig{
    Host:      "smtp.gmail.com",
    RateLimit: 5, // Max 5 emails per second
}
```

### Retry Logic

```go
config := email.SMTPConfig{
    Host:         "smtp.gmail.com",
    MaxRetries:   5,           // Retry up to 5 times
    RetryDelay:   time.Second, // Start with 1s delay
    RetryBackoff: 2.0,         // Double delay each retry (1s, 2s, 4s, 8s, 16s)
}
```

### Context Timeouts

```go
// Timeout after 10 seconds
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

err := mailer.Send(ctx, to, subject, body)
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by the Go community's best practices
- Thanks to all contributors

## 📞 Support

- 🐛 Issues: https://github.com/KARTIKrocks/goemail/issues
- 📖 Documentation: https://pkg.go.dev/github.com/KARTIKrocks/goemail

## ⭐ Star History

If you find this package useful, please consider giving it a star!
