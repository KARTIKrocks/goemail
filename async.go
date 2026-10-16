package email

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/KARTIKrocks/go-worker"
)

var (
	// ErrQueueFull is returned by AsyncSender.Send when the queue buffer is full.
	ErrQueueFull = errors.New("email: async queue is full")

	// ErrQueueClosed is returned by AsyncSender.Send after Close has been called.
	ErrQueueClosed = errors.New("email: async queue is closed")
)

// AsyncSender wraps a Sender and sends emails asynchronously via a buffered
// queue processed by background worker goroutines. It implements the Sender
// interface so it can be used anywhere a Sender is expected.
//
// A Sender that panics fails that one send with an error wrapping ErrPanicked
// instead of crashing the process.
type AsyncSender struct {
	sender       Sender
	pool         *worker.Pool
	queueSize    int
	workers      int
	logger       Logger
	errorHandler func(ctx context.Context, email *Email, err error)
	once         sync.Once
	closeErr     error // stored by Close so repeat calls return the same error
}

// AsyncOption configures an AsyncSender.
type AsyncOption func(*AsyncSender)

// WithQueueSize sets the channel buffer size for the async queue.
// Default: 100.
func WithQueueSize(size int) AsyncOption {
	return func(a *AsyncSender) {
		if size > 0 {
			a.queueSize = size
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
		sender:    sender,
		queueSize: 100,
		workers:   1,
		logger:    NoOpLogger{},
	}

	for _, opt := range opts {
		opt(a)
	}

	pool, err := worker.NewPool(
		worker.WithWorkers(a.workers),
		worker.WithQueueSize(a.queueSize),
		worker.WithLogger(poolLogger{a.logger}),
		worker.WithHooks(worker.Hooks{OnJobFailed: a.jobFailed}),
	)
	if err != nil {
		panic("email: " + err.Error()) // unreachable: the options only accept positive sizes
	}
	a.pool = pool

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
	if a.pool.IsClosed() {
		return ErrQueueClosed
	}

	if err := e.Validate(); err != nil {
		return err
	}

	return poolErr(a.pool.TrySubmitJob(&asyncJob{
		sender: a.sender,
		ctx:    context.WithoutCancel(ctx),
		email:  cloneEmail(e), // the caller may reuse e once Send returns
	}))
}

// SendWait validates the email eagerly, enqueues it, and blocks until the
// background worker has finished sending (or the context is cancelled).
func (a *AsyncSender) SendWait(ctx context.Context, e *Email) error {
	if a.pool.IsClosed() {
		return ErrQueueClosed
	}

	if err := e.Validate(); err != nil {
		return err
	}

	e = cloneEmail(e) // the job can outlive a cancelled ctx
	return poolErr(a.pool.SubmitWait(ctx, func(ctx context.Context) error {
		return wrapSend(a.sender.Send(ctx, e))
	}))
}

// Close stops accepting new emails, waits for in-flight work to complete,
// and closes the underlying sender. It is safe to call multiple times.
func (a *AsyncSender) Close() error {
	a.once.Do(func() {
		_ = a.pool.Close()
		a.closeErr = a.sender.Close()
	})
	return a.closeErr
}

// asyncJob is a fire-and-forget send. SendWait does not use it, so jobFailed
// reports only the failures nobody is waiting on.
type asyncJob struct {
	sender Sender
	ctx    context.Context
	email  *Email
}

func (j *asyncJob) Process(context.Context) error {
	return wrapSend(j.sender.Send(j.ctx, j.email)) //nolint:contextcheck // the caller's detached context, as documented on Send
}

func (a *AsyncSender) jobFailed(job worker.Job, err error, _ time.Duration) {
	j, ok := job.(*asyncJob)
	if !ok {
		return
	}
	err = poolErr(err)
	a.logger.Error("async email send failed",
		"error", err,
		"to", j.email.To,
		"subject", j.email.Subject,
	)
	if a.errorHandler != nil {
		a.errorHandler(j.ctx, j.email, err)
	}
}

// sendError marks an error returned by the wrapped Sender, so poolErr never
// mistakes it for one raised by the pool itself, even when the Sender runs a
// go-worker pool of its own.
type sendError struct{ err error }

func (e *sendError) Error() string { return e.err.Error() }

func wrapSend(err error) error {
	if err == nil {
		return nil
	}
	return &sendError{err}
}

// poolErr turns a pool result back into what callers expect: the Sender's own
// error untouched, a recovered panic wrapped in ErrPanicked (the error the
// recovery middleware also returns), and the pool's sentinels mapped to the
// AsyncSender ones.
func poolErr(err error) error {
	if se, ok := errors.AsType[*sendError](err); ok {
		return se.err
	}
	var pe *worker.PanicError
	switch {
	case errors.As(err, &pe):
		return fmt.Errorf("%w: %w", ErrPanicked, err)
	case errors.Is(err, worker.ErrPoolFull):
		return ErrQueueFull
	case errors.Is(err, worker.ErrPoolClosed):
		return ErrQueueClosed
	}
	return err
}

// poolLogger forwards the pool's warnings and errors, such as a recovered
// panic's stack trace, and drops its lifecycle messages.
type poolLogger struct{ Logger }

func (poolLogger) Debug(string, ...any) {}
func (poolLogger) Info(string, ...any)  {}
