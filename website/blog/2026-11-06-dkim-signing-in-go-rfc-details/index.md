---
slug: dkim-signing-in-go-rfc-details
title: "DKIM Signing in Go: The RFC Details That Break Silently"
authors: [kartik]
tags: [go, email, dkim, security, smtp]
description: A DKIM signature that passes a quick Gmail check can still be wrong in ways a stricter verifier catches — oversigning, canonicalization mismatches, and duplicate-header ordering all fail silently instead of raising an error.
---

# DKIM Signing in Go: The RFC Details That Break Silently

DKIM has an unusual failure mode compared to most of what a mail library
gets wrong: there's no error to catch. A malformed signature doesn't throw
an exception during signing — it produces a perfectly well-formed
`DKIM-Signature` header that computes to the wrong hash, and the only
symptom is that some receiving mail servers silently mark the message as
unsigned or spammy while others — the more lenient ones — verify it fine
anyway. That asymmetry is exactly why it's worth going through [RFC
6376](https://www.rfc-editor.org/rfc/rfc6376) carefully instead of testing
against whichever inbox happens to be open.

{/* truncate */}

## What's actually being signed

[goemail](https://github.com/KARTIKrocks/goemail)'s `SignMessage` signs two
separate hashes that get combined into one signature: a hash of the
canonicalized body (`bh=` in the signature header), and a hash over the
canonicalized _selected headers plus the signature header itself minus its
`b=` value_. Getting the body hash right and the header hash wrong is a
completely plausible way to ship a signature that fails verification while
every manual spot-check of "does the body hash match" passes.

```go
canonBody := canonicalizeBody(body, config.bodyCanon())
bodyHash := sha256.Sum256([]byte(canonBody))
bodyHashB64 := base64.StdEncoding.EncodeToString(bodyHash[:])
```

That part's close to what you'd guess. The header side has three separate
details that are each individually easy to get wrong, and RFC 6376 is
precise about all three.

## Detail one: canonicalization is not whitespace trimming

"Relaxed" canonicalization sounds like "be a bit more lenient about
whitespace," which undersells what it actually requires. RFC 6376 §3.4.2
specifies: lowercase the header name, unfold continuation lines, collapse
internal whitespace runs to a single space, strip whitespace around the
colon, strip trailing whitespace. Each of those is a distinct
transformation, and skipping any one of them produces a canonicalized
string that differs from a compliant verifier's by exactly the bytes you
skipped — which is enough to break the hash match even when the
_meaning_ of the header obviously hasn't changed to a human reading it.

```go
func canonicalizeHeader(header string, algo Canonicalization) string {
    if algo == CanonicalizationSimple {
        return header
    }
    before, after, ok := strings.Cut(header, ":")
    name := strings.ToLower(strings.TrimSpace(before))
    value := after
    // Unfold: remove CRLF followed by whitespace
    value = strings.ReplaceAll(value, "\r\n\t", " ")
    value = strings.ReplaceAll(value, "\r\n ", " ")
    // Collapse whitespace runs to a single space
    var collapsed strings.Builder
    inWSP := false
    for _, c := range value {
        if c == ' ' || c == '\t' {
            if !inWSP {
                collapsed.WriteByte(' ')
                inWSP = true
            }
        } else {
            collapsed.WriteRune(c)
            inWSP = false
        }
    }
    result := strings.TrimSpace(collapsed.String())
    return name + ":" + result
}
```

Notice the unfold step runs _before_ the whitespace-collapse step, not
after. A folded header continuation (`Subject: long line\r\n continued`)
unfolds to `long line continued` with the fold replaced by a single space —
but if collapsing ran first, the literal `\r\n` bytes (which aren't `' '`
or `'\t'`) would pass through the collapse loop untouched and end up still
embedded in the output. Order isn't incidental here; the two transforms
only compose correctly in the sequence the RFC specifies.

Body canonicalization has its own version of this same trap: trailing
empty lines have to be stripped, but only from the _end_ of the body, and
the body still needs a single trailing CRLF added back if trimming removed
everything. `removeTrailingEmptyLines` and `canonicalizeBody` handle this
as a specific algorithm, not a regex — a body that canonicalizes to the
empty string still has to produce `"\r\n"`, not `""`, or the body hash
is computed over the wrong byte sequence.

## Detail two: which headers get signed, and in what order

The `h=` tag in a DKIM signature lists which headers were included in the
hash — but it's a list of _names_, and the actual values are consumed in a
specific order when a message has duplicate headers. RFC 6376 §5.4 requires
bottom-to-top consumption: if a message somehow has two `Subject` headers,
the signer and verifier both have to agree on using the _last_ occurrence
first. Get the order backwards relative to a verifier and the hash
mismatches on any message with duplicate headers — which most messages
don't have, so this bug can sit untested indefinitely and then break on
the one message that does.

```go
func (p *parsedHeaders) get(name string) (string, bool) {
    lname := strings.ToLower(name)
    entries := p.byName[lname]
    if len(entries) == 0 {
        return "", false
    }
    // Pop from the end (bottom-to-top per RFC 6376 §5.4)
    last := entries[len(entries)-1]
    p.byName[lname] = entries[:len(entries)-1]
    return last, true
}
```

There's a second, related rule hiding in the same section: **oversigning**.
`h=` can list a header name more times than the message actually has
occurrences of it, specifically to prevent an attacker from _adding_ a new
header after the message is signed (a forwarded message with an injected
second `Subject`, say) — the extra name in `h=` with no corresponding
header contributes the RFC's defined "null string" to the hash instead of
being skipped:

```go
for _, h := range presentHeaders {
    headerLine, ok := hdrs.get(h)
    if !ok {
        continue
    }
    dataToSign.WriteString(canonicalizeHeader(headerLine, config.headerCanon()))
    dataToSign.WriteString("\r\n")
}
```

goemail's implementation only signs headers that are actually present
(`presentHeaders` is pre-filtered), which is the simpler and correct choice
for _this_ library's threat model — but it's worth knowing oversigning
exists as a technique, because a from-scratch DKIM implementation that
tries to be clever about "sign every header name in my default list
whether or not it's there" without understanding the null-string rule will
produce signatures that don't match what any compliant verifier computes.

## Detail three: the signature header signs itself, almost

The `DKIM-Signature` header is itself part of what gets hashed — with the
`b=` tag's value treated as empty, since the signature obviously can't sign
itself. That means the hash has to be computed over the real header
_structure_ (all the tags, in order, with a trailing empty `b=`) before the
actual signature bytes exist:

```go
dkimHeader := "DKIM-Signature: " + dkimTag // dkimTag ends in "b="
dataToSign.WriteString(canonicalizeHeader(dkimHeader, config.headerCanon()))

signature, err := signData([]byte(dataToSign.String()), config)
sigB64 := base64.StdEncoding.EncodeToString(signature)
fullDKIMHeader := "DKIM-Signature: " + dkimTag + foldSignature(sigB64) + "\r\n"
```

This is also where the RSA-SHA256 and Ed25519-SHA256 algorithms genuinely
diverge, not just in key type. RFC 8463 §3 specifies that Ed25519-SHA256
signs the SHA-256 _digest_ of the canonicalized data, the same as
RSA-SHA256 — but `ed25519.Sign` in Go's standard library signs raw message
bytes by convention (Ed25519 doesn't pre-hash internally the way RSA-PKCS1
does), so getting this right for Ed25519 means hashing explicitly before
calling `Sign`, where the RSA path gets the hash-then-sign behavior for
free from `rsa.SignPKCS1v15`:

```go
func signData(data []byte, config *DKIMConfig) ([]byte, error) {
    switch key := config.PrivateKey.(type) {
    case *rsa.PrivateKey:
        hash := sha256.Sum256(data)
        return rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, hash[:])
    case ed25519.PrivateKey:
        // RFC 8463 §3: ed25519-sha256 signs the SHA-256 hash of the
        // canonicalized signed headers, not the raw data.
        hash := sha256.Sum256(data)
        return ed25519.Sign(key, hash[:]), nil
    default:
        return nil, fmt.Errorf("dkim: unsupported key type %T", key)
    }
}
```

Skip the explicit `sha256.Sum256` on the Ed25519 path and you get a
signature that's completely valid Ed25519 output, over the wrong input —
again, no error anywhere, just a signature that doesn't verify.

## Even the line-folding has a rule

RFC 6376 §3.7 limits header lines to 78 columns, and the base64 signature
gets folded well under that — goemail wraps at 72 characters per line,
leaving room for the leading whitespace continuation lines require — with
the fold happening _after_
encoding, at a column boundary the RFC specifies, not wherever happens to
be convenient:

```go
func foldSignature(sig string) string {
    var result strings.Builder
    const lineLen = 72 // leave room for leading whitespace
    for i := 0; i < len(sig); i += lineLen {
        end := min(i+lineLen, len(sig))
        if i == 0 {
            result.WriteString(sig[i:end])
        } else {
            result.WriteString("\r\n        " + sig[i:end])
        }
    }
    return result.String()
}
```

Getting this wrong doesn't break the signature — base64 folding is purely
cosmetic for the hash computation, since the verifier strips whitespace
before decoding `b=`. It's included here because it's the one place in
this list where getting it "wrong" is harmless, and knowing _which_ details
are cosmetic versus which ones silently break verification is as important
as getting the breaking ones right.

## Why this matters more than the code volume suggests

Every one of these details — canonicalization order, bottom-to-top
duplicate consumption, the Ed25519 explicit-hash requirement — produces
identical behavior to the _correct_ implementation on the overwhelming
majority of real messages: no duplicate headers, headers that don't need
unfolding, an RSA key instead of Ed25519. That's what makes DKIM bugs
dangerous in a way that most bugs aren't — the common case can't surface
them, and the first time a bug like this gets caught in production is when
a specific receiving server's strict verifier rejects a specific message
shape, with no earlier warning that anything was ever wrong.

```bash
go get github.com/KARTIKrocks/goemail
```

DKIM configuration is documented in the
[DKIM signing guide](https://kartikrocks.github.io/goemail/docs/dkim); the
full implementation, cross-referenced against RFC 6376 and RFC 8463
section numbers throughout, is in
[`dkim.go`](https://github.com/KARTIKrocks/goemail/blob/main/dkim.go).
