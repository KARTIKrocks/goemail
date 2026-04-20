package email

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- Test helpers ---

type recordingWebhookHandler struct {
	mu     sync.Mutex
	events []WebhookEvent
	err    error // error to return from HandleEvent
}

func (h *recordingWebhookHandler) HandleEvent(_ context.Context, event WebhookEvent) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.events = append(h.events, event)
	return h.err
}

func (h *recordingWebhookHandler) getEvents() []WebhookEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	events := make([]WebhookEvent, len(h.events))
	copy(events, h.events)
	return events
}

type staticParser struct {
	events []WebhookEvent
	err    error
}

func (p *staticParser) Parse(_ *http.Request) ([]WebhookEvent, error) {
	return p.events, p.err
}

func sampleEvents() []WebhookEvent {
	return []WebhookEvent{
		{
			Type:      EventDelivered,
			MessageID: "msg-001",
			Recipient: "user@example.com",
			Timestamp: time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC),
			Provider:  "test",
		},
		{
			Type:      EventBounced,
			MessageID: "msg-002",
			Recipient: "bounce@example.com",
			Timestamp: time.Date(2026, 3, 23, 12, 1, 0, 0, time.UTC),
			Provider:  "test",
			Reason:    "mailbox full",
		},
	}
}

// --- WebhookReceiver tests ---

func TestWebhookReceiver_DispatchesEvents(t *testing.T) {
	events := sampleEvents()
	parser := &staticParser{events: events}
	handler := &recordingWebhookHandler{}

	receiver := NewWebhookReceiver(parser, handler)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
	w := httptest.NewRecorder()

	receiver.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	got := handler.getEvents()
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	if got[0].Type != EventDelivered {
		t.Errorf("expected EventDelivered, got %s", got[0].Type)
	}
	if got[0].MessageID != "msg-001" {
		t.Errorf("expected msg-001, got %s", got[0].MessageID)
	}
	if got[1].Type != EventBounced {
		t.Errorf("expected EventBounced, got %s", got[1].Type)
	}
	if got[1].Reason != "mailbox full" {
		t.Errorf("expected 'mailbox full', got %s", got[1].Reason)
	}
}

func TestWebhookReceiver_RejectsNonPOST(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPut, http.MethodDelete, http.MethodPatch}

	parser := &staticParser{events: sampleEvents()}
	handler := &recordingWebhookHandler{}
	receiver := NewWebhookReceiver(parser, handler)

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := httptest.NewRequest(method, "/webhooks/email", nil)
			w := httptest.NewRecorder()

			receiver.ServeHTTP(w, req)

			if w.Code != http.StatusMethodNotAllowed {
				t.Errorf("expected 405, got %d", w.Code)
			}
		})
	}

	if len(handler.getEvents()) != 0 {
		t.Error("handler should not have received any events")
	}
}

func TestWebhookReceiver_ParseError(t *testing.T) {
	parser := &staticParser{err: errors.New("invalid payload")}
	handler := &recordingWebhookHandler{}
	receiver := NewWebhookReceiver(parser, handler)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
	w := httptest.NewRecorder()

	receiver.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
	if len(handler.getEvents()) != 0 {
		t.Error("handler should not have received any events")
	}
}

func TestWebhookReceiver_HandlerError(t *testing.T) {
	parser := &staticParser{events: sampleEvents()}
	handler := &recordingWebhookHandler{err: errors.New("db unavailable")}
	receiver := NewWebhookReceiver(parser, handler)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
	w := httptest.NewRecorder()

	receiver.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}

func TestWebhookReceiver_EmptyEvents(t *testing.T) {
	parser := &staticParser{events: []WebhookEvent{}}
	handler := &recordingWebhookHandler{}
	receiver := NewWebhookReceiver(parser, handler)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
	w := httptest.NewRecorder()

	receiver.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if len(handler.getEvents()) != 0 {
		t.Error("handler should not have received any events")
	}
}

func TestWebhookReceiver_EventFilter(t *testing.T) {
	events := []WebhookEvent{
		{Type: EventDelivered, MessageID: "msg-001"},
		{Type: EventBounced, MessageID: "msg-002"},
		{Type: EventOpened, MessageID: "msg-003"},
		{Type: EventClicked, MessageID: "msg-004"},
	}

	parser := &staticParser{events: events}
	handler := &recordingWebhookHandler{}
	receiver := NewWebhookReceiver(parser, handler,
		WithEventFilter(EventBounced, EventComplained),
	)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
	w := httptest.NewRecorder()

	receiver.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	got := handler.getEvents()
	if len(got) != 1 {
		t.Fatalf("expected 1 filtered event, got %d", len(got))
	}
	if got[0].Type != EventBounced {
		t.Errorf("expected EventBounced, got %s", got[0].Type)
	}
}

