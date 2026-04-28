import CodeBlock from '../components/CodeBlock';

export default function MiddlewareDocs() {
  return (
    <section id="middleware" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Middleware</h2>
      <p className="text-text-muted mb-4">
        Middleware lets you weave cross-cutting concerns — logging, metrics, hooks, recovery, tracing
        — around any <code>Sender</code> without modifying it. Each middleware is a function with the
        signature <code>func(Sender) Sender</code>, so they compose cleanly.
      </p>

      <h3 id="middleware-chain" className="text-lg font-semibold text-text-heading mt-8 mb-2">Chain</h3>
      <p className="text-text-muted mb-3">
        <code>email.Chain</code> applies middleware in the order they appear — the first middleware is
        the outermost wrapper, executed first on the way in and last on the way out:
      </p>
      <CodeBlock code={`wrapped := email.Chain(sender,
    email.WithRecovery(),       // outermost: catches panics from anything below
    email.WithLogging(logger),  // logs send + duration
    email.WithMetrics(metrics), // increments counters
    email.WithHooks(hooks),     // innermost: closest to actual send
)

mailer := email.NewMailer(wrapped, "no-reply@example.com")`} />

      <h3 id="middleware-builtin" className="text-lg font-semibold text-text-heading mt-8 mb-2">Built-in Middleware</h3>
      <p className="text-text-muted mb-3">
        Four middleware ship with the core package. They cover the most common needs and are safe to
        stack in any order.
      </p>
      <div className="overflow-x-auto my-4">
        <table className="min-w-full text-sm border border-border rounded-lg">
          <thead className="bg-bg-card text-text-heading">
            <tr>
              <th className="text-left px-3 py-2 border-b border-border">Middleware</th>
              <th className="text-left px-3 py-2 border-b border-border">Behavior</th>
            </tr>
          </thead>
          <tbody className="text-text-muted">
            <tr><td className="px-3 py-2 border-b border-border"><code>WithLogging(Logger)</code></td><td className="px-3 py-2 border-b border-border">Logs send start, success, and failure with duration</td></tr>
            <tr><td className="px-3 py-2 border-b border-border"><code>WithRecovery()</code></td><td className="px-3 py-2 border-b border-border">Catches panics and returns <code>ErrPanicked</code></td></tr>
            <tr><td className="px-3 py-2 border-b border-border"><code>WithHooks(SendHooks)</code></td><td className="px-3 py-2 border-b border-border"><code>OnSend</code>, <code>OnSuccess</code>, <code>OnFailure</code> callbacks</td></tr>
            <tr><td className="px-3 py-2"><code>WithMetrics(MetricsCollector)</code></td><td className="px-3 py-2">Counters and duration histograms via the metrics interface</td></tr>
          </tbody>
        </table>
      </div>

      <h3 id="middleware-hooks" className="text-lg font-semibold text-text-heading mt-8 mb-2">Send Hooks</h3>
      <p className="text-text-muted mb-3">
        <code>WithHooks</code> is the easiest way to plug in audit logging, success counters, or alerting
        without writing a full middleware:
      </p>
      <CodeBlock code={`hooks := email.SendHooks{
    OnSend: func(ctx context.Context, e *email.Email) {
        log.Printf("sending to %v", e.To)
    },
    OnSuccess: func(ctx context.Context, e *email.Email, d time.Duration) {
        metrics.SendDurationMS.Observe(d.Seconds() * 1000)
    },
    OnFailure: func(ctx context.Context, e *email.Email, err error) {
        alerts.Notify("email failed", "to", e.To, "err", err)
    },
}

wrapped := email.Chain(sender, email.WithHooks(hooks))`} />

      <h3 id="middleware-custom" className="text-lg font-semibold text-text-heading mt-8 mb-2">Custom Middleware</h3>
      <p className="text-text-muted mb-3">
        Implement <code>email.Middleware</code> directly when you need custom behavior — distributed
        tracing, throttling per recipient, blacklist filtering, etc.
      </p>
      <CodeBlock code={`type blacklistSender struct {
    next email.Sender
    deny map[string]bool
}

func (s *blacklistSender) Send(ctx context.Context, e *email.Email) error {
    for _, to := range e.To {
        if s.deny[strings.ToLower(to)] {
            return fmt.Errorf("recipient %q is blacklisted", to)
        }
    }
    return s.next.Send(ctx, e)
}

func (s *blacklistSender) Close() error { return s.next.Close() }

func WithBlacklist(deny map[string]bool) email.Middleware {
    return func(next email.Sender) email.Sender {
        return &blacklistSender{next: next, deny: deny}
    }
}

wrapped := email.Chain(sender, WithBlacklist(myDenyList))`} />
    </section>
  );
}
