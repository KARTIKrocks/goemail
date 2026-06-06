package email

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// --- Test helpers ---

const logLevelError = "error"

type recordingLogger struct {
	mu      sync.Mutex
	entries []logEntry
}

type logEntry struct {
	level string
	msg   string
	args  []any
}

func (l *recordingLogger) Debug(msg string, args ...any) {
	l.record("debug", msg, args)
}
func (l *recordingLogger) Info(msg string, args ...any) {
	l.record("info", msg, args)
}
func (l *recordingLogger) Warn(msg string, args ...any) {
	l.record("warn", msg, args)
}
func (l *recordingLogger) Error(msg string, args ...any) {
	l.record(logLevelError, msg, args)
}
func (l *recordingLogger) With(_ ...any) Logger { return l }

func (l *recordingLogger) record(level, msg string, args []any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.entries = append(l.entries, logEntry{level: level, msg: msg, args: args})
}

func (l *recordingLogger) getEntries() []logEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	entries := make([]logEntry, len(l.entries))
	copy(entries, l.entries)
	return entries
}

type recordingMetrics struct {
	attempts  atomic.Int64
	successes atomic.Int64
	failures  atomic.Int64
	durations atomic.Int64
}

func (m *recordingMetrics) IncSendAttempt()                     { m.attempts.Add(1) }
func (m *recordingMetrics) IncSendSuccess()                     { m.successes.Add(1) }
func (m *recordingMetrics) IncSendFailure()                     { m.failures.Add(1) }
func (m *recordingMetrics) ObserveSendDuration(_ time.Duration) { m.durations.Add(1) }

type panicSender struct{}

func (p *panicSender) Send(_ context.Context, _ *Email) error {
	panic("test panic")
}
func (p *panicSender) Close() error { return nil }

type failSender struct {
	err error
}

func (f *failSender) Send(_ context.Context, _ *Email) error {
	return f.err
}
func (f *failSender) Close() error { return nil }

func testEmail() *Email {
	e := NewEmail().
		SetFrom("from@example.com").
		AddTo("to@example.com").
		SetSubject("Test").
		SetBody("Hello")
	built, _ := e.Build()
	return built
}

// --- Chain tests ---

func TestChain_Order(t *testing.T) {
	var order []string
	makeMW := func(name string) Middleware {
		return func(next Sender) Sender {
			return &orderSender{next: next, name: name, order: &order}
		}
	}

	mock := NewMockSender()
	wrapped := Chain(mock, makeMW("A"), makeMW("B"), makeMW("C"))
	_ = wrapped.Send(context.Background(), testEmail())

	if len(order) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(order))
	}
	expected := []string{"A", "B", "C"}
	for i, name := range expected {
		if order[i] != name {
			t.Errorf("position %d: expected %s, got %s", i, name, order[i])
		}
	}
}

type orderSender struct {
	next  Sender
	name  string
	order *[]string
}

func (s *orderSender) Send(ctx context.Context, e *Email) error {
	*s.order = append(*s.order, s.name)
	return s.next.Send(ctx, e)
}
func (s *orderSender) Close() error { return s.next.Close() }

func TestChain_Empty(t *testing.T) {
	mock := NewMockSender()
	wrapped := Chain(mock)
	if wrapped != mock {
		t.Error("Chain with no middlewares should return the original sender")
	}
}

// --- WithLogging tests ---

func TestWithLogging_Success(t *testing.T) {
	logger := &recordingLogger{}
	mock := NewMockSender()
	wrapped := Chain(mock, WithLogging(logger))

	err := wrapped.Send(context.Background(), testEmail())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries := logger.getEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(entries))
	}
	if entries[0].level != "info" || entries[0].msg != "sending email" {
		t.Errorf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].level != "info" || entries[1].msg != "email sent successfully" {
		t.Errorf("unexpected second entry: %+v", entries[1])
	}
}

