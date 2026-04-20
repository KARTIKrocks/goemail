package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	email "github.com/KARTIKrocks/goemail"
)

func main() {
	// Configure SMTP
	config := email.SMTPConfig{
		Host:         os.Getenv("SMTP_HOST"),
		Port:         587,
		Username:     os.Getenv("SMTP_USERNAME"),
		Password:     os.Getenv("SMTP_PASSWORD"),
		From:         os.Getenv("SMTP_FROM"),
		UseTLS:       true,
		RateLimit:    5, // Limit to 5 emails per second
		MaxRetries:   3, // Retry up to 3 times
		RetryDelay:   time.Second,
		RetryBackoff: 2.0,
	}

	sender, err := email.NewSMTPSender(config)
	if err != nil {
		log.Fatalf("Failed to create sender: %v", err)
	}
	mailer := email.NewMailer(sender, config.From)

	// Create newsletter template
	newsletter := email.NewTemplate("newsletter")
	newsletter.SetSubject("Weekly Newsletter - {{.Date}}")
	newsletter.SetHTMLTemplate(`
<!DOCTYPE html>
<html>
<body>
    <h1>Weekly Newsletter</h1>
    <p>Hi {{.Name}},</p>
    <p>Here's what's new this week:</p>
    <ul>
        <li>New feature releases</li>
        <li>Product updates</li>
        <li>Community highlights</li>
    </ul>
    <p><a href="{{.UnsubscribeLink}}">Unsubscribe</a></p>
</body>
</html>
`)

	mailer.RegisterTemplate("newsletter", newsletter)

	// List of subscribers
	subscribers := []struct {
		Name  string
		Email string
	}{
		{"Alice Johnson", "alice@example.com"},
		{"Bob Smith", "bob@example.com"},
		{"Carol Williams", "carol@example.com"},
		{"David Brown", "david@example.com"},
		{"Eve Davis", "eve@example.com"},
	}

	// Create batch of emails
	emails := make([]*email.Email, 0, len(subscribers))
	for _, sub := range subscribers {
		data := map[string]any{
			"Name":            sub.Name,
			"Date":            time.Now().Format("January 2, 2006"),
			"UnsubscribeLink": fmt.Sprintf("https://example.com/unsubscribe?email=%s", sub.Email),
		}

		emailMsg, err := newsletter.Render(data)
		if err != nil {
			log.Fatalf("Failed to render template: %v", err)
		}

		emailMsg.AddTo(sub.Email)
		emails = append(emails, emailMsg)
	}

	// Send batch with concurrency limit of 3
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

	log.Printf("Sending newsletter to %d subscribers...", len(subscribers))
	start := time.Now()

	err = mailer.SendBatch(ctx, emails, 3)
	cancel()
	mailer.Close()
	if err != nil {
		log.Fatalf("Batch send failed: %v", err)
	}

	duration := time.Since(start)
	log.Printf("Successfully sent %d emails in %s", len(emails), duration)
	log.Printf("Average: %.2f emails/second", float64(len(emails))/duration.Seconds())
}