func TestWebhookReceiver_EventFilterStillReturns200(t *testing.T) {
	// All events are filtered out — should still return 200.
	parser := &staticParser{events: []WebhookEvent{
		{Type: EventOpened, MessageID: "msg-001"},
	}}
	handler := &recordingWebhookHandler{}
	receiver := NewWebhookReceiver(parser, handler,
		WithEventFilter(EventBounced),
	)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
	w := httptest.NewRecorder()

	receiver.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	if len(handler.getEvents()) != 0 {
		t.Error("expected no events dispatched")
	}
}

func TestWebhookReceiver_WithLogger(t *testing.T) {
	logger := &recordingLogger{}
	parser := &staticParser{events: sampleEvents()}
	handler := &recordingWebhookHandler{}

	receiver := NewWebhookReceiver(parser, handler, WithWebhookLogger(logger))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
	w := httptest.NewRecorder()

	receiver.ServeHTTP(w, req)

	entries := logger.getEntries()
	if len(entries) == 0 {
		t.Error("expected log entries from webhook receiver")
	}
	// Should have a debug entry about receiving events.
	found := false
	for _, e := range entries {
		if e.msg == "webhook received" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'webhook received' log entry")
	}
}

func TestWebhookReceiver_NilLogger(t *testing.T) {
	parser := &staticParser{events: sampleEvents()}
	handler := &recordingWebhookHandler{}
	receiver := NewWebhookReceiver(parser, handler, WithWebhookLogger(nil))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
	w := httptest.NewRecorder()

	// Should not panic.
	receiver.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

func TestWebhookReceiver_ParseErrorLogged(t *testing.T) {
	logger := &recordingLogger{}
	parser := &staticParser{err: errors.New("bad json")}
	handler := &recordingWebhookHandler{}

	receiver := NewWebhookReceiver(parser, handler, WithWebhookLogger(logger))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
	w := httptest.NewRecorder()

	receiver.ServeHTTP(w, req)

	entries := logger.getEntries()
	found := false
	for _, e := range entries {
		if e.level == logLevelError && e.msg == "webhook parse failed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error log for parse failure")
	}
}

func TestWebhookReceiver_HandlerErrorLogged(t *testing.T) {
	logger := &recordingLogger{}
	parser := &staticParser{events: sampleEvents()}
	handler := &recordingWebhookHandler{err: errors.New("handler failed")}

	receiver := NewWebhookReceiver(parser, handler, WithWebhookLogger(logger))

	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
	w := httptest.NewRecorder()

	receiver.ServeHTTP(w, req)

	entries := logger.getEntries()
	found := false
	for _, e := range entries {
		if e.level == logLevelError && e.msg == "webhook handler failed" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error log for handler failure")
	}
}

// --- WebhookHandlerFunc tests ---

func TestWebhookHandlerFunc(t *testing.T) {
	var called atomic.Bool

	fn := WebhookHandlerFunc(func(_ context.Context, event WebhookEvent) error {
		called.Store(true)
		if event.Type != EventDelivered {
			t.Errorf("expected EventDelivered, got %s", event.Type)
		}
		return nil
	})

	err := fn.HandleEvent(context.Background(), WebhookEvent{Type: EventDelivered})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called.Load() {
		t.Error("handler function was not called")
	}
}

func TestWebhookHandlerFunc_ReturnsError(t *testing.T) {
	handlerErr := errors.New("processing failed")
	fn := WebhookHandlerFunc(func(_ context.Context, _ WebhookEvent) error {
		return handlerErr
	})

	err := fn.HandleEvent(context.Background(), WebhookEvent{})
	if !errors.Is(err, handlerErr) {
		t.Errorf("expected %v, got %v", handlerErr, err)
	}
}

// --- WebhookEvent field tests ---

func TestWebhookEvent_AllFields(t *testing.T) {
	event := WebhookEvent{
		Type:       EventClicked,
		MessageID:  "msg-123",
		Recipient:  "user@example.com",
		Timestamp:  time.Date(2026, 3, 23, 12, 0, 0, 0, time.UTC),
		Provider:   "sendgrid",
		Reason:     "",
		URL:        "https://example.com/offer",
		UserAgent:  "Mozilla/5.0",
		IP:         "1.2.3.4",
		Tags:       []string{"campaign-march"},
		Metadata:   map[string]string{"custom": "value"},
		RawPayload: []byte(`{"event":"click"}`),
	}

	if event.Type != EventClicked {
		t.Errorf("unexpected type: %s", event.Type)
	}
	if event.URL != "https://example.com/offer" {
		t.Errorf("unexpected URL: %s", event.URL)
	}
	if len(event.Tags) != 1 || event.Tags[0] != "campaign-march" {
		t.Errorf("unexpected tags: %v", event.Tags)
	}
	if event.Metadata["custom"] != "value" {
		t.Errorf("unexpected metadata: %v", event.Metadata)
	}
}

