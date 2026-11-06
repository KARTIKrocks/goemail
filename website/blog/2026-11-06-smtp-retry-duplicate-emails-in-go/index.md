---
slug: smtp-retry-duplicate-emails-in-go
title: Your Go SMTP Client Is One Retry Away From Sending Duplicate Emails
authors: [kartik]
tags: [go, email, smtp, reliability]
description: A plain retry loop around SMTP looks safe. The two ways it actually isn't — retrying an error that was never going to succeed, and retrying a send that already half-succeeded.
---

# Your Go SMTP Client Is One Retry Away From Sending Duplicate Emails

Retry logic for email sending looks deceptively simple: wrap the send in a
loop, back off exponentially, give up after N attempts. Most hand-rolled
implementations get exactly that far and stop, because that much compiles,
passes a happy-path test, and looks done. It isn't. There are two specific
ways a retry loop around SMTP goes wrong, and both are invisible until
you're looking at a flood of duplicate bounce reports or a support ticket
that says "I got the same email six times."

{/* truncate */}

## Retrying the wrong half of the failures

The first mistake is treating every error from `smtp.Client` as equally
retryable. It isn't, and the SMTP protocol tells you so directly: RFC 5321
§4.2.1 splits reply codes into 4yz (transient — try again) and 5yz
(permanent — this will never succeed no matter how many times you ask).
A `550 mailbox unavailable` or `553 invalid sender syntax` retried three
times with exponential backoff doesn't get the email delivered on the
fourth try. It burns three attempts' worth of latency, and at list-sending
scale, it's the retry loop itself amplifying load against mail servers that
are already telling you to stop.

