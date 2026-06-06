package email

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/sync/errgroup"
)

// Mailer provides a high-level email sending interface.
// It is safe for concurrent use.
type Mailer struct {
	sender    Sender
	from      string
	mu        sync.RWMutex
	templates map[string]*Template
}

// cloneEmail creates a deep copy of an Email to avoid shared mutable state.
func cloneEmail(src *Email) *Email {
	if src == nil {
		return nil
	}
	dst := &Email{
		From:     src.From,
		ReplyTo:  src.ReplyTo,
		Subject:  src.Subject,
		Body:     src.Body,
		HTMLBody: src.HTMLBody,
		err:      src.err,
	}
	// Deep copy slices
	if len(src.To) > 0 {
		dst.To = make([]string, len(src.To))
		copy(dst.To, src.To)
	}
	if len(src.Cc) > 0 {
		dst.Cc = make([]string, len(src.Cc))
		copy(dst.Cc, src.Cc)
	}
	if len(src.Bcc) > 0 {
		dst.Bcc = make([]string, len(src.Bcc))
		copy(dst.Bcc, src.Bcc)
	}
	if len(src.Attachments) > 0 {
		dst.Attachments = make([]Attachment, len(src.Attachments))
		for i, att := range src.Attachments {
			dst.Attachments[i] = Attachment{
				Filename:    att.Filename,
				ContentType: att.ContentType,
				Data:        append([]byte(nil), att.Data...),
			}
		}
	}
	if len(src.Headers) > 0 {
		dst.Headers = make(map[string]string, len(src.Headers))
		for k, v := range src.Headers {
			dst.Headers[k] = v
		}
	}
	return dst
}

// NewMailer creates a new mailer
func NewMailer(sender Sender, from string) *Mailer {
	return &Mailer{
		sender:    sender,
		from:      from,
		templates: make(map[string]*Template),
	}
}

// MailerOption configures a Mailer.
type MailerOption func(*Mailer)

// WithMiddleware returns a MailerOption that wraps the Mailer's Sender
// with the given middlewares using Chain.
func WithMiddleware(middlewares ...Middleware) MailerOption {
	return func(m *Mailer) {
		m.sender = Chain(m.sender, middlewares...)
	}
}

// NewMailerWithOptions creates a new Mailer with the given options applied.
//
//	mailer := email.NewMailerWithOptions(sender, "from@example.com",
//	    email.WithMiddleware(
//	        email.WithLogging(logger),
//	        email.WithRecovery(),
//	    ),
//	)
func NewMailerWithOptions(sender Sender, from string, opts ...MailerOption) *Mailer {
	m := &Mailer{
		sender:    sender,
		from:      from,
		templates: make(map[string]*Template),
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// RegisterTemplate registers an email template.
// It is safe for concurrent use.
func (m *Mailer) RegisterTemplate(name string, tmpl *Template) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.templates[name] = tmpl
}

// Send sends a simple email
func (m *Mailer) Send(ctx context.Context, to []string, subject, body string) error {
	email := NewEmail().
		SetFrom(m.from).
		AddTo(to...).
		SetSubject(subject).
		SetBody(body)

	builtEmail, err := email.Build()
	if err != nil {
		return err
	}

	return m.sender.Send(ctx, builtEmail)
}

// SendHTML sends an HTML email
func (m *Mailer) SendHTML(ctx context.Context, to []string, subject, html string) error {
	email := NewEmail().
		SetFrom(m.from).
		AddTo(to...).
		SetSubject(subject).
		SetHTMLBody(html)

	builtEmail, err := email.Build()
	if err != nil {
		return err
	}

	return m.sender.Send(ctx, builtEmail)
}

// SendTemplate sends an email using a registered template
func (m *Mailer) SendTemplate(ctx context.Context, to []string, templateName string, data any) error {
	m.mu.RLock()
	tmpl, exists := m.templates[templateName]
	m.mu.RUnlock()
	if !exists {
		return fmt.Errorf("template not found: %s", templateName)
	}

	email, err := tmpl.Render(data)
	if err != nil {
		return fmt.Errorf("failed to render template: %w", err)
	}

	email.SetFrom(m.from).AddTo(to...)

	builtEmail, err := email.Build()
	if err != nil {
		return err
	}

	return m.sender.Send(ctx, builtEmail)
}

// SendEmail sends a custom email
func (m *Mailer) SendEmail(ctx context.Context, email *Email) error {
	if email == nil {
		return fmt.Errorf("nil email")
	}
	e := cloneEmail(email) // deep copy to avoid mutating the caller's email
	if e.From == "" {
		e.From = m.from
	}

	builtEmail, err := e.Build()
	if err != nil {
		return err
	}

	return m.sender.Send(ctx, builtEmail)
}

// SendBatch sends multiple emails concurrently with a concurrency limit.
// The concurrencyLimit parameter controls how many emails are sent simultaneously.
// If concurrencyLimit is <= 0, a default of 10 is used.
//
// All emails are validated before sending begins. If any email fails validation,
// the entire batch fails without sending any emails.
//
// If any email fails to send, the first error is returned. All emails are
// attempted regardless of individual failures.
func (m *Mailer) SendBatch(ctx context.Context, emails []*Email, concurrencyLimit int) error {
	if concurrencyLimit <= 0 {
		concurrencyLimit = 10
	}

	// Build copies with From filled in and validate each.
	prepared := make([]*Email, len(emails))
	for i, email := range emails {
		if email == nil {
			return fmt.Errorf("email %d is nil", i)
		}
		e := cloneEmail(email) // deep copy to avoid mutating the caller's email
		if e.From == "" {
			e.From = m.from
		}
		if _, err := e.Build(); err != nil {
			return fmt.Errorf("email %d validation failed: %w", i, err)
		}
		prepared[i] = e
	}

	// Send emails concurrently. Use a plain errgroup (no WithContext) so that
	// a single failure does not cancel other in-flight sends.
	g := &errgroup.Group{}
	g.SetLimit(concurrencyLimit)

	for _, email := range prepared {
		g.Go(func() error {
			if err := ctx.Err(); err != nil {
				return err
			}
			return m.sender.Send(ctx, email)
		})
	}

	return g.Wait()
}

// Close closes the mailer
func (m *Mailer) Close() error {
	return m.sender.Close()
}
