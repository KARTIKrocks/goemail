// Package email provides production-ready email sending capabilities for Go applications.
//
// This package offers a complete email solution with SMTP support, HTML and plain text
// templates, file attachments, retry logic, rate limiting, and comprehensive error handling.
// It is designed to be simple to use while providing the flexibility needed for production
// applications.
//
// # Features
//
//   - SMTP sending with TLS/STARTTLS support
//   - HTML and plain text email bodies
//   - Go template engine integration for dynamic content
//   - File attachment support with proper MIME encoding
//   - Automatic retry with exponential backoff
//   - Built-in rate limiting
//   - Pluggable logging interface
//   - Context support for timeouts and cancellation
//   - Email address validation and header injection protection
//   - Mock sender for testing
//   - Batch sending with concurrency control
//   - SMTP connection pooling for high-throughput sending
//   - Composable middleware pipeline (logging, metrics, recovery, hooks)
//   - Async queue worker for non-blocking background delivery
//   - Provider adapters: SendGrid, Mailgun, AWS SES (providers submodules)
//   - DKIM signing with RSA-SHA256 and Ed25519-SHA256 (RFC 6376/8463)
//   - Minimal dependencies (only golang.org/x/sync and golang.org/x/time)
//
// # Quick Start
//
// Basic email sending:
//
//	config := email.SMTPConfig{
//	    Host:     "smtp.gmail.com",
//	    Port:     587,
//	    Username: "your-email@gmail.com",
//	    Password: "your-app-password",
//	    From:     "your-email@gmail.com",
//	    UseTLS:   true,
//	}
//
//	sender, err := email.NewSMTPSender(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	mailer := email.NewMailer(sender, config.From)
//	defer mailer.Close()
//
//	ctx := context.Background()
//	err = mailer.Send(ctx,
//	    []string{"recipient@example.com"},
//	    "Hello!",
//	    "This is a test email.",
//	)
//
// # Using Templates
//
// Create and use email templates:
//
//	tmpl := email.NewTemplate("welcome")
//	tmpl.SetSubject("Welcome {{.Name}}!")
//	tmpl.SetHTMLTemplate(`
//	    <h1>Hello {{.Name}}!</h1>
//	    <p>Welcome to our service.</p>
//	`)
//
//	mailer.RegisterTemplate("welcome", tmpl)
//
//	data := map[string]any{"Name": "John Doe"}
//	err := mailer.SendTemplate(ctx, []string{"john@example.com"}, "welcome", data)
//
// # With Attachments
//
// Send emails with file attachments:
//
//	pdfData, _ := os.ReadFile("document.pdf")
//
//	email := email.NewEmail().
//	    SetFrom("sender@example.com").
//	    AddTo("recipient@example.com").
//	    SetSubject("Document Attached").
//	    SetBody("Please find the document attached.").
//	    AddAttachment("document.pdf", "application/pdf", pdfData)
//
//	err := mailer.SendEmail(ctx, email)
//
// # Logging
//
// The package uses a simple Logger interface for observability.
// By default, no logs are produced. To enable logging, provide
// a Logger implementation in SMTPConfig:
//
//	import "log/slog"
//
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	config := email.SMTPConfig{
//	    Host:   "smtp.gmail.com",
//	    Logger: email.NewSlogLogger(logger),
//	}
//
// The package logs at these levels:
//   - Debug: Connection details, SMTP commands, retry attempts
//   - Info: Email sent/received events, configuration
//   - Warn: Retries, rate limiting, non-fatal issues
//   - Error: Send failures, validation errors, connection errors
//
// # Connection Pooling
//
// For high-throughput sending, enable SMTP connection pooling to reuse
// established connections across sends, avoiding per-email TCP + TLS + AUTH overhead:
//
//	config := email.SMTPConfig{
//	    Host:     "smtp.gmail.com",
//	    Port:     587,
//	    Username: "your-email@gmail.com",
//	    Password: "your-app-password",
//	    UseTLS:   true,
//	    PoolSize: 5, // Enable pooling with 5 max connections
//	}
//
//	sender, err := email.NewSMTPSender(config)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer sender.Close() // Important: closes all pooled connections
//
// Pool configuration options:
//   - PoolSize: Max open connections (0 = disabled, default)
//   - MaxIdleConns: Max idle connections in pool (default: 2)
//   - PoolMaxLifetime: Max connection lifetime (default: 30m)
//   - PoolMaxIdleTime: Max idle time before eviction (default: 5m)
//   - MaxMessages: Max messages per connection before rotation (default: 100)
//   - PoolWaitTimeout: Max wait when pool is exhausted (default: 5s)
//
// # Middleware Pipeline
//
// The package supports composable middleware for cross-cutting concerns.
// Middleware wraps the Sender interface, following the same pattern as net/http middleware:
//
//	wrapped := email.Chain(sender,
//	    email.WithRecovery(),
//	    email.WithLogging(logger),
//	    email.WithHooks(hooks),
//	    email.WithMetrics(collector),
//	)
//	mailer := email.NewMailer(wrapped, "from@example.com")
//
// Or use NewMailerWithOptions:
//
//	mailer := email.NewMailerWithOptions(sender, "from@example.com",
//	    email.WithMiddleware(
//	        email.WithRecovery(),
//	        email.WithLogging(logger),
//	    ),
//	)
//
// Built-in middlewares:
//   - WithLogging: logs send attempts and outcomes using the Logger interface
//   - WithRecovery: catches panics and converts them to errors
//   - WithHooks: invokes configurable OnSend/OnSuccess/OnFailure callbacks
//   - WithMetrics: records send metrics via the MetricsCollector interface
//
// For OpenTelemetry tracing, see the providers/otelmail submodule
// (github.com/KARTIKrocks/goemail/providers/otelmail).
//
// # DKIM Signing
//
// Sign outgoing emails with DKIM-Signature headers for improved deliverability.
// Set DKIMConfig on SMTPConfig to automatically sign all outgoing SMTP messages:
//
//	privateKey, _ := email.ParseDKIMPrivateKey(pemData)
//
//	config := email.SMTPConfig{
//	    Host: "smtp.example.com",
//	    Port: 587,
//	    DKIM: &email.DKIMConfig{
//	        Domain:     "example.com",
//	        Selector:   "default",
//	        PrivateKey: privateKey,
//	    },
//	}
//
// For raw message signing (e.g., with AWS SES), use BuildRawMessageWithDKIM
// or call SignMessage directly. Supports RSA-SHA256 and Ed25519-SHA256 algorithms.
//
// # Async Sending
//
// Wrap any Sender with AsyncSender for non-blocking background delivery:
//
//	async := email.NewAsyncSender(sender,
//	    email.WithQueueSize(200),
//	    email.WithWorkers(3),
//	)
//	defer async.Close()
//
//	// Fire-and-forget
//	err := async.Send(ctx, myEmail)
//
//	// Or block until delivered
//	err = async.SendWait(ctx, myEmail)
//
// AsyncSender implements the Sender interface and composes with middleware.
//
// # Provider Adapters
//
// Send via HTTP APIs instead of SMTP using providers submodules:
//   - providers/sendgrid — SendGrid v3 Web API
//   - providers/mailgun — Mailgun v3 Messages API
//   - providers/ses — AWS SES v2 with raw MIME messages
//
// All adapters implement the Sender interface and live in separate Go modules.
//
// # Testing
//
// Use MockSender for testing:
//
//	mock := email.NewMockSender()
//	mailer := email.NewMailer(mock, "test@example.com")
//
//	err := mailer.Send(ctx, []string{"user@example.com"}, "Test", "Body")
//
//	// Verify
//	if mock.GetEmailCount() != 1 {
//	    t.Error("expected 1 email")
//	}
//
// # Error Handling
//
// The package provides detailed errors with context:
//
//	err := mailer.Send(ctx, to, subject, body)
//	if err != nil {
//	    var emailErr *email.Error
//	    if errors.As(err, &emailErr) {
//	        log.Printf("Operation: %s, From: %s, To: %v, Error: %v",
//	            emailErr.Op, emailErr.From, emailErr.To, emailErr.Err)
//	    }
//	}
//
// # Context Support
//
// All send operations accept context.Context for timeouts and cancellation:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	err := mailer.Send(ctx, to, subject, body)
//	if errors.Is(err, context.DeadlineExceeded) {
//	    log.Println("send timed out")
//	}
//
// # Security
//
// The package includes several security features:
//   - Email address validation using net/mail
//   - Email header injection protection
//   - Automatic sanitization of headers
//   - TLS/STARTTLS support for encrypted connections
//
// Always use environment variables for sensitive credentials:
//
//	config := email.SMTPConfig{
//	    Host:     os.Getenv("SMTP_HOST"),
//	    Username: os.Getenv("SMTP_USERNAME"),
//	    Password: os.Getenv("SMTP_PASSWORD"),
//	}
//
// For Gmail, use App Passwords instead of your regular password.
// Enable 2FA and generate an App Password at:
// https://myaccount.google.com/apppasswords
package email
