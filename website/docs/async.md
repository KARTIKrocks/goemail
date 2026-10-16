---
id: async
title: Async Sending
description: Wrap any Sender with a buffered queue and background workers for fire-and-forget delivery.
---

# Async Sending

`AsyncSender` wraps any `Sender` with a buffered queue and a fixed pool of
background workers. `Send` validates the email, drops it on the queue,
and returns immediately — workers handle the SMTP round-trip and retry.
It's the right tool when you don't want HTTP requests waiting on email
delivery, or when you want to absorb traffic spikes without blocking.

## Creating an AsyncSender

```go
sender, _ := email.NewSMTPSender(config)

async := email.NewAsyncSender(sender,
    email.WithQueueSize(200),
    email.WithWorkers(3),
    email.WithErrorHandler(func(ctx context.Context, e *email.Email, err error) {
        log.Printf("send to %v failed: %v", e.To, err)
    }),
)
defer async.Close()
```

`AsyncSender` implements `Sender`, so it composes with middleware and with
`Mailer`:

```go
wrapped := email.Chain(async, email.WithLogging(logger))
mailer := email.NewMailer(wrapped, "no-reply@example.com")
```

## Options

| Option | Default | Description |
| --- | --- | --- |
| `WithQueueSize(n)` | 100 | Buffer capacity. `Send` returns `ErrQueueFull` when full; `SendWait` waits for space. |
| `WithWorkers(n)` | 1 | Number of goroutines pulling from the queue. |
| `WithAsyncLogger(l)` | no-op | Logger for queue lifecycle events. |
| `WithErrorHandler(fn)` | no-op | Called for every send that fails after retries. |

## Send vs SendWait

`Send` is fire-and-forget — it returns once the email is queued. Use it
for the common case where the response should not depend on email
delivery (signups, notifications).

`SendWait` queues the email and blocks until a worker has finished
sending it. Use it when you need the result inline — for example in a
CLI tool or a synchronous test.

```go
// Fire and forget — returns once queued.
if err := async.Send(ctx, msg); err != nil {
    return err // ErrQueueFull or ErrQueueClosed
}

// Block until delivered.
if err := async.SendWait(ctx, msg); err != nil {
    return err // delivery error
}
```

## Panics

_Added in 0.4._ A panic in the wrapped `Sender` does not crash the process.
The send fails with an error wrapping `ErrPanicked`: `SendWait` returns it,
and for `Send` it goes to the error handler. Match it with
`errors.Is(err, email.ErrPanicked)`.

## Close

`Close` drains the queue, waits for in-flight workers to finish, then
closes the underlying sender. Always defer it during shutdown so queued
emails are not lost.

_Changed in 0.4._ A `SendWait` still waiting for queue space returns
`ErrQueueClosed` as soon as `Close` is called; before, `Close` blocked until
that email got a slot.

```go
async := email.NewAsyncSender(sender, email.WithWorkers(3))
defer async.Close() // drain → wait → close inner sender
```
