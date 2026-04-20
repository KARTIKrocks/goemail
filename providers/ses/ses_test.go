package ses

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"

	email "github.com/KARTIKrocks/goemail"
)

// mockSES implements sesAPI for testing.
type mockSES struct {
	input *sesv2.SendEmailInput
	err   error
}

func (m *mockSES) SendEmail(_ context.Context, params *sesv2.SendEmailInput, _ ...func(*sesv2.Options)) (*sesv2.SendEmailOutput, error) {
	m.input = params
	if m.err != nil {
		return nil, m.err
	}
	return &sesv2.SendEmailOutput{MessageId: aws.String("test-message-id")}, nil
}

func testEmail() *email.Email {
	e, _ := email.NewEmail().
		SetFrom("from@example.com").
		AddTo("to@example.com").
		SetSubject("Test").
		SetBody("Hello").
		Build()
	return e
}

func TestSend_Success(t *testing.T) {
	mock := &mockSES{}
	s := newSenderWithClient(mock, "")

	e := testEmail()
	if err := s.Send(context.Background(), e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.input == nil {
		t.Fatal("expected SendEmail to be called")
	}
	if got := *mock.input.FromEmailAddress; got != "from@example.com" {
		t.Fatalf("expected from 'from@example.com', got %q", got)
	}
	if len(mock.input.Destination.ToAddresses) != 1 || mock.input.Destination.ToAddresses[0] != "to@example.com" {
		t.Fatalf("unexpected to addresses: %v", mock.input.Destination.ToAddresses)
	}
	if mock.input.Content.Raw == nil {
		t.Fatal("expected raw message content")
	}
}

func TestSend_APIError(t *testing.T) {
	want := errors.New("SES API error")
	mock := &mockSES{err: want}
	s := newSenderWithClient(mock, "")

	err := s.Send(context.Background(), testEmail())
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, want) {
		t.Fatalf("expected %v in error chain, got %v", want, err)
	}
}

func TestSend_ConfigurationSet(t *testing.T) {
	mock := &mockSES{}
	s := newSenderWithClient(mock, "my-config-set")

	if err := s.Send(context.Background(), testEmail()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.input.ConfigurationSetName == nil || *mock.input.ConfigurationSetName != "my-config-set" {
		t.Fatalf("expected config set 'my-config-set', got %v", mock.input.ConfigurationSetName)
	}
}

func TestSend_NoConfigurationSet(t *testing.T) {
	mock := &mockSES{}
	s := newSenderWithClient(mock, "")

	if err := s.Send(context.Background(), testEmail()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.input.ConfigurationSetName != nil {
		t.Fatalf("expected nil config set, got %q", *mock.input.ConfigurationSetName)
	}
}

func TestSend_RawMessageContent(t *testing.T) {
	mock := &mockSES{}
	s := newSenderWithClient(mock, "")

	e, _ := email.NewEmail().
		SetFrom("from@example.com").
		AddTo("to@example.com").
		SetSubject("Raw Test").
		SetBody("Plain text body").
		SetHTMLBody("<h1>HTML body</h1>").
		Build()

	if err := s.Send(context.Background(), e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	raw := string(mock.input.Content.Raw.Data)
	if !strings.Contains(raw, "From: from@example.com") {
		t.Error("raw message missing From header")
	}
	if !strings.Contains(raw, "To: to@example.com") {
		t.Error("raw message missing To header")
	}
	if !strings.Contains(raw, "Subject: Raw Test") {
		t.Error("raw message missing Subject header")
	}
	if !strings.Contains(raw, "MIME-Version: 1.0") {
		t.Error("raw message missing MIME-Version header")
	}
}

func TestSend_AllFields(t *testing.T) {
	mock := &mockSES{}
	s := newSenderWithClient(mock, "")

	e, _ := email.NewEmail().
		SetFrom("from@example.com").
		AddTo("to@example.com").
		AddCc("cc@example.com").
		AddBcc("bcc@example.com").
		SetSubject("Full Test").
		SetBody("text").
		Build()

	if err := s.Send(context.Background(), e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dest := mock.input.Destination
	if len(dest.CcAddresses) != 1 || dest.CcAddresses[0] != "cc@example.com" {
		t.Fatalf("unexpected cc: %v", dest.CcAddresses)
	}
	if len(dest.BccAddresses) != 1 || dest.BccAddresses[0] != "bcc@example.com" {
		t.Fatalf("unexpected bcc: %v", dest.BccAddresses)
	}
}

func TestSender_ImplementsSender(t *testing.T) {
	var _ email.Sender = (*Sender)(nil)
}
