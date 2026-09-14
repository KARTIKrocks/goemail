---
id: smtp
title: SMTP
description: Configure the built-in SMTP sender — SMTPConfig, common providers, and connection pooling.
---

# SMTP

`NewSMTPSender` opens a connection to your SMTP server with TLS or
STARTTLS, authenticates, and implements the `Sender` interface. All knobs
— retries, timeouts, rate limits, pooling, DKIM — live in `SMTPConfig`.

## SMTPConfig

Only `Host`, `Port`, and credentials are required. Defaults are sensible
for most providers; tune them for your throughput and reliability needs.

```go
type SMTPConfig struct {
    // Required
    Host     string // SMTP server hostname
    Port     int    // 587 for STARTTLS, 465 for implicit TLS, 25 for plain
    Username string
    Password string
    From     string // Default sender for Mailer.Send

    // TLS
    UseTLS bool // STARTTLS upgrade after EHLO

    // Reliability
    Timeout      time.Duration // 30s
    MaxRetries   int           // 3
    RetryDelay   time.Duration // 1s
    RetryBackoff float64       // 2.0
    RateLimit    int           // emails/sec, default 10

    // Pooling (PoolSize > 0 enables it)
    PoolSize        int
    MaxIdleConns    int           // 2
    PoolMaxLifetime time.Duration // 30m
    PoolMaxIdleTime time.Duration // 5m
    MaxMessages     int           // per connection, default 100
    PoolWaitTimeout time.Duration // 5s

    // Optional integrations
    Logger Logger
    DKIM   *DKIMConfig
}
```

## Common Providers

SMTP settings for the most common transactional providers:

### Gmail

Use an [App Password](https://myaccount.google.com/apppasswords) with
2-Step Verification enabled.

```go
email.SMTPConfig{
    Host:     "smtp.gmail.com",
    Port:     587,
    Username: "you@gmail.com",
    Password: "your-app-password",
    UseTLS:   true,
}
```

### SendGrid

```go
email.SMTPConfig{
    Host:     "smtp.sendgrid.net",
    Port:     587,
    Username: "apikey",
    Password: os.Getenv("SENDGRID_API_KEY"),
    UseTLS:   true,
}
```

### AWS SES

```go
email.SMTPConfig{
    Host:     "email-smtp.us-east-1.amazonaws.com",
    Port:     587,
    Username: os.Getenv("SES_SMTP_USERNAME"),
    Password: os.Getenv("SES_SMTP_PASSWORD"),
    UseTLS:   true,
}
```

### Mailgun

```go
email.SMTPConfig{
    Host:     "smtp.mailgun.org",
    Port:     587,
    Username: "postmaster@mg.example.com",
    Password: os.Getenv("MAILGUN_SMTP_PASSWORD"),
    UseTLS:   true,
}
```

## Connection Pooling

For sustained throughput, set `PoolSize > 0` to reuse SMTP connections.
Without pooling, every send pays the TCP + TLS + AUTH handshake cost
(often 200–500ms per message).

```go
config := email.SMTPConfig{
    Host:            "smtp.gmail.com",
    Port:            587,
    Username:        "you@gmail.com",
    Password:        "app-password",
    UseTLS:          true,
    PoolSize:        5,                // up to 5 open connections
    MaxIdleConns:    2,                // keep 2 warm when idle
    PoolMaxLifetime: 30 * time.Minute, // recycle long-lived conns
    MaxMessages:     100,              // rotate after 100 messages
}

sender, err := email.NewSMTPSender(config)
if err != nil {
    log.Fatal(err)
}
defer sender.Close() // closes every pooled connection
```

**Tip:** always `defer sender.Close()` when pooling is enabled — otherwise
idle connections leak until the process exits.
