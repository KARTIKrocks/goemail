---
id: metrics
title: Metrics
description: Record send attempts, successes, failures, and latency with a small, library-agnostic MetricsCollector interface.
---

# Metrics

Wrap any `Sender` with `WithMetrics` to record send attempts, successes,
failures, and per-send latency. The interface is small and
library-agnostic — implement it with Prometheus, OpenTelemetry, statsd,
or whatever your stack uses.

## MetricsCollector

```go
type MetricsCollector interface {
    IncSendAttempt()
    IncSendSuccess()
    IncSendFailure()
    ObserveSendDuration(d time.Duration)
}
```

For testing or "metrics off in this environment" scenarios, use the
no-op collector:

```go
wrapped := email.Chain(sender, email.WithMetrics(email.NoOpMetricsCollector{}))
```

## Prometheus Example

A typical Prometheus implementation — three counters and a histogram:

```go
import (
    "time"

    email "github.com/KARTIKrocks/goemail"
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

type promMetrics struct {
    attempts prometheus.Counter
    success  prometheus.Counter
    failure  prometheus.Counter
    duration prometheus.Histogram
}

func newPromMetrics() *promMetrics {
    return &promMetrics{
        attempts: promauto.NewCounter(prometheus.CounterOpts{
            Name: "email_send_attempts_total",
        }),
        success: promauto.NewCounter(prometheus.CounterOpts{
            Name: "email_send_success_total",
        }),
        failure: promauto.NewCounter(prometheus.CounterOpts{
            Name: "email_send_failure_total",
        }),
        duration: promauto.NewHistogram(prometheus.HistogramOpts{
            Name:    "email_send_duration_seconds",
            Buckets: prometheus.DefBuckets,
        }),
    }
}

func (m *promMetrics) IncSendAttempt()                          { m.attempts.Inc() }
func (m *promMetrics) IncSendSuccess()                          { m.success.Inc() }
func (m *promMetrics) IncSendFailure()                          { m.failure.Inc() }
func (m *promMetrics) ObserveSendDuration(d time.Duration)      { m.duration.Observe(d.Seconds()) }

// Wire it up
wrapped := email.Chain(sender, email.WithMetrics(newPromMetrics()))
```
