package email

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestEmailBuilder(t *testing.T) {
	tests := []struct {
		name    string
		builder func() *Email
		wantErr bool
		errType error
	}{
		{
			name: "valid email",
			builder: func() *Email {
				return NewEmail().
					SetFrom("sender@example.com").
					AddTo("recipient@example.com").
					SetSubject("Test").
					SetBody("Body")
			},
			wantErr: false,
		},
		{
			name: "missing sender",
			builder: func() *Email {
				return NewEmail().
					AddTo("recipient@example.com").
					SetSubject("Test").
					SetBody("Body")
			},
			wantErr: true,
			errType: ErrNoSender,
		},
		{
			name: "missing recipients",
			builder: func() *Email {
				return NewEmail().
					SetFrom("sender@example.com").
					SetSubject("Test").
					SetBody("Body")
			},
			wantErr: true,
			errType: ErrNoRecipients,
		},
		{
			name: "missing subject",
			builder: func() *Email {
				return NewEmail().
					SetFrom("sender@example.com").
					AddTo("recipient@example.com").
					SetBody("Body")
			},
			wantErr: true,
			errType: ErrNoSubject,
		},
		{
			name: "missing body",
			builder: func() *Email {
				return NewEmail().
					SetFrom("sender@example.com").
					AddTo("recipient@example.com").
					SetSubject("Test")
			},
			wantErr: true,
			errType: ErrNoBody,
		},
		{
			name: "invalid email address",
			builder: func() *Email {
				return NewEmail().
					SetFrom("invalid-email").
					AddTo("recipient@example.com").
					SetSubject("Test").
					SetBody("Body")
			},
			wantErr: true,
		},
		{
			name: "header injection in subject",
			builder: func() *Email {
				return NewEmail().
					SetFrom("sender@example.com").
					AddTo("recipient@example.com").
					SetSubject("Test\r\nBcc: hacker@example.com").
					SetBody("Body")
			},
			wantErr: true,
			errType: ErrInvalidHeader,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email, err := tt.builder().Build()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if tt.errType != nil && !errors.Is(err, tt.errType) {
					t.Errorf("expected error type %v, got %v", tt.errType, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if email == nil {
					t.Error("expected email, got nil")
				}
			}
		})
	}
}

