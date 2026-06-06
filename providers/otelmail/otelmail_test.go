package otelmail_test

import (
	"context"
	"errors"
	"testing"

	email "github.com/KARTIKrocks/goemail"
	"github.com/KARTIKrocks/goemail/providers/otelmail"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func newTestProvider() (*tracetest.InMemoryExporter, trace.TracerProvider) {
	exp := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exp))
	return exp, tp
}

func testEmail() *email.Email {
	return email.NewEmail().
		SetFrom("sender@example.com").
		AddTo("alice@example.com", "bob@example.com").
		AddCc("cc@example.com").
		SetSubject("Test Subject").
		SetBody("Hello")
}

// stubSender is a minimal Sender for testing.
type stubSender struct {
	err    error
	called bool
	ctx    context.Context
}

func (s *stubSender) Send(ctx context.Context, _ *email.Email) error {
	s.called = true
	s.ctx = ctx
	return s.err
}

func (s *stubSender) Close() error { return nil }

// closeSender tracks Close() calls.
type closeSender struct {
	closed bool
}

func (s *closeSender) Send(context.Context, *email.Email) error { return nil }
func (s *closeSender) Close() error {
	s.closed = true
	return nil
}

func TestWithTracing_Success(t *testing.T) {
	exp, tp := newTestProvider()
	inner := &stubSender{}

	mw := otelmail.WithTracing(otelmail.WithTracerProvider(tp))
	sender := mw(inner)

	e := testEmail()
	err := sender.Send(context.Background(), e)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Name != "email.send" {
		t.Errorf("expected span name %q, got %q", "email.send", span.Name)
	}
	if span.SpanKind != trace.SpanKindClient {
		t.Errorf("expected SpanKindClient, got %v", span.SpanKind)
	}
	if span.Status.Code != codes.Ok {
		t.Errorf("expected status Ok, got %v", span.Status.Code)
	}

	assertAttr(t, span.Attributes, "email.from", "sender@example.com")
	assertAttr(t, span.Attributes, "email.to", "alice@example.com,bob@example.com")
	assertAttr(t, span.Attributes, "email.subject", "Test Subject")
	assertAttrInt(t, span.Attributes, "email.recipients.count", 3) // 2 To + 1 Cc
}

func TestWithTracing_Failure(t *testing.T) {
	exp, tp := newTestProvider()
	sendErr := errors.New("smtp: connection refused")
	inner := &stubSender{err: sendErr}

	mw := otelmail.WithTracing(otelmail.WithTracerProvider(tp))
	sender := mw(inner)

	err := sender.Send(context.Background(), testEmail())
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected %v, got %v", sendErr, err)
	}

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}

	span := spans[0]
	if span.Status.Code != codes.Error {
		t.Errorf("expected status Error, got %v", span.Status.Code)
	}
	if span.Status.Description != sendErr.Error() {
		t.Errorf("expected status description %q, got %q", sendErr.Error(), span.Status.Description)
	}

	// Check error was recorded as event
	found := false
	for _, ev := range span.Events {
		if ev.Name == "exception" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected exception event to be recorded")
	}
}

func TestWithTracing_CustomOptions(t *testing.T) {
	exp, tp := newTestProvider()
	inner := &stubSender{}

	mw := otelmail.WithTracing(
		otelmail.WithTracerProvider(tp),
		otelmail.WithTracerName("my-app"),
		otelmail.WithSpanName("mail.dispatch"),
	)
	sender := mw(inner)

	_ = sender.Send(context.Background(), testEmail())

	spans := exp.GetSpans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span, got %d", len(spans))
	}
	if spans[0].Name != "mail.dispatch" {
		t.Errorf("expected span name %q, got %q", "mail.dispatch", spans[0].Name)
	}
	if spans[0].InstrumentationScope.Name != "my-app" {
		t.Errorf("expected tracer name %q, got %q", "my-app", spans[0].InstrumentationScope.Name)
	}
}

func TestWithTracing_PropagatesContext(t *testing.T) {
	_, tp := newTestProvider()
	inner := &stubSender{}

	mw := otelmail.WithTracing(otelmail.WithTracerProvider(tp))
	sender := mw(inner)

	_ = sender.Send(context.Background(), testEmail())

	if !inner.called {
		t.Fatal("inner sender was not called")
	}

	// The context passed to the inner sender should contain an active span.
	span := trace.SpanFromContext(inner.ctx)
	if !span.SpanContext().IsValid() {
		t.Error("expected a valid span in the context passed to the inner sender")
	}
}

func TestWithTracing_Close(t *testing.T) {
	inner := &closeSender{}
	_, tp := newTestProvider()

	mw := otelmail.WithTracing(otelmail.WithTracerProvider(tp))
	sender := mw(inner)

	if err := sender.Close(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !inner.closed {
		t.Error("expected Close to be delegated to inner sender")
	}
}

// helpers

func assertAttr(t *testing.T, attrs []attribute.KeyValue, key, want string) {
	t.Helper()
	for _, a := range attrs {
		if string(a.Key) == key {
			if got := a.Value.AsString(); got != want {
				t.Errorf("attribute %s: expected %q, got %q", key, want, got)
			}
			return
		}
	}
	t.Errorf("attribute %s not found", key)
}

func assertAttrInt(t *testing.T, attrs []attribute.KeyValue, key string, want int) {
	t.Helper()
	for _, a := range attrs {
		if string(a.Key) == key {
			if got := a.Value.AsInt64(); got != int64(want) {
				t.Errorf("attribute %s: expected %d, got %d", key, want, got)
			}
			return
		}
	}
	t.Errorf("attribute %s not found", key)
}
