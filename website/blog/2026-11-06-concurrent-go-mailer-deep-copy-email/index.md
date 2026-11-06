---
slug: concurrent-go-mailer-deep-copy-email
title: Why Your Concurrent Go Mailer Needs to Deep-Copy Every Email
authors: [kartik]
tags: [go, email, concurrency, reliability]
description: Reusing an *Email struct across goroutines, or mutating it after Send returns, looks harmless until the data race detector — or a customer — finds the one byte that changed underneath you.
---

# Why Your Concurrent Go Mailer Needs to Deep-Copy Every Email

Here's a bug that passes every single-threaded test you'll write and shows
up only under real concurrent load:

```go
mailer := email.NewMailer(sender, "noreply@example.com")

e := email.NewEmail().SetFrom("noreply@example.com").SetSubject("Receipt")
for _, order := range orders {
    e.AddTo(order.CustomerEmail) // reusing e across iterations
    go mailer.SendEmail(ctx, e)
}
```

Every goroutine captured the _same_ `*Email`. `AddTo` appends to `e.To` on
every loop iteration while earlier goroutines are concurrently reading that
same slice to build their own message. Some customers get the receipt
addressed to a `To` list with three other people on it. Some get a `fatal
error: concurrent map read and map write` if `Headers` is involved. None of
this shows up if you test by sending one email at a time.

{/* truncate */}

## The fix that looks like it's not needed