func TestEmailHeaderValidation(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		value   string
		wantErr bool
	}{
		{"valid header", "X-Custom", "value", false},
		{"CRLF in key", "X-Custom\r\n", "value", true},
		{"CRLF in value", "X-Custom", "value\r\nhacker", true},
		{"colon in key", "X:Custom", "value", true},
		{"empty value", "X-Custom", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			email := NewEmail().
				SetFrom("sender@example.com").
				AddTo("recipient@example.com").
				SetSubject("Test").
				SetBody("Body").
				AddHeader(tt.key, tt.value)
			_, err := email.Build()
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMockSender(t *testing.T) {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	ctx := context.Background()

	// Send first email
	err := mailer.Send(ctx, []string{"user1@example.com"}, "Test 1", "Body 1")
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	// Send second email
	err = mailer.Send(ctx, []string{"user2@example.com"}, "Test 2", "Body 2")
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	// Verify count
	if count := mock.GetEmailCount(); count != 2 {
		t.Errorf("expected 2 emails, got %d", count)
	}

	// Verify last email
	last := mock.GetLastEmail()
	if last == nil {
		t.Fatal("last email is nil")
	}
	if last.Subject != "Test 2" {
		t.Errorf("expected subject 'Test 2', got %q", last.Subject)
	}

	// Verify emails to specific recipient
	emails := mock.GetEmailsTo("user1@example.com")
	if len(emails) != 1 {
		t.Errorf("expected 1 email to user1, got %d", len(emails))
	}

	// Reset and verify
	mock.Reset()
	if count := mock.GetEmailCount(); count != 0 {
		t.Errorf("expected 0 emails after reset, got %d", count)
	}
}

func TestMockSenderCustomFunction(t *testing.T) {
	mock := NewMockSender()
	customErr := errors.New("custom send error")
	mock.SetSendFunc(func(_ context.Context, _ *Email) error {
		return customErr
	})

	mailer := NewMailer(mock, "test@example.com")
	ctx := context.Background()

	err := mailer.Send(ctx, []string{"user@example.com"}, "Test", "Body")
	if !errors.Is(err, customErr) {
		t.Errorf("expected custom error, got %v", err)
	}
}

func TestTemplate(t *testing.T) {
	tmpl := NewTemplate("test")
	tmpl.SetSubject("Hello {{.Name}}")
	if _, err := tmpl.SetTextTemplate("Plain text for {{.Name}}"); err != nil {
		t.Fatal(err)
	}
	if _, err := tmpl.SetHTMLTemplate("<h1>HTML for {{.Name}}</h1>"); err != nil {
		t.Fatal(err)
	}

	data := map[string]any{
		"Name": "John",
	}

	email, err := tmpl.Render(data)
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	if email.Subject != "Hello John" {
		t.Errorf("expected subject 'Hello John', got %q", email.Subject)
	}

	if !strings.Contains(email.Body, "John") {
		t.Errorf("expected body to contain 'John', got %q", email.Body)
	}

	if !strings.Contains(email.HTMLBody, "John") {
		t.Errorf("expected HTML body to contain 'John', got %q", email.HTMLBody)
	}
}

func TestMailerTemplate(t *testing.T) {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	// Register template
	tmpl := NewTemplate("welcome")
	tmpl.SetSubject("Welcome {{.Name}}")
	if _, err := tmpl.SetHTMLTemplate("<h1>Welcome {{.Name}}</h1>"); err != nil {
		t.Fatal(err)
	}
	mailer.RegisterTemplate("welcome", tmpl)

	// Send using template
	ctx := context.Background()
	data := map[string]any{"Name": "Alice"}
	err := mailer.SendTemplate(ctx, []string{"alice@example.com"}, "welcome", data)
	if err != nil {
		t.Fatalf("send template failed: %v", err)
	}

	// Verify
	if mock.GetEmailCount() != 1 {
		t.Errorf("expected 1 email, got %d", mock.GetEmailCount())
	}

	email := mock.GetLastEmail()
	if email.Subject != "Welcome Alice" {
		t.Errorf("expected subject 'Welcome Alice', got %q", email.Subject)
	}
}

func TestMailerTemplateNotFound(t *testing.T) {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	ctx := context.Background()
	err := mailer.SendTemplate(ctx, []string{"user@example.com"}, "nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent template")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' error, got %v", err)
	}
}

func TestBatchSending(t *testing.T) {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	emails := []*Email{
		NewEmail().AddTo("user1@example.com").SetSubject("Test 1").SetBody("Body 1"),
		NewEmail().AddTo("user2@example.com").SetSubject("Test 2").SetBody("Body 2"),
		NewEmail().AddTo("user3@example.com").SetSubject("Test 3").SetBody("Body 3"),
	}

	ctx := context.Background()
	err := mailer.SendBatch(ctx, emails, 2)
	if err != nil {
		t.Fatalf("batch send failed: %v", err)
	}

	if count := mock.GetEmailCount(); count != 3 {
		t.Errorf("expected 3 emails, got %d", count)
	}
}

func TestBatchSendingWithInvalidEmail(t *testing.T) {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	emails := []*Email{
		NewEmail().AddTo("user1@example.com").SetSubject("Test 1").SetBody("Body 1"),
		NewEmail().AddTo("invalid").SetSubject("Test 2").SetBody("Body 2"), // Invalid email
	}

	ctx := context.Background()
	err := mailer.SendBatch(ctx, emails, 2)
	if err == nil {
		t.Error("expected error for invalid email in batch")
	}

	// No emails should be sent if validation fails
	if count := mock.GetEmailCount(); count != 0 {
		t.Errorf("expected 0 emails when batch validation fails, got %d", count)
	}
}

func TestContextCancellation(t *testing.T) {
	mock := NewMockSender()
	mock.SetSendFunc(func(ctx context.Context, _ *Email) error {
		// Simulate slow send and check context
		select {
		case <-time.After(100 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	mailer := NewMailer(mock, "test@example.com")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	err := mailer.Send(ctx, []string{"user@example.com"}, "Test", "Body")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected context.DeadlineExceeded, got %v", err)
	}
}

func TestEmailWithAttachment(t *testing.T) {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	pdfData := []byte("fake pdf data")
	email := NewEmail().
		AddTo("user@example.com").
		SetSubject("Document").
		SetBody("See attachment").
		AddAttachment("document.pdf", "application/pdf", pdfData)

	ctx := context.Background()
	err := mailer.SendEmail(ctx, email)
	if err != nil {
		t.Fatalf("send with attachment failed: %v", err)
	}

	sent := mock.GetLastEmail()
	if len(sent.Attachments) != 1 {
		t.Errorf("expected 1 attachment, got %d", len(sent.Attachments))
	}

	if sent.Attachments[0].Filename != "document.pdf" {
		t.Errorf("expected filename 'document.pdf', got %q", sent.Attachments[0].Filename)
	}
}

func TestMultipleRecipients(t *testing.T) {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	email := NewEmail().
		AddTo("user1@example.com", "user2@example.com").
		AddCc("manager@example.com").
		AddBcc("archive@example.com").
		SetSubject("Team Update").
		SetBody("Important message")

	ctx := context.Background()
	err := mailer.SendEmail(ctx, email)
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	sent := mock.GetLastEmail()
	if len(sent.To) != 2 {
		t.Errorf("expected 2 To recipients, got %d", len(sent.To))
	}
	if len(sent.Cc) != 1 {
		t.Errorf("expected 1 Cc recipient, got %d", len(sent.Cc))
	}
	if len(sent.Bcc) != 1 {
		t.Errorf("expected 1 Bcc recipient, got %d", len(sent.Bcc))
	}
}

func TestHTMLAndPlainTextBody(t *testing.T) {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	email := NewEmail().
		AddTo("user@example.com").
		SetSubject("Test").
		SetBody("Plain text body").
		SetHTMLBody("<h1>HTML body</h1>")

	ctx := context.Background()
	err := mailer.SendEmail(ctx, email)
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}

	sent := mock.GetLastEmail()
	if sent.Body != "Plain text body" {
		t.Errorf("unexpected plain text body: %q", sent.Body)
	}
	if sent.HTMLBody != "<h1>HTML body</h1>" {
		t.Errorf("unexpected HTML body: %q", sent.HTMLBody)
	}
}

func TestSMTPConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  SMTPConfig
		wantErr bool
	}{
		{
			name: "valid config",
			config: SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "user",
				Password: "pass",
			},
			wantErr: false,
		},
		{
			name: "missing host",
			config: SMTPConfig{
				Port:     587,
				Username: "user",
				Password: "pass",
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			config: SMTPConfig{
				Host:     "smtp.example.com",
				Port:     70000,
				Username: "user",
				Password: "pass",
			},
			wantErr: true,
		},
		{
			name: "username without password",
			config: SMTPConfig{
				Host:     "smtp.example.com",
				Port:     587,
				Username: "user",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// --- Error type tests ---

func TestErrorType(t *testing.T) {
	inner := errors.New("connection refused")
	emailErr := &Error{
		Op:   "send",
		From: "sender@example.com",
		To:   []string{"recipient@example.com"},
		Err:  inner,
	}

	// Test Error() string
	errStr := emailErr.Error()
	if !strings.Contains(errStr, "send") {
		t.Errorf("error string should contain operation, got %q", errStr)
	}
	if !strings.Contains(errStr, "sender@example.com") {
		t.Errorf("error string should contain from address, got %q", errStr)
	}
	if !strings.Contains(errStr, "recipient@example.com") {
		t.Errorf("error string should contain to address, got %q", errStr)
	}

	// Test Unwrap
	if !errors.Is(emailErr, inner) {
		t.Error("Unwrap should return inner error")
	}
}

func TestErrorTypeNoRecipients(t *testing.T) {
	emailErr := &Error{
		Op:   "validate",
		From: "sender@example.com",
		Err:  ErrNoRecipients,
	}

	errStr := emailErr.Error()
	if strings.Contains(errStr, "to=") {
		t.Errorf("error string should not contain to when empty, got %q", errStr)
	}
}

// --- buildMessage tests ---

// assertMessageContains is a test helper that checks a message string contains all expected substrings.
func assertMessageContains(t *testing.T, msg string, expected []string) {
	t.Helper()
	for _, s := range expected {
		if !strings.Contains(msg, s) {
			t.Errorf("message should contain %q", s)
		}
	}
}

func TestBuildMessage(t *testing.T) {
	sender, err := NewSMTPSender(SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
	})
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	t.Run("plain text only", func(t *testing.T) {
		email := &Email{
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Subject: "Test Subject",
			Body:    "Hello World",
			Headers: make(map[string]string),
		}

		msg, err := sender.buildMessage(email)
		if err != nil {
			t.Fatalf("buildMessage failed: %v", err)
		}

		assertMessageContains(t, string(msg), []string{
			"From: sender@example.com",
			"Subject: Test Subject",
			"Content-Type: text/plain; charset=UTF-8",
			"Content-Transfer-Encoding: quoted-printable",
			"MIME-Version: 1.0",
			"Message-ID:",
			"Date:",
		})
	})

	t.Run("HTML only", func(t *testing.T) {
		email := &Email{
			From:     "sender@example.com",
			To:       []string{"recipient@example.com"},
			Subject:  "Test",
			HTMLBody: "<h1>Hello</h1>",
			Headers:  make(map[string]string),
		}

		msg, err := sender.buildMessage(email)
		if err != nil {
			t.Fatalf("buildMessage failed: %v", err)
		}

		assertMessageContains(t, string(msg), []string{
			"Content-Type: text/html; charset=UTF-8",
		})
	})

	t.Run("multipart alternative", func(t *testing.T) {
		email := &Email{
			From:     "sender@example.com",
			To:       []string{"recipient@example.com"},
			Subject:  "Test",
			Body:     "Plain text",
			HTMLBody: "<h1>HTML</h1>",
			Headers:  make(map[string]string),
		}

		msg, err := sender.buildMessage(email)
		if err != nil {
			t.Fatalf("buildMessage failed: %v", err)
		}

		msgStr := string(msg)
		assertMessageContains(t, msgStr, []string{
			"multipart/alternative",
		})
		// Without attachments, the outer type should be alternative, not mixed.
		if strings.Contains(msgStr, "multipart/mixed") {
			t.Error("text+HTML without attachments should use multipart/alternative, not multipart/mixed")
		}
	})

	t.Run("with attachments", func(t *testing.T) {
		email := &Email{
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Subject: "Test",
			Body:    "See attached",
			Headers: make(map[string]string),
			Attachments: []Attachment{
				{Filename: "test.txt", ContentType: "text/plain", Data: []byte("hello")},
			},
		}

		msg, err := sender.buildMessage(email)
		if err != nil {
			t.Fatalf("buildMessage failed: %v", err)
		}

		assertMessageContains(t, string(msg), []string{
			"multipart/mixed",
			"Content-Disposition: attachment; filename=\"test.txt\"",
			"Content-Transfer-Encoding: base64",
		})
	})

	t.Run("with Cc and ReplyTo", func(t *testing.T) {
		email := &Email{
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Cc:      []string{"cc@example.com"},
			ReplyTo: "replyto@example.com",
			Subject: "Test",
			Body:    "Hello",
			Headers: make(map[string]string),
		}

		msg, err := sender.buildMessage(email)
		if err != nil {
			t.Fatalf("buildMessage failed: %v", err)
		}

		assertMessageContains(t, string(msg), []string{
			"Cc: cc@example.com",
			"Reply-To: replyto@example.com",
		})
	})

	t.Run("with custom headers", func(t *testing.T) {
		email := &Email{
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Subject: "Test",
			Body:    "Hello",
			Headers: map[string]string{
				"X-Custom":   "value1",
				"X-Priority": "1",
				"Message-ID": "<custom-id@example.com>",
			},
		}

		msg, err := sender.buildMessage(email)
		if err != nil {
			t.Fatalf("buildMessage failed: %v", err)
		}

		msgStr := string(msg)
		assertMessageContains(t, msgStr, []string{"X-Custom: value1"})
		if strings.Count(msgStr, "Message-ID:") != 1 {
			t.Error("should only have one Message-ID header")
		}
	})

	t.Run("non-ASCII subject", func(t *testing.T) {
		email := &Email{
			From:    "sender@example.com",
			To:      []string{"recipient@example.com"},
			Subject: "Héllo Wörld",
			Body:    "Hello",
			Headers: make(map[string]string),
		}

		msg, err := sender.buildMessage(email)
		if err != nil {
			t.Fatalf("buildMessage failed: %v", err)
		}

		assertMessageContains(t, string(msg), []string{"=?UTF-8?"})
	})
}

// --- wrapText tests ---

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected string
	}{
		{"short text", "hello", 76, "hello"},
		{"exact width", "abcdef", 6, "abcdef"},
		{"needs wrapping", "abcdefghij", 4, "abcd\r\nefgh\r\nij"},
		{"empty", "", 76, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapText(tt.text, tt.width)
			if got != tt.expected {
				t.Errorf("wrapText(%q, %d) = %q, want %q", tt.text, tt.width, got, tt.expected)
			}
		})
	}
}

// --- generateUniqueID tests ---

func TestGenerateUniqueID(t *testing.T) {
	id1 := generateUniqueID()
	id2 := generateUniqueID()

	if id1 == "" {
		t.Error("generateUniqueID should not return empty string")
	}
	if len(id1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("expected 32 char hex string, got %d chars: %q", len(id1), id1)
	}
	if id1 == id2 {
		t.Error("two calls to generateUniqueID should return different values")
	}
}

// --- sanitizeFilename tests ---

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"normal", "document.pdf", "document.pdf"},
		{"with quotes", `file"name.pdf`, "file_name.pdf"},
		{"with newlines", "file\r\nname.pdf", "filename.pdf"},
		{"with null", "file\x00name.pdf", "filename.pdf"},
		{"with slashes", "path/to\\file.pdf", "path_to_file.pdf"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// --- encodeHeader tests ---

func TestEncodeHeader(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		encoded bool
	}{
		{"ASCII only", "Hello World", false},
		{"non-ASCII", "Héllo Wörld", true},
		{"emoji", "Hello 🎉", true},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := encodeHeader(tt.input)
			if tt.encoded {
				if !strings.Contains(got, "=?UTF-8?") {
					t.Errorf("expected RFC 2047 encoding, got %q", got)
				}
			} else {
				if got != tt.input {
					t.Errorf("expected unchanged %q, got %q", tt.input, got)
				}
			}
		})
	}
}

