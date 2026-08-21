# Style and pattern rationale

Context for the scoped rules in `config.json`. This file is freeform prose read
alongside the diff; `config.json` is what actually gates comment scope and
severity.

## CRLF injection — why this is the top rule

CHANGELOG 0.2.0's Security section documents the exact shape of this
library's real vulnerability class:

- Custom header names outside the RFC 5322 field-name set, or CR/LF in header
  values, must be rejected when building a message.
- An email address that fails `net/mail.ParseAddress` must have CR/LF
  **stripped**, not passed through unmodified into a header — a value that
  looks like an address but doesn't parse is exactly the kind of input an
  attacker controls.
- Attachment filenames and content types need the same CR/LF check, since
  both end up in a MIME header line.

Any new place a string reaches `smtp.go`'s message bytes or a MIME header in
`mime.go` needs to go through the same validation `AddHeader`/`Validate`
already does — not a fresh, slightly-different sanitization written inline.

## DKIM canonicalization is exact-bytes correctness, not style

`dkim.go` implements RFC 6376 (RSA-SHA256) and RFC 8463 (Ed25519-SHA256)
canonicalization. The failure mode for a subtly wrong `canonicalizeHeader` or
`canonicalizeBody` is not a compile error or an obviously-wrong test — it's a
DKIM-Signature header that some receiving mail servers verify and others
don't, discovered in production as a deliverability problem weeks later.
Treat any change to header/body canonicalization, unfolding, trailing-CRLF
handling, or the tag-list building in `buildDKIMTagList` as requiring a spec
citation or a known-answer test vector, not just "the existing tests still
pass."

## Pool and async lifecycles

`smtpPool` (`pool.go`) and `AsyncSender` (`async.go`) are the two places this
library runs background goroutines and holds resources across calls. A
connection acquired via `get`/`dial`/`tryGetIdle` that isn't returned or
closed on every path is a slow leak that won't show up in a quick test run —
it shows up as exhausted connections under sustained load. Similarly, a
worker goroutine in `AsyncSender` without a termination path tied to `Close`
is invisible until something notices the process won't exit cleanly.

## Mailer's copy-on-send contract

`Mailer` documents itself as safe for concurrent use, and the mechanism is
`cloneEmail` deep-copying every outgoing `*Email` before it's mutated or
handed to a `Sender`. CHANGELOG 0.2.0 fixed a real bug where `SendEmail`/
`SendBatch` skipped this, letting a caller's reused slice/map get mutated
out from under a concurrent send. Any new `Mailer` method, or any
`Middleware` that touches the `*Email` it's given, needs to go through the
same copy discipline rather than assuming the caller won't reuse the value.

## Multi-module boundary

Five modules: root (`.`), `providers/sendgrid`, `providers/mailgun`,
`providers/ses`, `providers/otelmail`, each with its own `go.mod`, tied
together at dev time by the committed `go.work` (overriding each provider's
pinned `require` on the root module — no `replace` directives). A change to
the `Sender` interface, `Email`, or a sentinel error in the root needs the
corresponding change in every provider that implements or consumes it, in the
same PR; the root module's own tests won't catch a provider mismatch. When
adding a module, the `Makefile`'s `MODULES`/`SUB_MODULES` variables need
updating too.
