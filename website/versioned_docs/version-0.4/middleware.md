---
id: middleware
title: Middleware
description: Compose logging, metrics, hooks, and recovery around any Sender using the Chain function.
---

# Middleware

Middleware lets you weave cross-cutting concerns — logging, metrics,
hooks, recovery, tracing — around any `Sender` without modifying it. Each
middleware is a function with the signature `func(Sender) Sender`, so they
compose cleanly.

## Chain

`email.Chain` applies middleware in the order they appear — the first
middleware is the outermost wrapper, executed first on the way in and
last on the way out:

```go
wrapped := email.Chain(sender,
    email.WithRecovery(),       // outermost: catches panics from anything below
    email.WithLogging(logger),  // logs send + duration
    email.WithMetrics(metrics), // increments counters
    email.WithHooks(hooks),     // innermost: closest to actual send
)

mailer := email.NewMailer(wrapped, "no-reply@example.com")
```

## Built-in Middleware

Four middleware ship with the core package. They cover the most common
needs and are safe to stack in any order.

| Middleware | Behavior |
| --- | --- |
| `WithLogging(Logger)` | Logs send start, success, and failure with duration |
| `WithRecovery()` | Catches panics and returns `ErrPanicked` |
| `WithHooks(SendHooks)` | `OnSend`, `OnSuccess`, `OnFailure` callbacks |
| `WithMetrics(MetricsCollector)` | Counters and duration histograms via the metrics interface |

## Send Hooks

`WithHooks` is the easiest way to plug in audit logging, success
counters, or alerting without writing a full middleware:

```go
hooks := email.SendHooks{
    OnSend: func(ctx context.Context, e *email.Email) {
        log.Printf("sending to %v", e.To)
    },
    OnSuccess: func(ctx context.Context, e *email.Email, d time.Duration) {
        metrics.SendDurationMS.Observe(d.Seconds() * 1000)
    },
    OnFailure: func(ctx context.Context, e *email.Email, err error) {
        alerts.Notify("email failed", "to", e.To, "err", err)
    },
}

wrapped := email.Chain(sender, email.WithHooks(hooks))
```

## Custom Middleware

Implement `email.Middleware` directly when you need custom behavior —
distributed tracing, throttling per recipient, blacklist filtering, etc.

```go
type blacklistSender struct {
    next email.Sender
    deny map[string]bool
}

func (s *blacklistSender) Send(ctx context.Context, e *email.Email) error {
    for _, to := range e.To {
        if s.deny[strings.ToLower(to)] {
            return fmt.Errorf("recipient %q is blacklisted", to)
        }
    }
    return s.next.Send(ctx, e)
}

func (s *blacklistSender) Close() error { return s.next.Close() }

func WithBlacklist(deny map[string]bool) email.Middleware {
    return func(next email.Sender) email.Sender {
        return &blacklistSender{next: next, deny: deny}
    }
}

wrapped := email.Chain(sender, WithBlacklist(myDenyList))
```
