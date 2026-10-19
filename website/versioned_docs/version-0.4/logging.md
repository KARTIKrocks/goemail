---
id: logging
title: Logging
description: Bring your own logger via a small Logger interface, with a built-in slog adapter.
---

# Logging

goemail does not pull in a logging library. Instead it defines a small
`Logger` interface that mirrors structured-logging conventions and ships
an adapter for the standard library's `slog`. Plug in your existing
logger by implementing four methods.

## Logger Interface

```go
type Logger interface {
    Debug(msg string, keysAndValues ...any)
    Info(msg string, keysAndValues ...any)
    Warn(msg string, keysAndValues ...any)
    Error(msg string, keysAndValues ...any)
    With(keysAndValues ...any) Logger
}
```

`With` returns a child logger with extra structured fields attached —
handy for threading request IDs through middleware.

## slog

For Go 1.21+, use `NewSlogLogger` to wrap any `*slog.Logger`:

```go
import (
    "log/slog"
    "os"

    email "github.com/KARTIKrocks/goemail"
)

slogLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))

config := email.SMTPConfig{
    Host:   "smtp.gmail.com",
    Logger: email.NewSlogLogger(slogLogger),
}
```

## Custom Loggers

Wrapping zap, zerolog, or logrus is a small adapter type. Here is the
minimal shape — fill in the bodies with your logger of choice:

```go
type myLogger struct{ /* underlying logger */ }

func (l myLogger) Debug(msg string, kv ...any)    { /* ... */ }
func (l myLogger) Info(msg string, kv ...any)     { /* ... */ }
func (l myLogger) Warn(msg string, kv ...any)     { /* ... */ }
func (l myLogger) Error(msg string, kv ...any)    { /* ... */ }
func (l myLogger) With(kv ...any) email.Logger    { return l /* with fields */ }
```
