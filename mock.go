package email

import (
	"context"
	"sync"
)

// MockSender is a mock email sender for testing.
// It records all sent emails in memory and provides methods
// to inspect them for test assertions.
type MockSender struct {
	mu     sync.Mutex
	emails []*Email
	sendFn func(ctx context.Context, email *Email) error // optional custom send function
}

// NewMockSender creates a new mock sender
func NewMockSender() *MockSender {
	return &MockSender{
		emails: []*Email{},
	}
}

// SetSendFunc sets a custom send function for testing error scenarios.
// If nil, Send will succeed by default.
//
// Example:
//
//	mock := email.NewMockSender()
//	mock.SetSendFunc(func(ctx context.Context, email *Email) error {
//	    return errors.New("smtp connection failed")
//	})
func (m *MockSender) SetSendFunc(fn func(ctx context.Context, email *Email) error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.sendFn = fn
}

// Send records the email
func (m *MockSender) Send(ctx context.Context, email *Email) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate email
	if err := email.Validate(); err != nil {
		return err
	}

	// Call custom send function if set
	if m.sendFn != nil {
		if err := m.sendFn(ctx, email); err != nil {
			return err
		}
	}

	// Record email
	m.emails = append(m.emails, email)
	return nil
}

// GetSentEmails returns all sent emails
func (m *MockSender) GetSentEmails() []*Email {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Return a copy to prevent race conditions
	emails := make([]*Email, len(m.emails))
	copy(emails, m.emails)
	return emails
}

// GetLastEmail returns the last sent email, or nil if no emails have been sent
func (m *MockSender) GetLastEmail() *Email {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.emails) == 0 {
		return nil
	}

	return m.emails[len(m.emails)-1]
}

// GetEmailCount returns the number of sent emails
func (m *MockSender) GetEmailCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.emails)
}

// GetEmailsTo returns all emails sent to a specific recipient
func (m *MockSender) GetEmailsTo(recipient string) []*Email {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*Email
	for _, email := range m.emails {
		for _, to := range email.To {
			if to == recipient {
				result = append(result, email)
				break
			}
		}
	}
	return result
}

// GetEmailsBySubject returns all emails with a specific subject
func (m *MockSender) GetEmailsBySubject(subject string) []*Email {
	m.mu.Lock()
	defer m.mu.Unlock()

	var result []*Email
	for _, email := range m.emails {
		if email.Subject == subject {
			result = append(result, email)
		}
	}
	return result
}

// Reset clears all sent emails
func (m *MockSender) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.emails = []*Email{}
	m.sendFn = nil
}

// Close closes the mock sender
func (m *MockSender) Close() error {
	return nil
}