// --- quotedPrintableEncode tests ---

func TestQuotedPrintableEncode(t *testing.T) {
	// ASCII text should pass through mostly unchanged
	got := quotedPrintableEncode("Hello World")
	if !strings.Contains(got, "Hello World") {
		t.Errorf("ASCII text should be preserved, got %q", got)
	}

	// Non-ASCII should be encoded
	got = quotedPrintableEncode("Héllo")
	if !strings.Contains(got, "=") {
		t.Errorf("non-ASCII should be encoded, got %q", got)
	}
}

// --- NewSMTPSender tests ---

func TestNewSMTPSender(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		sender, err := NewSMTPSender(SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sender == nil {
			t.Fatal("sender should not be nil")
		}
	})

	t.Run("invalid config", func(t *testing.T) {
		_, err := NewSMTPSender(SMTPConfig{
			Port: 587,
		})
		if err == nil {
			t.Error("expected error for missing host")
		}
	})

	t.Run("defaults are set", func(t *testing.T) {
		sender, err := NewSMTPSender(SMTPConfig{
			Host: "smtp.example.com",
			Port: 587,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sender.config.Timeout != DefaultTimeout {
			t.Errorf("expected default timeout %v, got %v", DefaultTimeout, sender.config.Timeout)
		}
		if sender.config.MaxRetries != DefaultMaxRetries {
			t.Errorf("expected default max retries %d, got %d", DefaultMaxRetries, sender.config.MaxRetries)
		}
		if sender.config.RetryDelay != DefaultRetryDelay {
			t.Errorf("expected default retry delay %v, got %v", DefaultRetryDelay, sender.config.RetryDelay)
		}
		if sender.config.RetryBackoff != DefaultRetryBackoff {
			t.Errorf("expected default retry backoff %f, got %f", DefaultRetryBackoff, sender.config.RetryBackoff)
		}
		if sender.config.RateLimit != DefaultRateLimit {
			t.Errorf("expected default rate limit %d, got %d", DefaultRateLimit, sender.config.RateLimit)
		}
	})

	t.Run("with logger", func(t *testing.T) {
		logger := NewSlogLogger(slog.New(slog.NewTextHandler(os.Stderr, nil)))
		sender, err := NewSMTPSender(SMTPConfig{
			Host:   "smtp.example.com",
			Port:   587,
			Logger: logger,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sender.logger == nil {
			t.Error("logger should be set")
		}
	})
}

// --- SlogLogger tests ---

func TestSlogLogger(t *testing.T) {
	logger := NewSlogLogger(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	// These should not panic
	logger.Debug("debug message", "key", "value")
	logger.Info("info message", "key", "value")
	logger.Warn("warn message", "key", "value")
	logger.Error("error message", "key", "value")

	// With should return a new logger
	withLogger := logger.With("requestID", "123")
	if withLogger == nil {
		t.Error("With should return a non-nil logger")
	}
	withLogger.Info("with context")
}

// --- NoOpLogger tests ---

func TestNoOpLogger(t *testing.T) {
	logger := NoOpLogger{}

	// These should not panic
	logger.Debug("debug")
	logger.Info("info")
	logger.Warn("warn")
	logger.Error("error")

	withLogger := logger.With("key", "value")
	if withLogger == nil {
		t.Error("With should return a non-nil logger")
	}
}

// --- LoadTemplateFromFile tests ---

func TestLoadTemplateFromFile(t *testing.T) {
	t.Run("valid file", func(t *testing.T) {
		// Create a temp file
		dir := t.TempDir()
		path := filepath.Join(dir, "template.html")
		if err := os.WriteFile(path, []byte("<h1>Hello {{.Name}}</h1>"), 0644); err != nil {
			t.Fatal(err)
		}

		tmpl, err := LoadTemplateFromFile("test", path)
		if err != nil {
			t.Fatalf("LoadTemplateFromFile failed: %v", err)
		}

		email, err := tmpl.Render(map[string]any{"Name": "World"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}

		if !strings.Contains(email.HTMLBody, "Hello World") {
			t.Errorf("expected 'Hello World' in HTML body, got %q", email.HTMLBody)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		_, err := LoadTemplateFromFile("test", "/nonexistent/path.html")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})
}

// --- Template error handling tests ---

func TestTemplateErrors(t *testing.T) {
	t.Run("invalid text template", func(t *testing.T) {
		tmpl := NewTemplate("test")
		_, err := tmpl.SetTextTemplate("{{.Invalid")
		if err == nil {
			t.Error("expected error for invalid template syntax")
		}
	})

	t.Run("invalid HTML template", func(t *testing.T) {
		tmpl := NewTemplate("test")
		_, err := tmpl.SetHTMLTemplate("{{.Invalid")
		if err == nil {
			t.Error("expected error for invalid template syntax")
		}
	})
}

// --- FS / directory template loading tests ---

func TestLoadTemplateFromFS(t *testing.T) {
	t.Run("html file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "welcome.html"), []byte("<h1>Hello {{.Name}}</h1>"), 0644); err != nil {
			t.Fatal(err)
		}

		fsys := os.DirFS(dir)
		tmpl, err := LoadTemplateFromFS(fsys, "welcome", "welcome.html")
		if err != nil {
			t.Fatalf("LoadTemplateFromFS failed: %v", err)
		}

		email, err := tmpl.Render(map[string]any{"Name": "World"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if !strings.Contains(email.HTMLBody, "Hello World") {
			t.Errorf("expected 'Hello World' in HTML body, got %q", email.HTMLBody)
		}
		if email.Body != "" {
			t.Errorf("expected empty text body, got %q", email.Body)
		}
	})

	t.Run("txt file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "welcome.txt"), []byte("Hello {{.Name}}"), 0644); err != nil {
			t.Fatal(err)
		}

		fsys := os.DirFS(dir)
		tmpl, err := LoadTemplateFromFS(fsys, "welcome", "welcome.txt")
		if err != nil {
			t.Fatalf("LoadTemplateFromFS failed: %v", err)
		}

		email, err := tmpl.Render(map[string]any{"Name": "World"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if !strings.Contains(email.Body, "Hello World") {
			t.Errorf("expected 'Hello World' in text body, got %q", email.Body)
		}
		if email.HTMLBody != "" {
			t.Errorf("expected empty HTML body, got %q", email.HTMLBody)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		fsys := os.DirFS(t.TempDir())
		_, err := LoadTemplateFromFS(fsys, "test", "nonexistent.html")
		if err == nil {
			t.Error("expected error for nonexistent file")
		}
	})

	t.Run("invalid template syntax", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "bad.html"), []byte("{{.Invalid"), 0644); err != nil {
			t.Fatal(err)
		}

		fsys := os.DirFS(dir)
		_, err := LoadTemplateFromFS(fsys, "bad", "bad.html")
		if err == nil {
			t.Error("expected error for invalid template syntax")
		}
	})
}

func TestLoadTemplatesFromDir(t *testing.T) {
	t.Run("html and txt pairs", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "welcome.html"), []byte("<h1>Hello {{.Name}}</h1>"), 0644)
		os.WriteFile(filepath.Join(dir, "welcome.txt"), []byte("Hello {{.Name}}"), 0644)
		os.WriteFile(filepath.Join(dir, "reset.html"), []byte("<p>Reset: {{.Link}}</p>"), 0644)

		templates, err := LoadTemplatesFromDir(dir, "*.html", "*.txt")
		if err != nil {
			t.Fatalf("LoadTemplatesFromDir failed: %v", err)
		}

		if len(templates) != 2 {
			t.Fatalf("expected 2 templates, got %d", len(templates))
		}

		// Check welcome template has both HTML and text
		welcomeTmpl := templates["welcome"]
		if welcomeTmpl == nil {
			t.Fatal("expected 'welcome' template")
		}
		email, err := welcomeTmpl.Render(map[string]any{"Name": "Alice"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if !strings.Contains(email.HTMLBody, "Hello Alice") {
			t.Errorf("expected 'Hello Alice' in HTML body, got %q", email.HTMLBody)
		}
		if !strings.Contains(email.Body, "Hello Alice") {
			t.Errorf("expected 'Hello Alice' in text body, got %q", email.Body)
		}

		// Check reset template has only HTML
		resetTmpl := templates["reset"]
		if resetTmpl == nil {
			t.Fatal("expected 'reset' template")
		}
		email, err = resetTmpl.Render(map[string]any{"Link": "https://example.com"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if !strings.Contains(email.HTMLBody, "https://example.com") {
			t.Errorf("expected link in HTML body, got %q", email.HTMLBody)
		}
	})

	t.Run("html only", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "notify.html"), []byte("<p>{{.Msg}}</p>"), 0644)

		templates, err := LoadTemplatesFromDir(dir, "*.html")
		if err != nil {
			t.Fatalf("LoadTemplatesFromDir failed: %v", err)
		}
		if len(templates) != 1 {
			t.Fatalf("expected 1 template, got %d", len(templates))
		}
		if templates["notify"] == nil {
			t.Error("expected 'notify' template")
		}
	})

	t.Run("no matches", func(t *testing.T) {
		dir := t.TempDir()
		_, err := LoadTemplatesFromDir(dir, "*.html")
		if err == nil {
			t.Error("expected error when no templates match")
		}
	})

	t.Run("invalid template in directory", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "bad.html"), []byte("{{.Invalid"), 0644)

		_, err := LoadTemplatesFromDir(dir, "*.html")
		if err == nil {
			t.Error("expected error for invalid template syntax")
		}
	})
}