[goemail](https://github.com/KARTIKrocks/goemail)'s `isRetryableError`
exists specifically to make that distinction before a single backoff timer
starts:

```go
func isRetryableError(err error) bool {
    if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
        return false
    }
    // RFC 5321 §4.2.1: 5yz replies indicate permanent failure. Retrying
    // "550 mailbox unavailable" or "553 invalid sender" just burns attempts
    // and can amplify bounces on large lists.
    var tpErr *textproto.Error
    if errors.As(err, &tpErr) && tpErr.Code >= 500 && tpErr.Code < 600 {
        return false
    }
    var emailErr *Error
    if !errors.As(err, &emailErr) {
        return true
    }
    if emailErr.Op == "validate" || emailErr.Op == "build_message" {
        return false
    }
    return !errors.Is(emailErr.Err, ErrNoRecipients) &&
        !errors.Is(emailErr.Err, ErrNoSender) &&
        !errors.Is(emailErr.Err, ErrNoSubject) &&
        !errors.Is(emailErr.Err, ErrNoBody)
}
```

Four separate checks, each catching a category of error that retrying
cannot fix: a cancelled or expired context (the caller gave up — retrying
now just ignores that), a 5xx SMTP reply (deterministic protocol-level
rejection), a validation failure (the message itself is malformed — the
_ninth_ attempt at sending an email with no recipients fails identically to
the first), and message-building failures (same message, same encoding
bug, every time). What's left after those four — connection timeouts,
4xx greylisting responses, a dropped TCP connection mid-handshake — is the
actual transient-failure set retries exist for.

The reason this lives in its own function instead of being inlined into
the retry loop: it's independently testable against every error shape
`sendOnce` can produce, and it's the one piece of this logic that's load-
bearing for a decision ("give up now" vs "wait and try again") that's easy
to get backwards without a test forcing each branch.

## Retrying something that already half-succeeded

The second mistake is subtler and matters more: **where in the SMTP
conversation did the failure happen?** `MAIL FROM` failing is clean — nothing
happened, retry freely. `DATA` succeeding and then the connection dropping
before the server's final OK is a different problem entirely. Depending on
exactly when the drop occurred relative to the server committing the
message, a naive retry can resend a message the recipient's mailbox already
has a copy of — the duplicate-send problem, and the reason "I got this email
twice" tickets exist at all.

goemail doesn't solve this with cleverness at the connection level — SMTP
doesn't give you an idempotency token to solve it with. It solves it by
making retries happen at the right granularity: the entire `sendOnce` —
dial, optional `StartTLS`, `Auth`, `Mail`/`Rcpt`/`Data` — is one retry unit,
and `sendOnConn` resets connection state on partial failure before the
connection goes back in the pool:

```go
func (s *SMTPSender) sendOnConn(pc *pooledConn, from string, recipients []string, msg []byte) error {
    if err := pc.conn.SetDeadline(time.Now().Add(s.config.Timeout)); err != nil {
        return fmt.Errorf("set deadline: %w", err)
    }
    if err := sendMailData(pc.client, from, recipients, msg); err != nil {
        _ = pc.client.Reset() // best-effort state cleanup
        return err
    }
    if err := pc.conn.SetDeadline(time.Time{}); err != nil {
        return fmt.Errorf("clear deadline: %w", err)
    }
    return nil
}
```

And on the pooled path, any failure discards the connection rather than
returning it to the pool for reuse:

```go
func (s *SMTPSender) sendPooled(ctx context.Context, from string, recipients []string, msg []byte) error {
    pc, err := s.pool.get(ctx)
    if err != nil {
        return fmt.Errorf("pool get: %w", err)
    }
    if err = s.sendOnConn(pc, from, recipients, msg); err != nil {
        s.pool.discard(pc)
        return err
    }
    pc.msgCount++
    s.pool.put(pc)
    return nil
}
```

That `discard` instead of `put` on the error path is the detail that's easy
to skip and expensive to skip: a connection that failed mid-`DATA` is left
in protocol state no one can predict without re-querying the server, and
handing it back into the pool means the _next_ caller's unrelated send
inherits that undefined state. Discarding it means the next attempt — retry
or otherwise — starts clean, from a fresh `MAIL FROM`, on a connection
that's never seen the previous failure.

This doesn't make SMTP delivery exactly-once — nothing can, without the
receiving server supporting something like `CHUNKING`/`BDAT` with its own
transaction semantics, which most servers don't expose. What it buys is
_at-least-once with a clean boundary_: a given attempt either fully
succeeds or is cleanly discarded, so the worst case under retry is "this
message occasionally arrives twice because the server's final ack was lost
in transit after it actually committed the message" — a problem inherent to
the SMTP protocol itself, not one goemail's retry loop adds on top of it.

## Backoff has to respect the caller giving up

One more detail that's easy to get wrong in isolation: exponential backoff
has to be cancellable, or a caller whose context deadline has already
passed sits through the full delay anyway before the next attempt notices.

```go
func (s *SMTPSender) waitForRetry(ctx context.Context, email *Email, attempt int) error {
    const maxRetryDelay = 5 * time.Minute
    delay := time.Duration(float64(s.config.RetryDelay) *
        math.Pow(s.config.RetryBackoff, float64(attempt-1)))
    if delay > maxRetryDelay || delay <= 0 {
        delay = maxRetryDelay
    }

    timer := time.NewTimer(delay)
    select {
    case <-timer.C:
        return nil
    case <-ctx.Done():
        timer.Stop()
        return &Error{Op: "send", From: email.From, To: email.To, Err: ctx.Err()}
    }
}
```

The `delay <= 0` branch matters for a reason that isn't obvious from
reading the formula alone: `math.Pow` with a large enough exponent and
`RetryBackoff` pushes `delay` past what a `time.Duration` (an `int64` count
of nanoseconds) can represent, and that overflow wraps to a negative
number — silently, with no panic to catch it. Capping at `maxRetryDelay`
whenever `delay` comes out non-positive turns a silent overflow into a
bounded wait instead of a `time.NewTimer` call with a nonsensical, possibly
negative duration.

## What this buys you

A retry loop that doesn't distinguish 5xx from everything else just adds
latency to failures that were never going to succeed. One that doesn't
discard a connection after a failed `DATA` command turns a single bad send
into a corrupted connection the next unrelated caller inherits. Getting
both right is what separates "has a retry loop" from "is safe to retry
automatically" — and it's the second one that actually matters once this
code is sending real mail to real inboxes.

```bash
go get github.com/KARTIKrocks/goemail
```

Retry and rate-limit configuration is documented in the
[reliability guide](https://kartikrocks.github.io/goemail/docs/reliability);
the full retry implementation is in
[`smtp.go`](https://github.com/KARTIKrocks/goemail/blob/main/smtp.go).
