---
slug: another-email-library-for-go
title: "Why I Built Another Email Library for Go (and What SMTP in Go Actually Gets Wrong)"
authors: [kartik]
tags: [go, email, smtp, opensource, backend]
description: Go has had an email library for over a decade. None of the options solve the problem that actually costs a production team a week — so here's an honest survey of what exists, and where the gap really is.
---

# Why I Built Another Email Library for Go (and What SMTP in Go Actually Gets Wrong)

"Another email library" deserves skepticism. Go has had `net/smtp` in the
standard library since Go 1, and there's been at least one popular
third-party wrapper around it for most of the last decade. So before
explaining why [goemail](https://github.com/KARTIKrocks/goemail) exists, the
honest version of this post starts with what's already out there — and what
each option leaves you to build yourself.

{/* truncate */}

## The landscape, as it actually is

**[`net/smtp`](https://pkg.go.dev/net/smtp)** is the foundation everything
else is built on, goemail included. It gives you a `Client` with `Mail`,
`Rcpt`, `Data`, `StartTLS`, and `PlainAuth` — a correct, minimal
implementation of the SMTP protocol itself. You can reuse one `Client` for
several sequential sends if you hold onto it yourself, but there's no
pool: no management of multiple reusable connections, no eviction of stale
ones, no concurrent checkout — and no retries, no rate limiting, no MIME
multipart assembly, no header injection protection beyond what you
remember to check yourself. That's by design — it's a transport primitive,
not a mail client — but it means every production service ends up writing
the same 200 lines around it.

**`gomail`** (the original [`go-gomail/gomail`](https://github.com/go-gomail/gomail),
and its forks — [`go-mail/mail`](https://github.com/go-mail/mail), archived
by its owner in late 2024, and a Shopify fork archived mid-2024) was for
years the default answer to "what do I use instead of raw `net/smtp`." It's
a genuinely pleasant API for _composing_ a message — attachments, HTML
alternatives, a `Dialer` that reuses one connection for a batch — and for a
lot of side projects that's the whole job. What it was never built for:
retry logic, DKIM signing, rate limiting, or anything resembling a
provider-agnostic abstraction over SendGrid or SES. And as of this writing,
every maintained fork I could find has been archived read-only by its owner
— which is its own argument for not building new production code on it.

**[`wneessen/go-mail`](https://github.com/wneessen/go-mail)** is the actively
maintained modern alternative, and the most direct comparison to goemail.
It keeps a genuinely minimal dependency footprint (standard library only in
the core), reuses SMTP connections, has pluggable SMTP auth methods beyond
`PlainAuth`, and ships proper debug logging of the wire protocol — all
things `net/smtp` leaves you to build. Where it draws its line: DKIM signing
exists, but as a separate [`go-mail-middleware`](https://github.com/wneessen/go-mail-middleware)
module built on `go-msgauth`, not in the core library. Rate limiting and
HTTP-based provider adapters (SendGrid, Mailgun, SES) aren't part of the
project at all — reasonably so, since pulling in `aws-sdk-go-v2` to sign
SES requests is exactly the kind of dependency weight a "standard-library
only" project is right to refuse.

**The provider SDKs themselves** — SendGrid's, Mailgun's, AWS's — solve
delivery through their own HTTP APIs, which sidesteps SMTP's problems
entirely (no raw sockets, no STARTTLS negotiation, often provider-side retry
and rate limiting already). The cost is that your code is now written
against SendGrid's API, not an interface — swapping providers, or falling
back from SendGrid to raw SMTP when a provider has an outage, means
rewriting the call site, not swapping a struct.

None of this is a knock on any of them. `net/smtp` is correctly scoped as a
primitive. gomail solved message composition well for the use case it had.
go-mail is a well-built, actively maintained SMTP client that makes a
deliberate, defensible choice to stay small. The gap is specifically the
**reliability and provider-abstraction layer** — retries, pooling, rate
limiting, DKIM, and a `Sender` interface that's the same whether the message
goes out over SMTP or SendGrid's API — packaged as one thing instead of
split across "write it yourself" and "pick a separate module per provider."

## What that layer actually has to do

This isn't abstract. Across the rest of this series, the specific problems
that show up the moment a `net/smtp` proof-of-concept meets real traffic —
and that neither the standard library nor a composition-focused wrapper
hands you — were:

- **Header injection that a single check, called from one place, won't
  catch.** A `Subject` or custom header built from user input is an open
  CRLF-injection vector the moment it reaches a MIME header unchecked.
  [Article 2](/blog/header-injection-vulnerability-in-go-email-code) covers
  why `AddHeader`'s guard and `Validate()`'s repeated check both exist —
  one covers the convenience path, the other covers every path, including
  the ones that bypass the convenience method entirely.
- **Retries that don't know a permanent failure from a transient one.**
  Retrying a `550 mailbox unavailable` doesn't get the email delivered — it
  just burns attempts and amplifies load on an already-bouncing address.
  [Article 3](/blog/smtp-retry-duplicate-emails-in-go) is about the RFC 5321
  status-code logic that decides what's worth retrying, and why a failed
  send still isn't safe to retry blindly even once that's sorted out.
- **Shared mutable state between a caller and whatever's sending on their
  behalf.** A `*Email` reused or mutated across goroutines races with
  whatever's reading it to build the outgoing message.
  [Article 4](/blog/concurrent-go-mailer-deep-copy-email) covers why every
  send path in goemail deep-copies the message before using it — and the
  real limits of what that copy can and can't protect against once the
  caller is mutating concurrently rather than just reusing sequentially.
- **DKIM signing that's correct per the RFC, not just "it validates in one
  inbox provider's checker."** Header canonicalization, duplicate-header
  ordering, and signature folding all have specific rules in RFC 6376 that
  are easy to get approximately right and silently wrong.
  [Article 5](/blog/dkim-signing-in-go-rfc-details) goes through the details
  that break signatures in ways that only show up against a strict verifier.
- **One interface, regardless of transport.** A `Sender` that's SMTP today
  and SendGrid's HTTP API next quarter, without touching call sites — the
  reason goemail's provider adapters
  ([SendGrid](https://github.com/KARTIKrocks/goemail/tree/main/providers/sendgrid),
  [Mailgun](https://github.com/KARTIKrocks/goemail/tree/main/providers/mailgun),
  [SES](https://github.com/KARTIKrocks/goemail/tree/main/providers/ses)) live
  in separate modules instead of separate codebases.

None of these are hard problems individually. They're the kind of thing a
team writes correctly on the third try, after the first version leaks a
goroutine, the second one double-sends on retry, and the third one finally
gets a security review before a header-injection report comes in from
outside. goemail's pitch is narrow: do that work once, keep the root module
dependency-light (only `golang.org/x/time` and a tiny worker-pool package —
the AWS SDK and OpenTelemetry stay opt-in, in their own `providers/` modules),
and let the team skip re-deriving it.

## Is it the library you need?

Depends on what you're building. If all you need is "send one HTML email
with an attachment from a side project," `net/smtp` plus fifteen lines, or
gomail if you're already using it, is lighter weight than adopting goemail's
configuration surface. If you want a minimal-dependency SMTP client with
your own choice of middleware stack, go-mail is a strong, actively
maintained option — and its DKIM middleware is worth a look even if you
don't use the rest of it.

Where goemail earns its place is the specific, common case: a Go service
that owns outbound email, needs retries and rate limiting and DKIM to be
correct from day one, will likely add a provider adapter or two as the
product grows, and doesn't want each of those to be a separate research
project. That's the gap none of the above fill as one package, and it's the
one goemail was built to close.

```bash
go get github.com/KARTIKrocks/goemail
```

The [quick start](https://kartikrocks.github.io/goemail/docs/getting-started)
is the fastest way to see SMTP, templates, and retries working end to end;
the full capability list is in the
[README](https://github.com/KARTIKrocks/goemail#why-goemail), and the rest
of this series goes deep on each piece above.