func TestLoadTemplatesFromFS(t *testing.T) { //nolint:gocyclo
	t.Run("with DirFS", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "order.html"), []byte("<p>Order #{{.ID}}</p>"), 0644)
		os.WriteFile(filepath.Join(dir, "order.txt"), []byte("Order #{{.ID}}"), 0644)

		fsys := os.DirFS(dir)
		templates, err := LoadTemplatesFromFS(fsys, "*.html", "*.txt")
		if err != nil {
			t.Fatalf("LoadTemplatesFromFS failed: %v", err)
		}

		if len(templates) != 1 {
			t.Fatalf("expected 1 template, got %d", len(templates))
		}

		tmpl := templates["order"]
		if tmpl == nil {
			t.Fatal("expected 'order' template")
		}

		email, err := tmpl.Render(map[string]any{"ID": 42})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if !strings.Contains(email.HTMLBody, "Order #42") {
			t.Errorf("expected 'Order #42' in HTML body, got %q", email.HTMLBody)
		}
		if !strings.Contains(email.Body, "Order #42") {
			t.Errorf("expected 'Order #42' in text body, got %q", email.Body)
		}
	})

	t.Run("htm extension", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "legacy.htm"), []byte("<p>{{.Val}}</p>"), 0644)

		fsys := os.DirFS(dir)
		templates, err := LoadTemplatesFromFS(fsys, "*.htm")
		if err != nil {
			t.Fatalf("LoadTemplatesFromFS failed: %v", err)
		}

		tmpl := templates["legacy"]
		if tmpl == nil {
			t.Fatal("expected 'legacy' template")
		}

		email, err := tmpl.Render(map[string]any{"Val": "test"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if !strings.Contains(email.HTMLBody, "test") {
			t.Errorf("expected 'test' in HTML body, got %q", email.HTMLBody)
		}
	})

	t.Run("gohtml extension", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "alert.gohtml"), []byte("<div>{{.Msg}}</div>"), 0644)

		templates, err := LoadTemplatesFromFS(os.DirFS(dir), "*.gohtml")
		if err != nil {
			t.Fatalf("LoadTemplatesFromFS failed: %v", err)
		}

		email, err := templates["alert"].Render(map[string]any{"Msg": "fire"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if !strings.Contains(email.HTMLBody, "fire") {
			t.Errorf("expected 'fire' in HTML body, got %q", email.HTMLBody)
		}
	})

	t.Run("tmpl extension", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "invoice.tmpl"), []byte("<p>#{{.ID}}</p>"), 0644)

		templates, err := LoadTemplatesFromFS(os.DirFS(dir), "*.tmpl")
		if err != nil {
			t.Fatalf("LoadTemplatesFromFS failed: %v", err)
		}

		email, err := templates["invoice"].Render(map[string]any{"ID": 99})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if !strings.Contains(email.HTMLBody, "#99") {
			t.Errorf("expected '#99' in HTML body, got %q", email.HTMLBody)
		}
	})

	t.Run("unsupported extension", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "bad.xyz"), []byte("content"), 0644)

		_, err := LoadTemplatesFromFS(os.DirFS(dir), "*.xyz")
		if err == nil {
			t.Error("expected error for unsupported extension")
		}
		if !strings.Contains(err.Error(), "unsupported template extension") {
			t.Errorf("expected unsupported extension error, got: %v", err)
		}
	})

	t.Run("subject sidecar", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "welcome.html"), []byte("<h1>Hi {{.Name}}</h1>"), 0644)
		os.WriteFile(filepath.Join(dir, "welcome.txt"), []byte("Hi {{.Name}}"), 0644)
		os.WriteFile(filepath.Join(dir, "welcome.subject"), []byte("Welcome, {{.Name}}!\n"), 0644)

		templates, err := LoadTemplatesFromFS(os.DirFS(dir), "*.html", "*.txt", "*.subject")
		if err != nil {
			t.Fatalf("LoadTemplatesFromFS failed: %v", err)
		}
		if len(templates) != 1 {
			t.Fatalf("expected 1 template, got %d", len(templates))
		}

		tmpl := templates["welcome"]
		if tmpl == nil {
			t.Fatal("expected 'welcome' template")
		}

		email, err := tmpl.Render(map[string]any{"Name": "Alice"})
		if err != nil {
			t.Fatalf("Render failed: %v", err)
		}
		if email.Subject != "Welcome, Alice!" {
			t.Errorf("expected subject 'Welcome, Alice!', got %q", email.Subject)
		}
		if !strings.Contains(email.HTMLBody, "Hi Alice") {
			t.Errorf("expected HTML body 'Hi Alice', got %q", email.HTMLBody)
		}
		if !strings.Contains(email.Body, "Hi Alice") {
			t.Errorf("expected text body 'Hi Alice', got %q", email.Body)
		}
	})

	t.Run("subject sidecar with invalid template syntax", func(t *testing.T) {
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "broken.html"), []byte("<p>ok</p>"), 0644)
		os.WriteFile(filepath.Join(dir, "broken.subject"), []byte("Hi {{.Name"), 0644)

		_, err := LoadTemplatesFromFS(os.DirFS(dir), "*.html", "*.subject")
		if err == nil {
			t.Fatal("expected error for invalid subject template syntax")
		}
		if !strings.Contains(err.Error(), "subject") {
			t.Errorf("expected subject parse error, got: %v", err)
		}
	})
}

