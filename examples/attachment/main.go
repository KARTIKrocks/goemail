package main

import (
	"context"
	"log"
	"os"
	"time"

	email "github.com/KARTIKrocks/goemail"
)

func main() {
	// Configure SMTP
	config := email.SMTPConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     587,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
		UseTLS:   true,
	}

	sender, err := email.NewSMTPSender(config)
	if err != nil {
		log.Fatalf("Failed to create sender: %v", err)
	}
	mailer := email.NewMailer(sender, config.From)

	// Read file to attach
	pdfData, err := os.ReadFile("invoice.pdf")
	if err != nil {
		mailer.Close()
		log.Fatalf("Failed to read file: %v", err)
	}

	// Create email with attachment
	emailMsg := email.NewEmail().
		SetFrom(config.From).
		AddTo("customer@example.com").
		SetSubject("Your Invoice - January 2024").
		SetHTMLBody(`
<!DOCTYPE html>
<html>
<body>
    <h2>Invoice Attached</h2>
    <p>Dear Customer,</p>
    <p>Please find attached your invoice for January 2024.</p>
    <p>Thank you for your business!</p>
    <p>Best regards,<br>Billing Department</p>
</body>
</html>
`).
		AddAttachment("invoice.pdf", "application/pdf", pdfData)

	// Build and validate
	builtEmail, err := emailMsg.Build()
	if err != nil {
		mailer.Close()
		log.Fatalf("Failed to build email: %v", err)
	}

	// Send
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	err = mailer.SendEmail(ctx, builtEmail)
	cancel()
	mailer.Close()
	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}

	log.Println("Email with attachment sent successfully!")
}
