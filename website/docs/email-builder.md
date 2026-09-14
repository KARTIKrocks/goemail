---
id: email-builder
title: Email Builder
description: The chainable Email builder for recipients, bodies, attachments, custom headers, and validation.
---

# Email Builder

For messages that need anything beyond the four-argument `Send` helper —
multiple recipients, CC/BCC, custom headers, or attachments — use the
chainable `Email` builder. Every setter returns the same `*Email` so calls
can be fluently composed.

## Fields & Recipients

```go
msg := email.NewEmail().
    SetFrom("sender@example.com").
    AddTo("user1@example.com", "user2@example.com").
    AddCc("manager@example.com").
    AddBcc("archive@example.com").
    SetReplyTo("support@example.com").
    SetSubject("Important Update")
```

Recipients can be passed as bare addresses (`"alice@example.com"`) or with
display names (`"Alice <alice@example.com>"`). Each `AddTo`, `AddCc`, and
`AddBcc` accepts variadic arguments and is additive.

## Plain Text & HTML

Set either or both. When both are present goemail produces a
multipart/alternative message and the recipient's mail client picks the
best fit.

```go
msg.
    SetBody("Plain-text fallback for clients without HTML support.").
    SetHTMLBody("<h1>Hello</h1><p>This is the rich version.</p>")
```

## Attachments

```go
pdfData, err := os.ReadFile("invoice.pdf")
if err != nil {
    log.Fatal(err)
}

msg := email.NewEmail().
    SetFrom("billing@example.com").
    AddTo("customer@example.com").
    SetSubject("Your invoice").
    SetBody("Please find your invoice attached.").
    AddAttachment("invoice.pdf", "application/pdf", pdfData)
```

Use the correct MIME type for each attachment (`image/png`, `text/csv`,
`application/pdf`, etc.) — clients use it to render previews and pick
icons.

## Custom Headers

Add any RFC 5322 header. goemail validates header names and strips CR/LF
to prevent header injection from untrusted input.

```go
msg.
    AddHeader("X-Priority", "1").
    AddHeader("X-Campaign-ID", campaignID).
    AddHeader("List-Unsubscribe", "<https://example.com/unsubscribe?u=123>")
```

## Build & Validate

Call `Build` to validate the email before sending. `Build` checks that the
sender, at least one recipient, and a subject are present, and that all
addresses parse. `Mailer.SendEmail` calls `Build` for you.

```go
built, err := msg.Build()
if err != nil {
    return fmt.Errorf("invalid email: %w", err)
}

if err := mailer.SendEmail(ctx, built); err != nil {
    return err
}
```