// --- Mailer tests ---

func TestMailerSendHTML(t *testing.T) {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	ctx := context.Background()
	err := mailer.SendHTML(ctx, []string{"user@example.com"}, "Test", "<h1>Hello</h1>")
	if err != nil {
		t.Fatalf("SendHTML failed: %v", err)
	}

	sent := mock.GetLastEmail()
	if sent.HTMLBody != "<h1>Hello</h1>" {
		t.Errorf("unexpected HTML body: %q", sent.HTMLBody)
	}
}

func TestMailerClose(t *testing.T) {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	err := mailer.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}
}

func TestMailerSendBatchDefaultConcurrency(t *testing.T) {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	emails := []*Email{
		NewEmail().AddTo("user@example.com").SetSubject("Test").SetBody("Body"),
	}

	ctx := context.Background()
	err := mailer.SendBatch(ctx, emails, 0) // 0 should use default
	if err != nil {
		t.Fatalf("batch send failed: %v", err)
	}

	if mock.GetEmailCount() != 1 {
		t.Errorf("expected 1 email, got %d", mock.GetEmailCount())
	}
}

// --- MockSender additional tests ---

func TestMockSenderGetSentEmails(t *testing.T) {
	mock := NewMockSender()
	ctx := context.Background()

	email1 := NewEmail().
		SetFrom("test@example.com").
		AddTo("user1@example.com").
		SetSubject("Test 1").
		SetBody("Body 1")
	if err := mock.Send(ctx, email1); err != nil {
		t.Fatal(err)
	}

	emails := mock.GetSentEmails()
	if len(emails) != 1 {
		t.Errorf("expected 1 email, got %d", len(emails))
	}
}

