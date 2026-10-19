---
id: reliability
title: Retry & Rate Limiting
description: Configurable exponential-backoff retries and a built-in token-bucket rate limiter for SMTP sends.
---

# Retry & Rate Limiting

SMTP is not always reliable: connections drop, providers throttle, and
downstream mail servers temporarily refuse delivery. `SMTPSender` handles
these cases for you with configurable retries and a built-in
token-bucket rate limiter.

## Retry Logic

Retries use exponential backoff. With the defaults (`MaxRetries=3`,
`RetryDelay=1s`, `RetryBackoff=2.0`), failed sends are retried after 1s,
2s, and 4s before giving up.

```go
config := email.SMTPConfig{
    Host:         "smtp.gmail.com",
    Port:         587,
    Username:     "you@gmail.com",
    Password:     "app-password",
    UseTLS:       true,
    MaxRetries:   5,                // up to 5 retries
    RetryDelay:   500 * time.Millisecond,
    RetryBackoff: 2.0,              // 0.5s, 1s, 2s, 4s, 8s
}
```

Only transient failures (network errors, 4xx SMTP responses) are
retried. Permanent failures — invalid recipient, authentication failure,
message rejected — fail immediately.

## Rate Limiting

Most SMTP providers cap how many emails per second they accept from a
single sender. Set `RateLimit` in messages-per-second; `SMTPSender`
blocks each `Send` until a token is available.

```go
config := email.SMTPConfig{
    Host:      "smtp.gmail.com",
    RateLimit: 5, // up to 5 messages/second
}
```

The default is `10/s`. A value of `0` disables rate limiting entirely.

## Context Timeouts

Every `Send` takes a `context.Context`. Use it to bound how long an
individual send may take, or to cancel work when the request that
triggered the send is done.

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := mailer.Send(ctx, to, subject, body); err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("send timed out")
    }
    return err
}
```
