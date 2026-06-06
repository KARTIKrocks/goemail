package email

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// slowSender blocks on Send until its channel is closed or a value is sent.
type slowSender struct {
	gate chan struct{}
	mu   sync.Mutex
	sent []*Email
}

func newSlowSender() *slowSender {
	return &slowSender{gate: make(chan struct{})}
}

func (s *slowSender) Send(_ context.Context, e *Email) error {
	<-s.gate // block until released
	s.mu.Lock()
	s.sent = append(s.sent, e)
	s.mu.Unlock()
	return nil
}

func (s *slowSender) Close() error { return nil }

func (s *slowSender) release() { close(s.gate) }

// countSender counts successful sends in a thread-safe way.
type countSender struct {
	count atomic.Int64
}

func (c *countSender) Send(_ context.Context, _ *Email) error {
	c.count.Add(1)
	return nil
}

func (c *countSender) Close() error { return nil }

// --- Tests ---

func TestAsyncSender_Send(t *testing.T) {
	mock := NewMockSender()
	a := NewAsyncSender(mock)

	e := testEmail()
	if err := a.Send(context.Background(), e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := a.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}

	if mock.GetEmailCount() != 1 {
		t.Fatalf("expected 1 email, got %d", mock.GetEmailCount())
	}
}

func TestAsyncSender_SendWait_Success(t *testing.T) {
	mock := NewMockSender()
	a := NewAsyncSender(mock)
	defer a.Close()

	e := testEmail()
	if err := a.SendWait(context.Background(), e); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.GetEmailCount() != 1 {
		t.Fatalf("expected 1 email, got %d", mock.GetEmailCount())
	}
}

func TestAsyncSender_SendWait_Failure(t *testing.T) {
	want := errors.New("boom")
	a := NewAsyncSender(&failSender{err: want})
	defer a.Close()

	e := testEmail()
	err := a.SendWait(context.Background(), e)
	if !errors.Is(err, want) {
		t.Fatalf("expected %v, got %v", want, err)
	}
}

func TestAsyncSender_Send_ValidationError(t *testing.T) {
	mock := NewMockSender()
	a := NewAsyncSender(mock)
	defer a.Close()

	// Email with no recipients => validation fails
	e := &Email{
		From:    "from@example.com",
		Subject: "Test",
		Body:    "Hello",
		Headers: make(map[string]string),
	}
	err := a.Send(context.Background(), e)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !errors.Is(err, ErrNoRecipients) {
		t.Fatalf("expected ErrNoRecipients, got %v", err)
	}
}

func TestAsyncSender_Send_QueueFull(t *testing.T) {
	slow := newSlowSender()
	a := NewAsyncSender(slow, WithQueueSize(1))

	e := testEmail()

	// Fill the queue — worker is blocked on slow sender
	if err := a.Send(context.Background(), e); err != nil {
		t.Fatalf("first send: %v", err)
	}

	// Second send should fill the buffer
	if err := a.Send(context.Background(), e); err != nil {
		// Queue size 1 + 1 in worker = might still have room;
		// try a third
		t.Logf("second send succeeded unexpectedly or failed: %v", err)
	}

	// Keep trying until we get ErrQueueFull
	for range 100 {
		err := a.Send(context.Background(), e)
		if errors.Is(err, ErrQueueFull) {
			slow.release()
			a.Close()
			return
		}
	}
	slow.release()
	a.Close()
	t.Fatal("expected ErrQueueFull but never received it")
}

func TestAsyncSender_Send_AfterClose(t *testing.T) {
	mock := NewMockSender()
	a := NewAsyncSender(mock)
	a.Close()

	e := testEmail()
	err := a.Send(context.Background(), e)
	if !errors.Is(err, ErrQueueClosed) {
		t.Fatalf("expected ErrQueueClosed, got %v", err)
	}
}

func TestAsyncSender_Close_DrainsQueue(t *testing.T) {
	mock := NewMockSender()
	a := NewAsyncSender(mock, WithQueueSize(50))

	for range 20 {
		e := testEmail()
		if err := a.Send(context.Background(), e); err != nil {
			t.Fatalf("send: %v", err)
		}
	}

	if err := a.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if mock.GetEmailCount() != 20 {
		t.Fatalf("expected 20 emails, got %d", mock.GetEmailCount())
	}
}

func TestAsyncSender_Close_Idempotent(t *testing.T) {
	mock := NewMockSender()
	a := NewAsyncSender(mock)

	if err := a.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}

func TestAsyncSender_ErrorHandler(t *testing.T) {
	want := errors.New("send failed")
	var (
		mu        sync.Mutex
		gotEmail  *Email
		gotErr    error
		handlerCh = make(chan struct{})
	)

	handler := func(_ context.Context, e *Email, err error) {
		mu.Lock()
		gotEmail = e
		gotErr = err
		mu.Unlock()
		close(handlerCh)
	}

	a := NewAsyncSender(&failSender{err: want}, WithErrorHandler(handler))

	e := testEmail()
	if err := a.Send(context.Background(), e); err != nil {
		t.Fatalf("send: %v", err)
	}

	// Wait for handler to be called
	select {
	case <-handlerCh:
	case <-time.After(5 * time.Second):
		t.Fatal("error handler not called within timeout")
	}

	a.Close()

	mu.Lock()
	defer mu.Unlock()
	if !errors.Is(gotErr, want) {
		t.Fatalf("expected %v, got %v", want, gotErr)
	}
	if gotEmail == nil {
		t.Fatal("expected email in error handler")
	}
}

func TestAsyncSender_MultipleWorkers(t *testing.T) {
	cs := &countSender{}
	a := NewAsyncSender(cs, WithWorkers(3), WithQueueSize(200))

	const n = 100
	for range n {
		e := testEmail()
		if err := a.Send(context.Background(), e); err != nil {
			t.Fatalf("send: %v", err)
		}
	}

	if err := a.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	if got := cs.count.Load(); got != n {
		t.Fatalf("expected %d sends, got %d", n, got)
	}
}

func TestAsyncSender_ImplementsSender(t *testing.T) {
	var _ Sender = (*AsyncSender)(nil)
}

func TestAsyncSender_WithMiddleware(t *testing.T) {
	mock := NewMockSender()
	metrics := &recordingMetrics{}

	wrapped := Chain(mock, WithMetrics(metrics))
	a := NewAsyncSender(wrapped)

	e := testEmail()
	if err := a.SendWait(context.Background(), e); err != nil {
		t.Fatalf("send: %v", err)
	}

	a.Close()

	if metrics.successes.Load() != 1 {
		t.Fatalf("expected 1 success, got %d", metrics.successes.Load())
	}
}

func TestAsyncSender_SendWait_ContextCancel(t *testing.T) {
	slow := newSlowSender()
	a := NewAsyncSender(slow, WithQueueSize(1))

	ctx, cancel := context.WithCancel(context.Background())

	// Fill the queue so SendWait blocks on enqueue
	e := testEmail()
	_ = a.Send(context.Background(), e)
	_ = a.Send(context.Background(), e)

	// Cancel the context
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := a.SendWait(ctx, testEmail())
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	slow.release()
	a.Close()
}

func TestAsyncSender_Concurrency(t *testing.T) {
	cs := &countSender{}
	a := NewAsyncSender(cs, WithWorkers(4), WithQueueSize(200))

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)

	for range n {
		go func() {
			defer wg.Done()
			e := testEmail()
			_ = a.Send(context.Background(), e)
		}()
	}

	wg.Wait()
	a.Close()

	if got := cs.count.Load(); got != n {
		t.Fatalf("expected %d sends, got %d", n, got)
	}
}