func TestMockSenderGetEmailsBySubject(t *testing.T) {
	mock := NewMockSender()
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		subject := "Test"
		if i == 1 {
			subject = "Other"
		}
		email := NewEmail().
			SetFrom("test@example.com").
			AddTo("user@example.com").
			SetSubject(subject).
			SetBody("Body")
		if err := mock.Send(ctx, email); err != nil {
			t.Fatal(err)
		}
	}

	emails := mock.GetEmailsBySubject("Test")
	if len(emails) != 2 {
		t.Errorf("expected 2 emails with subject 'Test', got %d", len(emails))
	}
}

func TestMockSenderGetLastEmailEmpty(t *testing.T) {
	mock := NewMockSender()
	if mock.GetLastEmail() != nil {
		t.Error("GetLastEmail should return nil when no emails sent")
	}
}

// --- isRetryableError tests ---

func TestIsRetryableError_PermanentSMTP5xx(t *testing.T) {
	// 5xx replies per RFC 5321 are permanent failures and should not be retried.
	permanent := []int{500, 501, 550, 553, 571}
	for _, code := range permanent {
		tpErr := &textproto.Error{Code: code, Msg: "permanent"}
		if isRetryableError(tpErr) {
			t.Errorf("code %d should not be retryable", code)
		}
		// Also confirm it's detected when wrapped.
		wrapped := fmt.Errorf("add recipient x: %w", tpErr)
		if isRetryableError(wrapped) {
			t.Errorf("wrapped code %d should not be retryable", code)
		}
	}
}

