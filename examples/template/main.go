package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"time"

	email "github.com/KARTIKrocks/goemail"
)

func main() {
	// Setup logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Configure SMTP with logging
	config := email.SMTPConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     587,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
		UseTLS:   true,
		Logger:   email.NewSlogLogger(logger),
	}

	sender, err := email.NewSMTPSender(config)
	if err != nil {
		log.Fatalf("Failed to create sender: %v", err)
	}
	mailer := email.NewMailer(sender, config.From)

	// Create welcome template
	welcomeTmpl := email.NewTemplate("welcome")
	welcomeTmpl.SetSubject("Welcome to {{.AppName}}, {{.Name}}!")

	welcomeTmpl.SetHTMLTemplate(`
<!DOCTYPE html>
<html>
<head>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background: #4CAF50; color: white; padding: 20px; text-align: center; }
        .content { padding: 20px; background: #f9f9f9; }
        .button { background: #4CAF50; color: white; padding: 10px 20px; text-decoration: none; display: inline-block; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Welcome to {{.AppName}}!</h1>
        </div>
        <div class="content">
            <h2>Hello {{.Name}},</h2>
            <p>Thank you for signing up! We're excited to have you on board.</p>
            <p>To get started, please verify your email address:</p>
            <a href="{{.VerifyLink}}" class="button">Verify Email</a>
            <p>If you didn't create this account, you can safely ignore this email.</p>
            <p>Best regards,<br>The {{.AppName}} Team</p>
        </div>
    </div>
</body>
</html>
`)

	// Also provide plain text version
	welcomeTmpl.SetTextTemplate(`
Welcome to {{.AppName}}, {{.Name}}!

Thank you for signing up! We're excited to have you on board.

To get started, please verify your email address by clicking this link:
{{.VerifyLink}}

If you didn't create this account, you can safely ignore this email.

Best regards,
The {{.AppName}} Team
`)

	// Register template
	mailer.RegisterTemplate("welcome", welcomeTmpl)

	// Send welcome email
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	data := map[string]any{
		"Name":       "John Doe",
		"AppName":    "MyAwesomeApp",
		"VerifyLink": "https://example.com/verify?token=abc123xyz",
	}

	err = mailer.SendTemplate(ctx, []string{"john.doe@example.com"}, "welcome", data)
	cancel()
	if err != nil {
		log.Fatalf("Failed to send welcome email: %v", err)
	}

	log.Println("Welcome email sent successfully!")
}
