package sendgrid

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/mail"

	email "github.com/KARTIKrocks/goemail"
)

// Config holds the SendGrid provider configuration.
type Config struct {
	// APIKey is the SendGrid API key (required).
	APIKey string

	// BaseURL is the SendGrid API base URL.
	// Default: "https://api.sendgrid.com".
	BaseURL string

	// HTTPClient is the HTTP client used for API calls.
	// Default: http.DefaultClient.
	HTTPClient *http.Client
}

// Sender sends emails through the SendGrid v3 API.
type Sender struct {
	apiKey string
	base   string
	client *http.Client
}

// New creates a new SendGrid Sender.
func New(cfg Config) (*Sender, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("sendgrid: API key is required")
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.sendgrid.com"
	}
	c := cfg.HTTPClient
	if c == nil {
		c = http.DefaultClient
	}
	return &Sender{apiKey: cfg.APIKey, base: base, client: c}, nil
}

// Send sends an email through SendGrid. It implements email.Sender.
func (s *Sender) Send(ctx context.Context, e *email.Email) error {
	if err := e.Validate(); err != nil {
		return fmt.Errorf("sendgrid: %w", err)
	}

	payload := s.buildPayload(e)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("sendgrid: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.base+"/v3/mail/send", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sendgrid: create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("sendgrid: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("sendgrid: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Close is a no-op for SendGrid (HTTP is stateless).
func (s *Sender) Close() error { return nil }

// --- payload types ---

type sgPayload struct {
	Personalizations []sgPersonalization `json:"personalizations"`
	From             sgAddress           `json:"from"`
	Subject          string              `json:"subject"`
	Content          []sgContent         `json:"content"`
	Attachments      []sgAttachment      `json:"attachments,omitempty"`
	ReplyTo          *sgAddress          `json:"reply_to,omitempty"`
	Headers          map[string]string   `json:"headers,omitempty"`
}

type sgPersonalization struct {
	To  []sgAddress `json:"to"`
	Cc  []sgAddress `json:"cc,omitempty"`
	Bcc []sgAddress `json:"bcc,omitempty"`
}

type sgAddress struct {
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
}

// parseSGAddress parses an RFC 5322 address string into a sgAddress,
// separating the display name from the bare email. Falls back to using
// the raw string as the email if parsing fails.
func parseSGAddress(addr string) sgAddress {
	parsed, err := mail.ParseAddress(addr)
	if err != nil {
		return sgAddress{Email: addr}
	}
	return sgAddress{Email: parsed.Address, Name: parsed.Name}
}

type sgContent struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type sgAttachment struct {
	Content  string `json:"content"`
	Filename string `json:"filename"`
	Type     string `json:"type,omitempty"`
}

func (s *Sender) buildPayload(e *email.Email) sgPayload {
	p := sgPayload{
		From:    parseSGAddress(e.From),
		Subject: e.Subject,
	}

	// Personalizations
	pers := sgPersonalization{}
	for _, addr := range e.To {
		pers.To = append(pers.To, parseSGAddress(addr))
	}
	for _, addr := range e.Cc {
		pers.Cc = append(pers.Cc, parseSGAddress(addr))
	}
	for _, addr := range e.Bcc {
		pers.Bcc = append(pers.Bcc, parseSGAddress(addr))
	}
	p.Personalizations = []sgPersonalization{pers}

	// Content
	if e.Body != "" {
		p.Content = append(p.Content, sgContent{Type: "text/plain", Value: e.Body})
	}
	if e.HTMLBody != "" {
		p.Content = append(p.Content, sgContent{Type: "text/html", Value: e.HTMLBody})
	}

	// Attachments
	for _, att := range e.Attachments {
		p.Attachments = append(p.Attachments, sgAttachment{
			Content:  base64.StdEncoding.EncodeToString(att.Data),
			Filename: att.Filename,
			Type:     att.ContentType,
		})
	}

	// Reply-To
	if e.ReplyTo != "" {
		rt := parseSGAddress(e.ReplyTo)
		p.ReplyTo = &rt
	}

	// Custom headers
	if len(e.Headers) > 0 {
		p.Headers = e.Headers
	}

	return p
}