func TestIsRetryableError_Transient4xx(t *testing.T) {
	// 4xx replies are transient and should be retried.
	transient := []int{421, 450, 451, 452}
	for _, code := range transient {
		tpErr := &textproto.Error{Code: code, Msg: "transient"}
		if !isRetryableError(tpErr) {
			t.Errorf("code %d should be retryable", code)
		}
	}
}

// --- SMTPSender.Close test ---

func TestSMTPSenderClose(t *testing.T) {
	sender, err := NewSMTPSender(SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
	})
	if err != nil {
		t.Fatalf("NewSMTPSender failed: %v", err)
	}

	err = sender.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}
}

// --- Validation edge cases ---

func TestValidateReplyTo(t *testing.T) {
	email := NewEmail().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetReplyTo("invalid-reply-to").
		SetSubject("Test").
		SetBody("Body")

	_, err := email.Build()
	if err == nil {
		t.Error("expected error for invalid reply-to address")
	}
}

func TestValidateValidReplyTo(t *testing.T) {
	email := NewEmail().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetReplyTo("reply@example.com").
		SetSubject("Test").
		SetBody("Body")

	_, err := email.Build()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateInvalidCcAddress(t *testing.T) {
	email := NewEmail().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		AddCc("invalid-cc").
		SetSubject("Test").
		SetBody("Body")

	_, err := email.Build()
	if err == nil {
		t.Error("expected error for invalid cc address")
	}
}

func TestValidateInvalidBccAddress(t *testing.T) {
	email := NewEmail().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		AddBcc("invalid-bcc").
		SetSubject("Test").
		SetBody("Body")

	_, err := email.Build()
	if err == nil {
		t.Error("expected error for invalid bcc address")
	}
}

func TestValidateCcOnlyRecipient(t *testing.T) {
	email := NewEmail().
		SetFrom("sender@example.com").
		AddCc("cc@example.com").
		SetSubject("Test").
		SetBody("Body")

	_, err := email.Build()
	if err != nil {
		t.Errorf("CC-only recipient should be valid, got: %v", err)
	}
}

func TestValidateBccOnlyRecipient(t *testing.T) {
	email := NewEmail().
		SetFrom("sender@example.com").
		AddBcc("bcc@example.com").
		SetSubject("Test").
		SetBody("Body")

	_, err := email.Build()
	if err != nil {
		t.Errorf("BCC-only recipient should be valid, got: %v", err)
	}
}

// --- Example functions for godoc ---

func ExampleNewEmail() {
	email, err := NewEmail().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetSubject("Hello").
		SetBody("World").
		Build()
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(email.Subject)
	// Output: Hello
}

func ExampleNewMockSender() {
	mock := NewMockSender()
	mailer := NewMailer(mock, "test@example.com")

	ctx := context.Background()
	if err := mailer.Send(ctx, []string{"user@example.com"}, "Test", "Hello"); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(mock.GetEmailCount())
	fmt.Println(mock.GetLastEmail().Subject)
	// Output:
	// 1
	// Test
}

func ExampleNewTemplate() {
	tmpl := NewTemplate("welcome")
	tmpl.SetSubject("Welcome {{.Name}}")
	if _, err := tmpl.SetTextTemplate("Hello {{.Name}}, welcome!"); err != nil {
		fmt.Println("error:", err)
		return
	}

	email, err := tmpl.Render(map[string]any{"Name": "Alice"})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(email.Subject)
	fmt.Println(email.Body)
	// Output:
	// Welcome Alice
	// Hello Alice, welcome!
}

// --- Benchmarks ---

func BenchmarkBuildMessage(b *testing.B) {
	sender, err := NewSMTPSender(SMTPConfig{
		Host: "smtp.example.com",
		Port: 587,
	})
	if err != nil {
		b.Fatal(err)
	}

	email := &Email{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com"},
		Subject: "Benchmark Test",
		Body:    "This is a benchmark test body.",
		Headers: make(map[string]string),
	}

	b.ResetTimer()
	for range b.N {
		_, _ = sender.buildMessage(email)
	}
}

func BenchmarkWrapText(b *testing.B) {
	text := strings.Repeat("abcdefghijklmnop", 100)
	b.ResetTimer()
	for range b.N {
		wrapText(text, 76)
	}
}

func BenchmarkGenerateUniqueID(b *testing.B) {
	for range b.N {
		generateUniqueID()
	}
}

func BenchmarkValidateEmail(b *testing.B) {
	email := NewEmail().
		SetFrom("sender@example.com").
		AddTo("recipient@example.com").
		SetSubject("Test").
		SetBody("Body")

	b.ResetTimer()
	for range b.N {
		_ = email.Validate()
	}
}
