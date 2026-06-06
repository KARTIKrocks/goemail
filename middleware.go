package email

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// Middleware wraps a Sender to add cross-cutting behavior.
// It follows the same pattern as net/http middleware.
type Middleware func(Sender) Sender

// Chain applies middlewares to a Sender in order.
// The first middleware in the list is the outermost (executed first).
//
//	wrapped := email.Chain(sender, loggingMW, recoveryMW, metricsMW)
//
// A call to wrapped.Send() executes: logging -> recovery -> metrics -> sender.
func Chain(sender Sender, middlewares ...Middleware) Sender {
	for i := len(middlewares) - 1; i >= 0; i-- {
		sender = middlewares[i](sender)
	}
	return sender
}

// ErrPanicked is returned when a panic is recovered by the recovery middleware.
var ErrPanicked = errors.New("email: panic recovered in send pipeline")

// SendHooks defines optional callbacks for email send lifecycle events.
// All fields are optional; nil callbacks are skipped.
// Callbacks must be safe for concurrent use.
type SendHooks struct {
	// OnSend is called before each send attempt.
	OnSend func(ctx context.Context, e *Email)

	// OnSuccess is called after a successful send, with the duration.
	OnSuccess func(ctx context.Context, e *Email, duration time.Duration)

	// OnFailure is called after a failed send, with the duration and error.
	OnFailure func(ctx context.Context, e *Email, duration time.Duration, err error)
}

// WithLogging returns a Middleware that logs send attempts using the
// provided Logger. It logs the start of each send, success with duration,
// and failure with duration and error.
func WithLogging(logger Logger) Middleware {
	if logger == nil {
		logger = NoOpLogger{}
	}
	return func(next Sender) Sender {
		return &loggingSender{next: next, logger: logger}
	}
}

type loggingSender struct {
	next   Sender
	logger Logger
}

func (s *loggingSender) Send(ctx context.Context, e *Email) error {
	s.logger.Info("sending email",
		"to", e.To,
		"subject", e.Subject,
		"from", e.From,
	)

	start := time.Now()
	err := s.next.Send(ctx, e)
	duration := time.Since(start)

	if err != nil {
		s.logger.Error("email send failed",
			"to", e.To,
			"subject", e.Subject,
			"duration", duration.String(),
			"error", err,
		)
	} else {
		s.logger.Info("email sent successfully",
			"to", e.To,
			"subject", e.Subject,
			"duration", duration.String(),
		)
	}
	return err
}

func (s *loggingSender) Close() error {
	return s.next.Close()
}

// WithRecovery returns a Middleware that catches panics in downstream
// Send calls and converts them to errors wrapping ErrPanicked.
func WithRecovery() Middleware {
	return func(next Sender) Sender {
		return &recoverySender{next: next}
	}
}

type recoverySender struct {
	next Sender
}

func (s *recoverySender) Send(ctx context.Context, e *Email) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrPanicked, r)
		}
	}()
	return s.next.Send(ctx, e)
}

func (s *recoverySender) Close() error {
	return s.next.Close()
}

// WithHooks returns a Middleware that invokes the provided callbacks
// at each stage of the send lifecycle. Nil callbacks are safely skipped.
func WithHooks(hooks SendHooks) Middleware {
	return func(next Sender) Sender {
		return &hooksSender{next: next, hooks: hooks}
	}
}

type hooksSender struct {
	next  Sender
	hooks SendHooks
}

func (s *hooksSender) Send(ctx context.Context, e *Email) error {
	if s.hooks.OnSend != nil {
		s.hooks.OnSend(ctx, e)
	}

	start := time.Now()
	err := s.next.Send(ctx, e)
	duration := time.Since(start)

	if err != nil {
		if s.hooks.OnFailure != nil {
			s.hooks.OnFailure(ctx, e, duration, err)
		}
	} else {
		if s.hooks.OnSuccess != nil {
			s.hooks.OnSuccess(ctx, e, duration)
		}
	}
	return err
}

func (s *hooksSender) Close() error {
	return s.next.Close()
}
