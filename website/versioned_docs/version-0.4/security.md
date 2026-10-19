---
id: security
title: Security
description: Operational and protocol-level habits that prevent the most common email-related security problems.
---

# Security

A few small habits prevent the most common email-related security
problems. goemail enforces the protocol-level ones automatically; the
rest are operational.

## Environment Variables

Never check SMTP passwords or API keys into source control. Read them
from the environment or a secret manager at startup.

```go
config := email.SMTPConfig{
    Host:     os.Getenv("SMTP_HOST"),
    Port:     587,
    Username: os.Getenv("SMTP_USERNAME"),
    Password: os.Getenv("SMTP_PASSWORD"),
    From:     os.Getenv("SMTP_FROM"),
    UseTLS:   true,
}
```

## Header Injection

If user input flows into a recipient list, subject, or custom header, an
attacker who controls that input can try to inject extra headers (`Bcc:`,
`Reply-To:`) by embedding CR/LF. goemail validates header names against
RFC 5322 and strips CR/LF from header values automatically — `AddHeader`
returns nothing for invalid input rather than silently corrupting the
message.

Recipient and subject validation happens in `Email.Build`:

```go
built, err := email.NewEmail().
    SetFrom("noreply@example.com").
    AddTo(userSuppliedAddress). // validated below
    SetSubject(userSuppliedSubject).
    SetBody("...").
    Build()
if err != nil {
    // Reject the request — don't try to "clean up" the input
    return fmt.Errorf("invalid email: %w", err)
}
```

## App Passwords

Most consumer mail providers no longer accept account passwords for
SMTP. Generate a scoped **App Password**:

- **Gmail:** enable 2-Step Verification, then visit
  [myaccount.google.com/apppasswords](https://myaccount.google.com/apppasswords).
- **Yahoo / iCloud:** generate from account security settings.
- **Microsoft 365:** use OAuth 2.0 instead — basic SMTP auth is being
  phased out.

Treat App Passwords as production credentials: rotate them on suspected
leak, scope them per environment, and store them only in your secret
manager.
