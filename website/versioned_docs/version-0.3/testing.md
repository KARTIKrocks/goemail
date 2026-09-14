---
id: testing
title: Testing
description: Test email-sending code without a real SMTP server using the in-memory MockSender.
---

# Testing

Testing email-sending code without a real SMTP server is one of the main
reasons goemail keeps the `Sender` interface small. `NewMockSender`
returns an in-memory sender that records every email it receives, with
helpers to inspect and assert on them.

## MockSender

```go
func TestSendWelcomeEmail(t *testing.T) {
    mock := email.NewMockSender()
    mailer := email.NewMailer(mock, "test@example.com")

    err := mailer.Send(context.Background(),
        []string{"alice@example.com"},
        "Welcome",
        "Welcome to MyApp!",
    )
    if err != nil {
        t.Fatalf("send failed: %v", err)
    }

    if got := mock.GetEmailCount(); got != 1 {
        t.Fatalf("got %d emails, want 1", got)
    }
}
```

For tests that need to simulate failure, configure the mock to return an
error:

```go
mock := email.NewMockSender()
mock.SetError(errors.New("simulated SMTP failure"))

err := mailer.Send(ctx, to, subject, body)
// err is non-nil — assert your retry / fallback logic
```

## Inspecting Sent Mail

Use `GetLastEmail`, `GetEmails`, or `GetEmailCount` to make assertions
about subject, recipients, body, attachments, or headers:

```go
sent := mock.GetLastEmail()

if sent.Subject != "Welcome" {
    t.Errorf("Subject = %q, want %q", sent.Subject, "Welcome")
}

if len(sent.To) != 1 || sent.To[0] != "alice@example.com" {
    t.Errorf("To = %v, want [alice@example.com]", sent.To)
}

if !strings.Contains(sent.Body, "Welcome") {
    t.Errorf("Body missing greeting: %s", sent.Body)
}

// Reset between subtests
mock.Reset()
```
