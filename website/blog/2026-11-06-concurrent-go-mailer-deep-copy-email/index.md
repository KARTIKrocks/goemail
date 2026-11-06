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

e := email.NewEmail().
    SetFrom("noreply@example.com").
    SetSubject("Receipt").
    SetBody("Thanks for your order.")

for _, order := range orders {
    e.To = []string{order.CustomerEmail} // reusing e across iterations
    go func() {
        if err := mailer.SendEmail(ctx, e); err != nil {
            log.Println("send failed:", err)
        }
    }()
}
```

Every goroutine captured the _same_ `*Email` pointer. The main goroutine
reassigns `e.To` on every loop iteration — not appending, so there's no
recipient list growing across sends, just a plain field overwrite — while
a previously spawned goroutine is concurrently inside `mailer.SendEmail`,
reading that same field to build its own copy of the message. That read
and that write have no ordering relationship the Go memory model
guarantees; `go test -race` will flag it, and in practice some customers
get a receipt addressed to the _next_ order's recipient instead of their
own. None of this shows up if you test by sending one email at a time.

{/* truncate */}

## What the internal copy fixes, and what it can't

The obvious response to the example above is "don't do that, it's clearly
a bug in the caller's code" — and that's true in a way worth being precise
about. The race happens entirely in the caller's own goroutine, on the
caller's own field assignment, before
[goemail](https://github.com/KARTIKrocks/goemail)'s `Mailer` ever gets a
chance to run. No amount of copying inside `SendEmail` can retroactively
fix a write that already raced with a read that started before the copy
existed — once you're dispatching a send with `go` and mutating the same
`*Email` from the dispatching goroutine afterward, the data race is
unavoidable regardless of what the callee does with its copy of the
pointer. The fix for _this_ bug is in the caller: build a separate
`*Email` per order instead of reusing one.

Where the internal copy earns its keep is a different, far more common
pattern: calling `Send` or `SendEmail` _synchronously_, in a loop, and
reusing or mutating the same `*Email` for the next iteration once the
previous call has returned:

```go
e := email.NewEmail().SetFrom("noreply@example.com").SetSubject("Receipt")
for _, order := range orders {
    e.To = []string{order.CustomerEmail}
    if err := mailer.SendEmail(ctx, e); err != nil { // blocks until done
        log.Println("send failed:", err)
    }
}
```

This has no race — Go's sequential program order guarantees the next
`e.To = ...` can't run until `SendEmail` has already returned — but it
still needs the library to not hold onto a reference to `e` past the
point where `Send` hands the message off. `Mailer.SendEmail` doesn't take
the caller's pointer at face value. It copies first:

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

After `SendEmail` returns, nothing the caller does to their original
`*Email` — reassigning `To`, mutating `Headers`, anything — can reach the
copy that was already handed to the underlying `Sender`. That's the
guarantee this buys: safe to reuse _sequentially_, once a call has
returned, not safe to mutate _concurrently_ with a call that's still in
flight.

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

That design choice also hands the caller a sharp edge along with the
convenience: `j.email` is still the caller's own, live, mutable object. If
the error handler fires concurrently with the caller mutating that same
`*Email` elsewhere — or if the handler's response is "retry by calling
`Send` again with this object" — the caller is responsible for
synchronizing that access or for having kept a separate snapshot to retry
with, the same way `AsyncSender` keeps `toSend` separate from `email`
internally. The library can't do that synchronization on the caller's
behalf without taking away the one thing this parameter exists to
provide: the caller's own, unwrapped reference.

## What this buys you, and what it doesn't

None of this is exotic Go — it's `make` plus `copy` plus a documented
contract. What it buys is narrower than "safe no matter what you do with
the pointer," and worth stating precisely rather than rounding up:
`Mailer` and `AsyncSender` guarantee that once a call has taken your
`*Email` — `Send` or `SendEmail` returned, `SendBatch` finished preparing
its copies, `AsyncSender.Send` returned after enqueueing — nothing
internal holds a reference to it anymore, so reusing or mutating that same
`*Email` for your _next_ message is always safe. It is not, and can't be
made to be, a guarantee that it's safe to mutate a `*Email` concurrently
with a call that's still in flight on it — that race exists in the
caller's own code, before the library's copy ever runs, and no internal
`make`/`copy` pair can reach backward in time to prevent it.

The practical rule that follows: one `*Email` per concurrent send. Build a
fresh one per goroutine, per recipient, per retry — exactly like the
corrected loop earlier in this post — and whether the library deep-copies
internally stops being something you need to reason about at all, because
there's no shared mutable state left for a race to happen on.

```bash
go get github.com/KARTIKrocks/goemail
```

The full clone implementation is in
[`mailer.go`](https://github.com/KARTIKrocks/goemail/blob/main/mailer.go);
async sending semantics are documented in the
[async sending guide](https://kartikrocks.github.io/goemail/docs/async).