// --- Concurrency test ---

func TestWebhookReceiver_Concurrency(t *testing.T) {
	events := sampleEvents()
	parser := &staticParser{events: events}
	handler := &recordingWebhookHandler{}
	receiver := NewWebhookReceiver(parser, handler)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)

	for range n {
		go func() {
			defer wg.Done()
			req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
			w := httptest.NewRecorder()
			receiver.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", w.Code)
			}
		}()
	}
	wg.Wait()

	got := handler.getEvents()
	expected := n * len(events)
	if len(got) != expected {
		t.Errorf("expected %d events, got %d", expected, len(got))
	}
}

// --- Handler error does not stop dispatch ---

// Per the receiver's documented contract, a failure mid-batch must NOT
// short-circuit the remaining events: the receiver returns 500 (so the
// provider retries the batch), but every event in the batch is still
// dispatched so it has the chance to succeed before the retry. Handlers
// are expected to be idempotent.
func TestWebhookReceiver_HandlerErrorDoesNotStopDispatch(t *testing.T) {
	events := []WebhookEvent{
		{Type: EventDelivered, MessageID: "msg-001"},
		{Type: EventBounced, MessageID: "msg-002"},
		{Type: EventOpened, MessageID: "msg-003"},
	}

	callCount := 0
	handler := WebhookHandlerFunc(func(_ context.Context, _ WebhookEvent) error {
		callCount++
		if callCount == 2 {
			return errors.New("fail on second event")
		}
		return nil
	})

	parser := &staticParser{events: events}
	receiver := NewWebhookReceiver(parser, handler)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
	w := httptest.NewRecorder()

	receiver.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	if callCount != len(events) {
		t.Errorf("expected handler called %d times (all events), got %d", len(events), callCount)
	}
}

// --- Integration with http.ServeMux ---

func TestWebhookReceiver_ServeMuxIntegration(t *testing.T) {
	parser := &staticParser{events: []WebhookEvent{
		{Type: EventDelivered, MessageID: "msg-001", Provider: "test"},
	}}
	handler := &recordingWebhookHandler{}
	receiver := NewWebhookReceiver(parser, handler)

	mux := http.NewServeMux()
	mux.Handle("/webhooks/email", receiver)

	server := httptest.NewServer(mux)
	defer server.Close()

	resp, err := http.Post(server.URL+"/webhooks/email", "application/json",
		strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("POST failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	got := handler.getEvents()
	if len(got) != 1 {
		t.Fatalf("expected 1 event, got %d", len(got))
	}
	if got[0].Provider != "test" {
		t.Errorf("expected provider 'test', got %s", got[0].Provider)
	}
}

// --- EventType constants test ---

func TestEventTypeConstants(t *testing.T) {
	types := []struct {
		et   EventType
		want string
	}{
		{EventDelivered, "delivered"},
		{EventBounced, "bounced"},
		{EventDeferred, "deferred"},
		{EventOpened, "opened"},
		{EventClicked, "clicked"},
		{EventComplained, "complained"},
		{EventUnsubscribed, "unsubscribed"},
		{EventDropped, "dropped"},
	}

	for _, tt := range types {
		if string(tt.et) != tt.want {
			t.Errorf("EventType %q != %q", tt.et, tt.want)
		}
	}
}

// --- Benchmark ---

func BenchmarkWebhookReceiver(b *testing.B) {
	events := make([]WebhookEvent, 10)
	for i := range events {
		events[i] = WebhookEvent{
			Type:      EventDelivered,
			MessageID: fmt.Sprintf("msg-%d", i),
			Recipient: "user@example.com",
			Provider:  "bench",
		}
	}

	parser := &staticParser{events: events}
	handler := WebhookHandlerFunc(func(_ context.Context, _ WebhookEvent) error {
		return nil
	})
	receiver := NewWebhookReceiver(parser, handler)

	for b.Loop() {
		req := httptest.NewRequest(http.MethodPost, "/webhooks/email", nil)
		w := httptest.NewRecorder()
		receiver.ServeHTTP(w, req)
	}
}
