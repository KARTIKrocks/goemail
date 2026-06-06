package sendgrid

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	email "github.com/KARTIKrocks/goemail"
)

func testEmail() *email.Email {
	e, _ := email.NewEmail().
		SetFrom("from@example.com").
		AddTo("to@example.com").
		SetSubject("Test").
		SetBody("Hello").
		Build()
	return e
}

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestNew_ValidConfig(t *testing.T) {
	s, err := New(Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil sender")
	}
}

func TestSend_Success(t *testing.T) {
	var gotBody []byte
	var gotAuth string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	s, _ := New(Config{APIKey: "test-key", BaseURL: srv.URL})
	e := testEmail()

	if err := s.Send(context.Background(), e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify auth header
	if gotAuth != "Bearer test-key" {
		t.Fatalf("expected 'Bearer test-key', got %q", gotAuth)
	}

	// Verify payload structure
	var payload sgPayload
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.From.Email != "from@example.com" {
		t.Fatalf("expected from 'from@example.com', got %q", payload.From.Email)
	}
	if payload.Subject != "Test" {
		t.Fatalf("expected subject 'Test', got %q", payload.Subject)
	}
	if len(payload.Personalizations) != 1 {
		t.Fatalf("expected 1 personalization, got %d", len(payload.Personalizations))
	}
	if len(payload.Personalizations[0].To) != 1 || payload.Personalizations[0].To[0].Email != "to@example.com" {
		t.Fatalf("unexpected to: %+v", payload.Personalizations[0].To)
	}
	if len(payload.Content) != 1 || payload.Content[0].Value != "Hello" {
		t.Fatalf("unexpected content: %+v", payload.Content)
	}
}

func TestSend_AllFields(t *testing.T) {
	var gotBody []byte

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	s, _ := New(Config{APIKey: "key", BaseURL: srv.URL})

	e, _ := email.NewEmail().
		SetFrom("from@example.com").
		AddTo("to1@example.com").
		AddCc("cc@example.com").
		AddBcc("bcc@example.com").
		SetReplyTo("reply@example.com").
		SetSubject("Full Test").
		SetBody("Plain text").
		SetHTMLBody("<h1>HTML</h1>").
		AddAttachment("file.txt", "text/plain", []byte("content")).
		AddHeader("X-Custom", "value").
		Build()

	if err := s.Send(context.Background(), e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload sgPayload
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	pers := payload.Personalizations[0]
	if len(pers.Cc) != 1 || pers.Cc[0].Email != "cc@example.com" {
		t.Fatalf("unexpected cc: %+v", pers.Cc)
	}
	if len(pers.Bcc) != 1 || pers.Bcc[0].Email != "bcc@example.com" {
		t.Fatalf("unexpected bcc: %+v", pers.Bcc)
	}
	if payload.ReplyTo == nil || payload.ReplyTo.Email != "reply@example.com" {
		t.Fatalf("unexpected reply-to: %+v", payload.ReplyTo)
	}
	if len(payload.Content) != 2 {
		t.Fatalf("expected 2 content parts, got %d", len(payload.Content))
	}
	if len(payload.Attachments) != 1 || payload.Attachments[0].Filename != "file.txt" {
		t.Fatalf("unexpected attachments: %+v", payload.Attachments)
	}
	if payload.Headers["X-Custom"] != "value" {
		t.Fatalf("expected custom header, got %+v", payload.Headers)
	}
}

func TestSend_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"errors":[{"message":"bad request"}]}`))
	}))
	defer srv.Close()

	s, _ := New(Config{APIKey: "key", BaseURL: srv.URL})
	err := s.Send(context.Background(), testEmail())
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestSend_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	s, _ := New(Config{APIKey: "key", BaseURL: srv.URL})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.Send(ctx, testEmail())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestSender_ImplementsSender(t *testing.T) {
	var _ email.Sender = (*Sender)(nil)
}
