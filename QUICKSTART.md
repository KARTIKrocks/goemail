# Quick Start Guide

Get started with goemail in 5 minutes!

## Installation

```bash
go get github.com/KARTIKrocks/goemail
```

## 1. Basic Setup

Create a simple email sender:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/KARTIKrocks/goemail"
)

func main() {
    // Configure SMTP (Gmail example)
    config := email.SMTPConfig{
        Host:     "smtp.gmail.com",
        Port:     587,
        Username: "your-email@gmail.com",
        Password: "your-app-password", // See note below
        From:     "your-email@gmail.com",
        UseTLS:   true,
    }

    // Create sender and mailer
    sender, err := email.NewSMTPSender(config)
    if err != nil {
        log.Fatal(err)
    }
    mailer := email.NewMailer(sender, config.From)
    defer mailer.Close()

    // Send email
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    err = mailer.Send(ctx,
        []string{"recipient@example.com"},
        "Hello from Go!",
        "This is my first email using goemail!",
    )

    if err != nil {
        log.Fatal(err)
    }

    log.Println("Email sent successfully!")
}
```

### Gmail App Password Setup

For Gmail, you need to use an App Password instead of your regular password:

1. Go to https://myaccount.google.com/apppasswords
2. Enable 2-Step Verification if not already enabled
3. Select "Mail" and generate a password
4. Use this generated password in your config

## 2. Using Environment Variables

Store credentials securely:

```bash
# .env file
export SMTP_HOST="smtp.gmail.com"
export SMTP_PORT="587"
export SMTP_USERNAME="your-email@gmail.com"
export SMTP_PASSWORD="your-app-password"
export SMTP_FROM="your-email@gmail.com"
```

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

## 3. HTML Emails

Send beautiful HTML emails:

```go
html := `
<!DOCTYPE html>
<html>
<body style="font-family: Arial, sans-serif;">
    <div style="max-width: 600px; margin: 0 auto; padding: 20px;">
        <h1 style="color: #4CAF50;">Welcome!</h1>
        <p>Thank you for signing up.</p>
        <a href="https://example.com"
           style="background: #4CAF50; color: white; padding: 10px 20px;
                  text-decoration: none; display: inline-block;">
            Get Started
        </a>
    </div>
</body>
</html>
`

err := mailer.SendHTML(ctx, []string{"user@example.com"}, "Welcome!", html)
```

## 4. Using Templates

Create reusable email templates:

```go
// Create template
tmpl := email.NewTemplate("welcome")
tmpl.SetSubject("Welcome {{.Name}}!")
tmpl.SetHTMLTemplate(`
<h1>Hello {{.Name}}!</h1>
<p>Thanks for joining {{.AppName}}!</p>
`)

// Register template
mailer.RegisterTemplate("welcome", tmpl)

// Send email
data := map[string]any{
    "Name":    "John",
    "AppName": "MyApp",
}
err := mailer.SendTemplate(ctx, []string{"john@example.com"}, "welcome", data)
```

## 5. With Attachments

Send files with your emails:

```go
// Read file
data, _ := os.ReadFile("invoice.pdf")

// Create email with attachment
email := email.NewEmail().
    SetFrom("billing@example.com").
    AddTo("customer@example.com").
    SetSubject("Your Invoice").
    SetBody("Please find your invoice attached.").
    AddAttachment("invoice.pdf", "application/pdf", data)

// Build and send
builtEmail, _ := email.Build()
err := mailer.SendEmail(ctx, builtEmail)
```

## 6. Testing Your Code

Use the mock sender for tests:

```go
func TestEmailService(t *testing.T) {
    // Create mock sender
    mock := email.NewMockSender()
    mailer := email.NewMailer(mock, "test@example.com")

    // Send an email
    ctx := context.Background()
    err := mailer.Send(ctx,
        []string{"user@example.com"},
        "Welcome John!",
        "Thanks for signing up!",
    )
    if err != nil {
        t.Fatalf("send failed: %v", err)
    }

    // Verify email was sent
    if mock.GetEmailCount() != 1 {
        t.Error("expected 1 email to be sent")
    }

    // Check email content
    sent := mock.GetLastEmail()
    if sent.Subject != "Welcome John!" {
        t.Errorf("wrong subject: %s", sent.Subject)
    }
}
```

## Common SMTP Providers

### Gmail

```go
email.SMTPConfig{
    Host: "smtp.gmail.com",
    Port: 587,
    UseTLS: true,
}
```

### SendGrid

```go
email.SMTPConfig{
    Host: "smtp.sendgrid.net",
    Port: 587,
    Username: "apikey",
    Password: "your-api-key",
    UseTLS: true,
}
```

### AWS SES

```go
email.SMTPConfig{
    Host: "email-smtp.us-east-1.amazonaws.com",
    Port: 587,
    UseTLS: true,
}
```

### Mailgun

```go
email.SMTPConfig{
    Host: "smtp.mailgun.org",
    Port: 587,
    UseTLS: true,
}
```

## Next Steps

- Check out the [examples/](examples/) directory for more use cases
- Read the full [README.md](README.md) for all features
- See [LOGGER_ADAPTERS.md](LOGGER_ADAPTERS.md) for logging integration
- Review [CONTRIBUTING.md](CONTRIBUTING.md) if you want to contribute

## Troubleshooting

### "Authentication failed"

- Make sure you're using an App Password for Gmail (not your regular password)
- Check that your username and password are correct
- Verify your SMTP host and port

### "Connection timeout"

- Check your firewall settings
- Ensure Port 587 (or 465) is open
- Try without VPN if you're using one

### "Invalid recipient"

- Verify email addresses are valid
- Check for typos in recipient addresses

## Getting Help

- 📖 Documentation: See README.md for full docs
- 🐛 Issues: https://github.com/KARTIKrocks/goemail/issues
- 💬 Discussions: https://github.com/KARTIKrocks/goemail/discussions

Happy emailing! 📧
