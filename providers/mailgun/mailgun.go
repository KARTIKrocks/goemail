package mailgun

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"

	email "github.com/KARTIKrocks/goemail"
)

// Config holds the Mailgun provider configuration.
type Config struct {
	// Domain is the Mailgun sending domain (required, e.g., "mg.example.com").
	Domain string

	// APIKey is the Mailgun API key (required).
	APIKey string

	// BaseURL is the Mailgun API base URL.
	// Default: "https://api.mailgun.net".
	// For EU accounts use "https://api.eu.mailgun.net".
	BaseURL string

	// HTTPClient is the HTTP client used for API calls.
	// Default: http.DefaultClient.
	HTTPClient *http.Client
}

// Sender sends emails through the Mailgun v3 Messages API.
type Sender struct {
	domain string
	apiKey string
	base   string
	client *http.Client
}

// New creates a new Mailgun Sender.
func New(cfg Config) (*Sender, error) {
	if cfg.Domain == "" {
		return nil, errors.New("mailgun: domain is required")
	}
	if cfg.APIKey == "" {
		return nil, errors.New("mailgun: API key is required")
	}
	base := cfg.BaseURL
	if base == "" {
		base = "https://api.mailgun.net"
	}
	c := cfg.HTTPClient
	if c == nil {
		c = http.DefaultClient
	}
	return &Sender{domain: cfg.Domain, apiKey: cfg.APIKey, base: base, client: c}, nil
}

// Send sends an email through Mailgun. It implements email.Sender.
func (s *Sender) Send(ctx context.Context, e *email.Email) error {
	if err := e.Validate(); err != nil {
		return fmt.Errorf("mailgun: %w", err)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	// Required fields
	writeField(w, "from", e.From)
	for _, addr := range e.To {
		writeField(w, "to", addr)
	}
	writeField(w, "subject", e.Subject)

	// Optional text/html
	if e.Body != "" {
		writeField(w, "text", e.Body)
	}
	if e.HTMLBody != "" {
		writeField(w, "html", e.HTMLBody)
	}

	// Cc / Bcc
	for _, addr := range e.Cc {
		writeField(w, "cc", addr)
	}
	for _, addr := range e.Bcc {
		writeField(w, "bcc", addr)
	}

	// Reply-To and custom headers
	if e.ReplyTo != "" {
		writeField(w, "h:Reply-To", e.ReplyTo)
	}
	for key, value := range e.Headers {
		writeField(w, "h:"+key, value)
	}

	// Attachments. CreateFormFile would hard-code Content-Type to
	// application/octet-stream, which Mailgun then forwards verbatim and
	// clobbers the caller's real MIME type (e.g., application/pdf). Build
	// the part manually so att.ContentType is preserved.
	for _, att := range e.Attachments {
		ct := att.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="attachment"; filename="%s"`, mgEscapeQuotes(att.Filename)))
		h.Set("Content-Type", ct)
		part, err := w.CreatePart(h)
		if err != nil {
			return fmt.Errorf("mailgun: create attachment part: %w", err)
		}
		if _, err = part.Write(att.Data); err != nil {
			return fmt.Errorf("mailgun: write attachment data: %w", err)
		}
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("mailgun: close multipart writer: %w", err)
	}

	url := fmt.Sprintf("%s/v3/%s/messages", s.base, s.domain)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &buf)
	if err != nil {
		return fmt.Errorf("mailgun: create request: %w", err)
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.SetBasicAuth("api", s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("mailgun: send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("mailgun: API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// Close is a no-op for Mailgun (HTTP is stateless).
func (s *Sender) Close() error { return nil }

func writeField(w *multipart.Writer, key, value string) {
	_ = w.WriteField(key, value)
}

var mgQuoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func mgEscapeQuotes(s string) string {
	return mgQuoteEscaper.Replace(s)
}
