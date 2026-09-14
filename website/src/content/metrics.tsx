import CodeBlock from '../components/CodeBlock';

export default function MetricsDocs() {
  return (
    <section id="metrics" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Metrics</h2>
      <p className="text-text-muted mb-4">
        Wrap any <code>Sender</code> with <code>WithMetrics</code> to record send attempts, successes,
        failures, and per-send latency. The interface is small and library-agnostic — implement it
        with Prometheus, OpenTelemetry, statsd, or whatever your stack uses.
      </p>

      <h3 id="metrics-interface" className="text-lg font-semibold text-text-heading mt-8 mb-2">MetricsCollector</h3>
      <CodeBlock code={`type MetricsCollector interface {
    IncSendAttempt()
    IncSendSuccess()
    IncSendFailure()
    ObserveSendDuration(d time.Duration)
}`} />

      <p className="text-text-muted mb-3">
        For testing or "metrics off in this environment" scenarios, use the no-op collector:
      </p>
      <CodeBlock code={`wrapped := email.Chain(sender, email.WithMetrics(email.NoOpMetricsCollector{}))`} />

      <h3 id="metrics-prometheus" className="text-lg font-semibold text-text-heading mt-8 mb-2">Prometheus Example</h3>
      <p className="text-text-muted mb-3">
        A typical Prometheus implementation — three counters and a histogram:
      </p>
      <CodeBlock code={`import (
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
wrapped := email.Chain(sender, email.WithMetrics(newPromMetrics()))`} />
    </section>
  );
}
