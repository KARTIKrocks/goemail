package email

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

var (
	// ErrQueueFull is returned by AsyncSender.Send when the queue buffer is full.
	ErrQueueFull = errors.New("email: async queue is full")

	// ErrQueueClosed is returned by AsyncSender.Send after Close has been called.
	ErrQueueClosed = errors.New("email: async queue is closed")
)

type asyncTask struct {
	ctx   context.Context
	email *Email
	errCh chan<- error // nil = fire-and-forget, non-nil = SendWait
}

// AsyncSender wraps a Sender and sends emails asynchronously via a buffered
// queue processed by background worker goroutines. It implements the Sender
// interface so it can be used anywhere a Sender is expected.
type AsyncSender struct {
	sender       Sender
	queue        chan *asyncTask
	workers      int
	logger       Logger
	errorHandler func(ctx context.Context, email *Email, err error)
	wg           sync.WaitGroup
	once         sync.Once
	closed       atomic.Bool
	closeMu      sync.RWMutex // protects queue from concurrent send/close
	closeErr     error        // stored by Close so repeat calls return the same error
}

// AsyncOption configures an AsyncSender.
type AsyncOption func(*AsyncSender)

// WithQueueSize sets the channel buffer size for the async queue.
// Default: 100.
func WithQueueSize(size int) AsyncOption {
	return func(a *AsyncSender) {
		if size > 0 {
			a.queue = make(chan *asyncTask, size)
		}
	}
}

// WithWorkers sets the number of worker goroutines that drain the queue.
// Default: 1.
func WithWorkers(n int) AsyncOption {
	return func(a *AsyncSender) {
		if n > 0 {
			a.workers = n
		}
	}
}

// WithAsyncLogger sets the logger used for background error reporting.
func WithAsyncLogger(l Logger) AsyncOption {
	return func(a *AsyncSender) {
		if l != nil {
			a.logger = l
		}
	}
}

// WithErrorHandler sets a callback that is invoked when a fire-and-forget
// send fails in the background.
func WithErrorHandler(fn func(ctx context.Context, email *Email, err error)) AsyncOption {
	return func(a *AsyncSender) {
		a.errorHandler = fn
	}
}

// NewAsyncSender creates a new AsyncSender that wraps the given Sender.
// Workers are started immediately. Call Close to drain the queue and shut down.
func NewAsyncSender(sender Sender, opts ...AsyncOption) *AsyncSender {
	a := &AsyncSender{
		sender:  sender,
		queue:   make(chan *asyncTask, 100),
		workers: 1,
		logger:  NoOpLogger{},
	}

	for _, opt := range opts {
		opt(a)
	}

	a.wg.Add(a.workers)
	for range a.workers {
		go a.worker()
	}

	return a
}

// Send validates the email eagerly and enqueues it for asynchronous delivery.
// It returns ErrQueueFull if the buffer is full, or ErrQueueClosed if the
// sender has been closed. Send implements the Sender interface.
//
// The caller's context is detached with [context.WithoutCancel] before the
// task is queued: cancelling ctx after Send returns has no effect on the
// background delivery, but request-scoped values (trace IDs, etc.) are
// preserved. Use SendWait if you need to block on completion or propagate
// cancellation.
func (a *AsyncSender) Send(ctx context.Context, e *Email) error {
	a.closeMu.RLock()
	defer a.closeMu.RUnlock()

	if a.closed.Load() {
		return ErrQueueClosed
	}

	if err := e.Validate(); err != nil {
		return err
	}

	task := &asyncTask{ctx: context.WithoutCancel(ctx), email: e}

	select {
	case a.queue <- task:
		return nil
	default:
		return ErrQueueFull
	}
}

// SendWait validates the email eagerly, enqueues it, and blocks until the
// background worker has finished sending (or the context is cancelled).
func (a *AsyncSender) SendWait(ctx context.Context, e *Email) error {
	a.closeMu.RLock()
	if a.closed.Load() {
		a.closeMu.RUnlock()
		return ErrQueueClosed
	}

	if err := e.Validate(); err != nil {
		a.closeMu.RUnlock()
		return err
	}

	errCh := make(chan error, 1)
	task := &asyncTask{ctx: ctx, email: e, errCh: errCh}

	select {
	case a.queue <- task:
		a.closeMu.RUnlock()
	case <-ctx.Done():
		a.closeMu.RUnlock()
		return ctx.Err()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Close stops accepting new emails, waits for in-flight work to complete,
// and closes the underlying sender. It is safe to call multiple times.
func (a *AsyncSender) Close() error {
	a.once.Do(func() {
		a.closeMu.Lock()
		a.closed.Store(true)
		close(a.queue)
		a.closeMu.Unlock()
		a.wg.Wait()
		a.closeErr = a.sender.Close()
	})
	return a.closeErr
}

func (a *AsyncSender) worker() {
	defer a.wg.Done()
	for task := range a.queue {
		err := a.sender.Send(task.ctx, task.email)
		if task.errCh != nil {
			task.errCh <- err
			continue
		}
		if err != nil {
			a.logger.Error("async email send failed",
				"error", err,
				"to", task.email.To,
				"subject", task.email.Subject,
			)
			if a.errorHandler != nil {
				a.errorHandler(task.ctx, task.email, err)
			}
		}
	}
}
