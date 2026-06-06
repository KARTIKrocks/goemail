package email

import (
	"context"
	"time"
)

// MetricsCollector defines the interface for collecting email send metrics.
// Implementations must be safe for concurrent use.
//
// This interface is intentionally minimal. Implement it to feed metrics
// into Prometheus, StatsD, OpenTelemetry, or any other system.
type MetricsCollector interface {
	// IncSendAttempt increments the counter for send attempts.
	IncSendAttempt()

	// IncSendSuccess increments the counter for successful sends.
	IncSendSuccess()

	// IncSendFailure increments the counter for failed sends.
	IncSendFailure()

	// ObserveSendDuration records the duration of a send operation.
	ObserveSendDuration(duration time.Duration)
}

// WithMetrics returns a Middleware that records send metrics using the
// provided MetricsCollector.
func WithMetrics(collector MetricsCollector) Middleware {
	return func(next Sender) Sender {
		return &metricsSender{next: next, collector: collector}
	}
}

type metricsSender struct {
	next      Sender
	collector MetricsCollector
}

func (s *metricsSender) Send(ctx context.Context, e *Email) error {
	s.collector.IncSendAttempt()

	start := time.Now()
	err := s.next.Send(ctx, e)
	duration := time.Since(start)

	s.collector.ObserveSendDuration(duration)

	if err != nil {
		s.collector.IncSendFailure()
	} else {
		s.collector.IncSendSuccess()
	}
	return err
}

func (s *metricsSender) Close() error {
	return s.next.Close()
}

// NoOpMetricsCollector is a metrics collector that discards all metrics.
// Useful as a default or in tests.
type NoOpMetricsCollector struct{}

func (NoOpMetricsCollector) IncSendAttempt()                     {}
func (NoOpMetricsCollector) IncSendSuccess()                     {}
func (NoOpMetricsCollector) IncSendFailure()                     {}
func (NoOpMetricsCollector) ObserveSendDuration(_ time.Duration) {}
