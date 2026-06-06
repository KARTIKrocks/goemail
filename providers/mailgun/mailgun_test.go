package mailgun

import (
	"context"
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

func TestNew_MissingDomain(t *testing.T) {
	_, err := New(Config{APIKey: "key"})
	if err == nil {
		t.Fatal("expected error for missing domain")
	}
}

func TestNew_MissingAPIKey(t *testing.T) {
	_, err := New(Config{Domain: "mg.example.com"})
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestNew_ValidConfig(t *testing.T) {
	s, err := New(Config{Domain: "mg.example.com", APIKey: "key"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s == nil {
		t.Fatal("expected non-nil sender")
	}
}

func TestSend_Success(t *testing.T) {
	var gotAuth string
	var gotContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")

		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("parse form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if got := r.FormValue("from"); got != "from@example.com" {
			t.Errorf("expected from 'from@example.com', got %q", got)
		}
		if got := r.FormValue("to"); got != "to@example.com" {
			t.Errorf("expected to 'to@example.com', got %q", got)
		}
		if got := r.FormValue("subject"); got != "Test" {
			t.Errorf("expected subject 'Test', got %q", got)
		}
		if got := r.FormValue("text"); got != "Hello" {
			t.Errorf("expected text 'Hello', got %q", got)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"<test@mg.example.com>","message":"Queued."}`))
	}))
	defer srv.Close()

	s, _ := New(Config{Domain: "mg.example.com", APIKey: "test-key", BaseURL: srv.URL})

	if err := s.Send(context.Background(), testEmail()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify basic auth
	if gotAuth == "" {
		t.Fatal("expected Authorization header")
	}

	// Verify multipart content type
	if gotContentType == "" {
		t.Fatal("expected Content-Type header")
	}
}

func TestSend_AllFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Errorf("parse form: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if got := r.FormValue("cc"); got != "cc@example.com" {
			t.Errorf("expected cc 'cc@example.com', got %q", got)
		}
		if got := r.FormValue("bcc"); got != "bcc@example.com" {
			t.Errorf("expected bcc 'bcc@example.com', got %q", got)
		}
		if got := r.FormValue("h:Reply-To"); got != "reply@example.com" {
			t.Errorf("expected reply-to 'reply@example.com', got %q", got)
		}
		if got := r.FormValue("html"); got != "<h1>HTML</h1>" {
			t.Errorf("expected html '<h1>HTML</h1>', got %q", got)
		}
		if got := r.FormValue("h:X-Custom"); got != "value" {
			t.Errorf("expected custom header 'value', got %q", got)
		}

		// Check attachment
		_, hdr, err := r.FormFile("attachment")
		if err != nil {
			t.Errorf("expected attachment: %v", err)
		} else {
			if hdr.Filename != "file.txt" {
				t.Errorf("expected attachment filename 'file.txt', got %q", hdr.Filename)
			}
			if got := hdr.Header.Get("Content-Type"); got != "text/plain" {
				t.Errorf("expected attachment Content-Type 'text/plain', got %q", got)
			}
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"<test@mg.example.com>","message":"Queued."}`))
	}))
	defer srv.Close()

	s, _ := New(Config{Domain: "mg.example.com", APIKey: "key", BaseURL: srv.URL})

	e, _ := email.NewEmail().
		SetFrom("from@example.com").
		AddTo("to@example.com").
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
}

func TestSend_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"message":"bad request"}`))
	}))
	defer srv.Close()

	s, _ := New(Config{Domain: "mg.example.com", APIKey: "key", BaseURL: srv.URL})
	err := s.Send(context.Background(), testEmail())
	if err == nil {
		t.Fatal("expected error for 400 response")
	}
}

func TestSend_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s, _ := New(Config{Domain: "mg.example.com", APIKey: "key", BaseURL: srv.URL})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := s.Send(ctx, testEmail())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestSend_BasicAuthHeader(t *testing.T) {
	var gotUser, gotPass string
	var gotOK bool

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUser, gotPass, gotOK = r.BasicAuth()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message":"Queued."}`))
	}))
	defer srv.Close()

	s, _ := New(Config{Domain: "mg.example.com", APIKey: "secret-key", BaseURL: srv.URL})
	s.Send(context.Background(), testEmail())

	if !gotOK {
		t.Fatal("expected basic auth")
	}
	if gotUser != "api" {
		t.Fatalf("expected user 'api', got %q", gotUser)
	}
	if gotPass != "secret-key" {
		t.Fatalf("expected password 'secret-key', got %q", gotPass)
	}
}

func TestSend_EURegionBaseURL(t *testing.T) {
	s, err := New(Config{
		Domain:  "mg.example.com",
		APIKey:  "key",
		BaseURL: "https://api.eu.mailgun.net",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.base != "https://api.eu.mailgun.net" {
		t.Fatalf("expected EU base URL, got %q", s.base)
	}
}

func TestSender_ImplementsSender(t *testing.T) {
	var _ email.Sender = (*Sender)(nil)
}
