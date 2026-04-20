package main

import (
	"context"
	"strings"
	"testing"

	email "github.com/KARTIKrocks/goemail"
)

// Example service that sends emails
type UserService struct {
	mailer *email.Mailer
}

func NewUserService(mailer *email.Mailer) *UserService {
	return &UserService{mailer: mailer}
}

func (s *UserService) RegisterUser(ctx context.Context, email, name string) error {
	// Business logic here...

	// Send welcome email
	return s.mailer.Send(ctx,
		[]string{email},
		"Welcome to Our Service!",
		"Thank you for registering, "+name+"!",
	)
}

func (s *UserService) SendPasswordReset(ctx context.Context, emailAddr, resetLink string) error {
	data := map[string]any{
		"ResetLink": resetLink,
	}
	return s.mailer.SendTemplate(ctx, []string{emailAddr}, "password_reset", data)
}

// Test using mock sender
func TestUserService_RegisterUser(t *testing.T) {
	// Create mock sender
	mock := email.NewMockSender()
	mailer := email.NewMailer(mock, "noreply@example.com")

	// Create service with mock
	service := NewUserService(mailer)

	// Test registration
	ctx := context.Background()
	err := service.RegisterUser(ctx, "newuser@example.com", "John Doe")
	if err != nil {
		t.Fatalf("RegisterUser failed: %v", err)
	}

	// Verify email was sent
	if mock.GetEmailCount() != 1 {
		t.Errorf("expected 1 email, got %d", mock.GetEmailCount())
	}

	// Check email content
	sentEmail := mock.GetLastEmail()
	if sentEmail == nil {
		t.Fatal("no email was sent")
	}

	if sentEmail.To[0] != "newuser@example.com" {
		t.Errorf("expected email to newuser@example.com, got %s", sentEmail.To[0])
	}

	if sentEmail.Subject != "Welcome to Our Service!" {
		t.Errorf("unexpected subject: %s", sentEmail.Subject)
	}

	// Verify email contains user's name
	if !strings.Contains(sentEmail.Body, "John Doe") {
		t.Errorf("email body should contain user name")
	}
}

func TestUserService_SendPasswordReset(t *testing.T) {
	// Create mock with custom send function to test error handling
	mock := email.NewMockSender()
	mailer := email.NewMailer(mock, "noreply@example.com")

	// Register password reset template
	tmpl := email.NewTemplate("password_reset")
	tmpl.SetSubject("Password Reset Request")
	tmpl.SetHTMLTemplate(`
		<h1>Reset Your Password</h1>
		<p>Click here: <a href="{{.ResetLink}}">Reset</a></p>
	`)
	mailer.RegisterTemplate("password_reset", tmpl)

	service := NewUserService(mailer)

	// Test password reset
	ctx := context.Background()
	err := service.SendPasswordReset(ctx, "user@example.com", "https://example.com/reset/abc123")
	if err != nil {
		t.Fatalf("SendPasswordReset failed: %v", err)
	}

	// Verify
	if mock.GetEmailCount() != 1 {
		t.Errorf("expected 1 email, got %d", mock.GetEmailCount())
	}

	sentEmail := mock.GetLastEmail()
	if !strings.Contains(sentEmail.HTMLBody, "Reset Your Password") {
		t.Error("email should contain reset instructions")
	}

	if !strings.Contains(sentEmail.HTMLBody, "abc123") {
		t.Error("email should contain reset token")
	}
}

func TestMockSender_GetEmailsTo(t *testing.T) {
	mock := email.NewMockSender()
	mailer := email.NewMailer(mock, "noreply@example.com")

	ctx := context.Background()

	// Send to different recipients
	mailer.Send(ctx, []string{"alice@example.com"}, "Test 1", "Body 1")
	mailer.Send(ctx, []string{"bob@example.com"}, "Test 2", "Body 2")
	mailer.Send(ctx, []string{"alice@example.com"}, "Test 3", "Body 3")

	// Get emails to specific recipient
	aliceEmails := mock.GetEmailsTo("alice@example.com")
	if len(aliceEmails) != 2 {
		t.Errorf("expected 2 emails to alice, got %d", len(aliceEmails))
	}

	bobEmails := mock.GetEmailsTo("bob@example.com")
	if len(bobEmails) != 1 {
		t.Errorf("expected 1 email to bob, got %d", len(bobEmails))
	}
}

func TestMockSender_Reset(t *testing.T) {
	mock := email.NewMockSender()
	mailer := email.NewMailer(mock, "noreply@example.com")

	ctx := context.Background()

	// Send some emails
	mailer.Send(ctx, []string{"user@example.com"}, "Test", "Body")
	mailer.Send(ctx, []string{"user@example.com"}, "Test", "Body")

	if mock.GetEmailCount() != 2 {
		t.Errorf("expected 2 emails, got %d", mock.GetEmailCount())
	}

	// Reset
	mock.Reset()

	if mock.GetEmailCount() != 0 {
		t.Errorf("expected 0 emails after reset, got %d", mock.GetEmailCount())
	}
}