func TestWithLogging_Failure(t *testing.T) {
	logger := &recordingLogger{}
	sendErr := errors.New("smtp error")
	sender := &failSender{err: sendErr}
	wrapped := Chain(sender, WithLogging(logger))

	err := wrapped.Send(context.Background(), testEmail())
	if !errors.Is(err, sendErr) {
		t.Fatalf("expected smtp error, got: %v", err)
	}

	entries := logger.getEntries()
	if len(entries) != 2 {
		t.Fatalf("expected 2 log entries, got %d", len(entries))
	}
	if entries[1].level != logLevelError || entries[1].msg != "email send failed" {
		t.Errorf("unexpected error entry: %+v", entries[1])
	}
}

func TestWithLogging_NilLogger(t *testing.T) {
	mock := NewMockSender()
	wrapped := Chain(mock, WithLogging(nil))

	// Should not panic
	err := wrapped.Send(context.Background(), testEmail())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- WithRecovery tests ---

func TestWithRecovery_NoPanic(t *testing.T) {
	mock := NewMockSender()
	wrapped := Chain(mock, WithRecovery())

	err := wrapped.Send(context.Background(), testEmail())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mock.GetEmailCount() != 1 {
		t.Error("expected 1 email sent")
	}
}

func TestWithRecovery_CatchesPanic(t *testing.T) {
	wrapped := Chain(&panicSender{}, WithRecovery())

	err := wrapped.Send(context.Background(), testEmail())
	if err == nil {
		t.Fatal("expected error from panic")
	}
	if !errors.Is(err, ErrPanicked) {
		t.Errorf("expected ErrPanicked, got: %v", err)
	}
}

func TestWithRecovery_Close(t *testing.T) {
	mock := NewMockSender()
	wrapped := Chain(mock, WithRecovery())

	err := wrapped.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- WithHooks tests ---

func TestWithHooks_AllCallbacks(t *testing.T) {
	var (
		onSendCalled    atomic.Bool
		onSuccessCalled atomic.Bool
		successDuration time.Duration
	)

	hooks := SendHooks{
		OnSend: func(_ context.Context, _ *Email) {
			onSendCalled.Store(true)
		},
		OnSuccess: func(_ context.Context, _ *Email, d time.Duration) {
			onSuccessCalled.Store(true)
			successDuration = d
		},
		OnFailure: func(_ context.Context, _ *Email, _ time.Duration, _ error) {
			t.Error("OnFailure should not be called on success")
		},
	}

	mock := NewMockSender()
	wrapped := Chain(mock, WithHooks(hooks))

	err := wrapped.Send(context.Background(), testEmail())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !onSendCalled.Load() {
		t.Error("OnSend was not called")
	}
	if !onSuccessCalled.Load() {
		t.Error("OnSuccess was not called")
	}
	if successDuration <= 0 {
		t.Error("expected positive duration")
	}
}

func TestWithHooks_OnFailure(t *testing.T) {
	var (
		onFailureCalled atomic.Bool
		capturedErr     error
	)

	sendErr := errors.New("send failed")
	hooks := SendHooks{
		OnFailure: func(_ context.Context, _ *Email, _ time.Duration, err error) {
			onFailureCalled.Store(true)
			capturedErr = err
		},
	}

	sender := &failSender{err: sendErr}
	wrapped := Chain(sender, WithHooks(hooks))

	_ = wrapped.Send(context.Background(), testEmail())
	if !onFailureCalled.Load() {
		t.Error("OnFailure was not called")
	}
	if !errors.Is(capturedErr, sendErr) {
		t.Errorf("expected %v, got %v", sendErr, capturedErr)
	}
}

func TestWithHooks_NilCallbacks(t *testing.T) {
	mock := NewMockSender()
	wrapped := Chain(mock, WithHooks(SendHooks{}))

	// Should not panic with nil callbacks
	err := wrapped.Send(context.Background(), testEmail())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- WithMetrics tests ---

func TestWithMetrics_Success(t *testing.T) {
	metrics := &recordingMetrics{}
	mock := NewMockSender()
	wrapped := Chain(mock, WithMetrics(metrics))

	err := wrapped.Send(context.Background(), testEmail())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.attempts.Load() != 1 {
		t.Errorf("expected 1 attempt, got %d", metrics.attempts.Load())
	}
	if metrics.successes.Load() != 1 {
		t.Errorf("expected 1 success, got %d", metrics.successes.Load())
	}
	if metrics.failures.Load() != 0 {
		t.Errorf("expected 0 failures, got %d", metrics.failures.Load())
	}
	if metrics.durations.Load() != 1 {
		t.Errorf("expected 1 duration observation, got %d", metrics.durations.Load())
	}
}

func TestWithMetrics_Failure(t *testing.T) {
	metrics := &recordingMetrics{}
	sender := &failSender{err: errors.New("fail")}
	wrapped := Chain(sender, WithMetrics(metrics))

	_ = wrapped.Send(context.Background(), testEmail())

	if metrics.attempts.Load() != 1 {
		t.Errorf("expected 1 attempt, got %d", metrics.attempts.Load())
	}
	if metrics.successes.Load() != 0 {
		t.Errorf("expected 0 successes, got %d", metrics.successes.Load())
	}
	if metrics.failures.Load() != 1 {
		t.Errorf("expected 1 failure, got %d", metrics.failures.Load())
	}
}

// --- NoOpMetricsCollector test ---

func TestNoOpMetricsCollector(t *testing.T) {
	var c NoOpMetricsCollector
	// Should not panic
	c.IncSendAttempt()
	c.IncSendSuccess()
	c.IncSendFailure()
	c.ObserveSendDuration(time.Second)
}

// --- Concurrency test ---

func TestMiddleware_Concurrency(t *testing.T) {
	metrics := &recordingMetrics{}
	mock := NewMockSender()
	wrapped := Chain(mock,
		WithRecovery(),
		WithLogging(NoOpLogger{}),
		WithMetrics(metrics),
	)

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		go func(i int) {
			defer wg.Done()
			e := NewEmail().
				SetFrom("from@example.com").
				AddTo("to@example.com").
				SetSubject(fmt.Sprintf("Test %d", i)).
				SetBody("Hello")
			built, _ := e.Build()
			_ = wrapped.Send(context.Background(), built)
		}(i)
	}
	wg.Wait()

	if metrics.attempts.Load() != n {
		t.Errorf("expected %d attempts, got %d", n, metrics.attempts.Load())
	}
	if mock.GetEmailCount() != n {
		t.Errorf("expected %d emails, got %d", n, mock.GetEmailCount())
	}
}

// --- Close chain test ---

func TestMiddleware_CloseChain(t *testing.T) {
	var closed atomic.Bool
	mock := &closeRecordingSender{closed: &closed}

	wrapped := Chain(mock,
		WithRecovery(),
		WithLogging(NoOpLogger{}),
		WithHooks(SendHooks{}),
		WithMetrics(&recordingMetrics{}),
	)

	err := wrapped.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !closed.Load() {
		t.Error("Close did not propagate to the inner sender")
	}
}

type closeRecordingSender struct {
	closed *atomic.Bool
}

func (s *closeRecordingSender) Send(_ context.Context, _ *Email) error { return nil }
func (s *closeRecordingSender) Close() error {
	s.closed.Store(true)
	return nil
}

// --- NewMailerWithOptions test ---

func TestNewMailerWithOptions(t *testing.T) {
	metrics := &recordingMetrics{}
	mock := NewMockSender()

	mailer := NewMailerWithOptions(mock, "from@example.com",
		WithMiddleware(
			WithMetrics(metrics),
		),
	)

	err := mailer.Send(context.Background(), []string{"to@example.com"}, "Test", "Hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if metrics.attempts.Load() != 1 {
		t.Errorf("expected 1 attempt, got %d", metrics.attempts.Load())
	}
	if mock.GetEmailCount() != 1 {
		t.Errorf("expected 1 email, got %d", mock.GetEmailCount())
	}
}
