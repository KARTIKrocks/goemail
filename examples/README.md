# Examples

Runnable programs demonstrating goemail. Each is a self-contained `main.go` —
`go run ./examples/<name>` from the repo root.

Most examples talk to a real SMTP server via `SMTP_HOST` / `SMTP_PORT` /
`SMTP_USERNAME` / `SMTP_PASSWORD` / `SMTP_FROM` environment variables (a local
catcher like [MailHog](https://github.com/mailhog/MailHog) or
[smtp4dev](https://github.com/rnwood/smtp4dev) works fine for trying these out
without a real mailbox). `middleware` and the `testing` example are the
exception — they use `email.MockSender` and need nothing external.

| Example | Demonstrates |
| --- | --- |
| [`basic`](basic) | The minimum viable send: `SMTPConfig` → `Sender` → `Mailer.Send` |
| [`template`](template) | Registering and rendering an HTML template with `Mailer.SendTemplate` |
| [`attachment`](attachment) | Attaching files with the `Email` builder |
| [`batch`](batch) | Rate limiting and retry with backoff across multiple sends |
| [`pool`](pool) | Reusing SMTP connections under concurrent load via `SMTPConfig.PoolSize` |
| [`middleware`](middleware) | Chaining `WithRecovery`, `WithLogging`, `WithHooks`, `WithMetrics` — no SMTP server needed |
| [`testing`](testing) | Unit-testing code that sends email with `email.MockSender`, no network at all |

Start with `basic`, then `template` or `attachment` for the builder API.
`pool` and `batch` show the reliability features (retries, rate limiting,
connection reuse) under load. `middleware` and `testing` are the two to read
first if you just want to see the API without standing up an SMTP server.

For DKIM signing, provider adapters (SendGrid/Mailgun/SES), async sending, and
webhook parsing — which don't have a dedicated example yet — see the
[Documentation](https://kartikrocks.github.io/goemail/) and
[pkg.go.dev](https://pkg.go.dev/github.com/KARTIKrocks/goemail).