The obvious response is "don't do that, it's clearly a bug in the caller's
code." True — and also exactly the kind of bug a library that holds itself
to being "safe for concurrent use" (as
[goemail](https://github.com/KARTIKrocks/goemail)'s `Mailer` documents
itself to be) has to defend against anyway, because the alternative is a
library whose safety guarantee is actually "safe for concurrent use, as long
as callers never reuse an `*Email`" — a caveat nobody reads until after
they've shipped the bug.

`Mailer.SendEmail` doesn't take the caller's pointer at face value. It
copies first:

```go
func (m *Mailer) SendEmail(ctx context.Context, email *Email) error {
    if email == nil {
        return errors.New("nil email")
    }
    e := cloneEmail(email) // deep copy to avoid mutating the caller's email
    if e.From == "" {
        e.From = m.from
    }

    builtEmail, err := e.Build()
    if err != nil {
        return err
    }

    return m.sender.Send(ctx, builtEmail)
}
```

And `SendBatch` does the same for every email in the batch before any of
them touch a goroutine:

```go
for i, email := range emails {
    e := cloneEmail(email) // deep copy to avoid mutating the caller's email
    if e.From == "" {
        e.From = m.from
    }
    if _, err := e.Build(); err != nil {
        return fmt.Errorf("email %d validation failed: %w", i, err)
    }
    prepared[i] = e
}
```

After this point, nothing the caller does to their original `*Email` —
appending to `To`, mutating `Headers`, anything — can reach the copy that's
actually in flight to a worker goroutine.

## Why a shallow copy isn't enough

The naive version of "copy the email" is `cp := *email` — a struct copy. For
`From`, `Subject`, `Body`, that's genuinely sufficient; they're strings,
and strings are immutable in Go, so two `Email` values can share the same
underlying string data safely forever. The bug is specifically in the
fields that are reference types: `To`, `Cc`, `Bcc` (`[]string`),
`Attachments` (`[]Attachment`), and `Headers` (`map[string]string`). A
struct copy copies the _slice header_ — pointer, length, capacity — not the
backing array. Two `Email` values after `cp := *email` point at the exact
same underlying array for `To`. `cp.To = append(cp.To, "x")` can still
mutate data the original slice is reading, if the append happens to reuse
the existing backing array rather than allocating a new one (it will,
whenever there's spare capacity) — which means the shallow-copy version of
this fix only fails _intermittently_, depending on how much spare capacity
happened to be in the slice at copy time. That's a worse failure mode than
no fix at all: it passes code review, passes CI, and fails in production
under exactly the load pattern that triggers it.

`cloneEmail` does the actual work, field by field, allocating new backing
storage for every reference type:

```go
func cloneEmail(src *Email) *Email {
    if src == nil {
        return nil
    }
    dst := &Email{
        From: src.From, ReplyTo: src.ReplyTo, Subject: src.Subject,
        Body: src.Body, HTMLBody: src.HTMLBody, err: src.err,
    }
    if len(src.To) > 0 {
        dst.To = make([]string, len(src.To))
        copy(dst.To, src.To)
    }
    // ...Cc, Bcc the same shape...
    if len(src.Attachments) > 0 {
        dst.Attachments = make([]Attachment, len(src.Attachments))
        for i, att := range src.Attachments {
            dst.Attachments[i] = Attachment{
                Filename: att.Filename, ContentType: att.ContentType,
                Data: append([]byte(nil), att.Data...), // the attachment bytes too
            }
        }
    }
    if len(src.Headers) > 0 {
        dst.Headers = make(map[string]string, len(src.Headers))
        maps.Copy(dst.Headers, src.Headers)
    }
    return dst
}
```

Notice `Attachment.Data` gets its own copy, not just the `[]Attachment`
slice header — a struct inside a slice inside a struct, three levels where
a shallow copy would have stopped at level one and silently shared the
actual attachment bytes across every clone.

## The same discipline, applied where it's least obvious

`AsyncSender` has the most interesting version of this because it has to
solve two different aliasing problems with two different clones in the same
function:

```go
func (a *AsyncSender) Send(ctx context.Context, e *Email) error {
    // ...
    return poolErr(a.pool.TrySubmitJob(&asyncJob{
        sender: a.sender,
        ctx:    context.WithoutCancel(ctx),
        email:  e,              // the caller's own pointer
        toSend: cloneEmail(e),  // the copy that actually gets sent
    }))
}
```

`toSend` is the clone — it's what actually reaches the wire, so the caller
mutating `e` after `Send` returns (which they're free to do; `Send` returns
immediately and queues the work) can't change what gets delivered. `email`
deliberately keeps the caller's _original_ pointer, not a clone, because
it's only ever used for one thing: reporting to the error handler if the
send fails.

```go
func (a *AsyncSender) jobFailed(job worker.Job, err error, _ time.Duration) {
    j, ok := job.(*asyncJob)
    if !ok {
        return
    }
    err = poolErr(err)
    a.logger.Error("async email send failed",
        "error", err,
        "to", j.toSend.To, // the copy: the caller may be changing j.email
        "subject", j.toSend.Subject,
    )
    if a.errorHandler != nil {
        a.errorHandler(j.ctx, j.email, err)
    }
}
```

That inline comment — `"to", j.toSend.To, // the copy: the caller may be
changing j.email` — is the whole design decision in one line: the _log_
has to describe what was actually sent, so it reads from the immutable
clone; the _error handler callback_ is handed the caller's own object on
purpose, so application code reacting to the failure (retrying, alerting,
whatever) is working with the same object identity it started with, not a
library-internal copy it has no other reference to. Using the wrong one of
these two pointers in either place would be a subtle bug that only shows up
when a caller actually mutates `e` between calling `Send` and the async
worker picking it up — exactly the kind of thing that's invisible until a
production timing window exposes it.

## What this buys you

None of this is exotic Go — it's `make` plus `copy` plus a documented
contract. What it buys is a guarantee that's actually true rather than
true-with-an-asterisk: `Mailer` and `AsyncSender` are safe for concurrent
use _including_ when callers reuse or mutate the `*Email` they passed in,
not just when they're careful not to. The alternative — documenting "don't
reuse the struct" and hoping every caller reads it — is the kind of
guarantee that holds right up until a batch job under load finds the one
code path that didn't.

```bash
go get github.com/KARTIKrocks/goemail
```

The full clone implementation is in
[`mailer.go`](https://github.com/KARTIKrocks/goemail/blob/main/mailer.go);
async sending semantics are documented in the
[async sending guide](https://kartikrocks.github.io/goemail/docs/async).
