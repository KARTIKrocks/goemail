import CodeBlock from '../components/CodeBlock';

export default function AsyncDocs() {
  return (
    <section id="async" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Async Sending</h2>
      <p className="text-text-muted mb-4">
        <code>AsyncSender</code> wraps any <code>Sender</code> with a buffered queue and a fixed pool of
        background workers. <code>Send</code> validates the email, drops it on the queue, and returns
        immediately — workers handle the SMTP round-trip and retry. It's the right tool when you don't
        want HTTP requests waiting on email delivery, or when you want to absorb traffic spikes without
        blocking.
      </p>

      <h3 id="async-creating" className="text-lg font-semibold text-text-heading mt-8 mb-2">Creating an AsyncSender</h3>
      <CodeBlock code={`sender, _ := email.NewSMTPSender(config)

async := email.NewAsyncSender(sender,
    email.WithQueueSize(200),
    email.WithWorkers(3),
    email.WithErrorHandler(func(ctx context.Context, e *email.Email, err error) {
        log.Printf("send to %v failed: %v", e.To, err)
    }),
)
defer async.Close()`} />

      <p className="text-text-muted mb-3">
        <code>AsyncSender</code> implements <code>Sender</code>, so it composes with middleware and
        with <code>Mailer</code>:
      </p>
      <CodeBlock code={`wrapped := email.Chain(async, email.WithLogging(logger))
mailer := email.NewMailer(wrapped, "no-reply@example.com")`} />

      <h3 id="async-options" className="text-lg font-semibold text-text-heading mt-8 mb-2">Options</h3>
      <div className="overflow-x-auto my-4">
        <table className="min-w-full text-sm border border-border rounded-lg">
          <thead className="bg-bg-card text-text-heading">
            <tr>
              <th className="text-left px-3 py-2 border-b border-border">Option</th>
              <th className="text-left px-3 py-2 border-b border-border">Default</th>
              <th className="text-left px-3 py-2 border-b border-border">Description</th>
            </tr>
          </thead>
          <tbody className="text-text-muted">
            <tr><td className="px-3 py-2 border-b border-border"><code>WithQueueSize(n)</code></td><td className="px-3 py-2 border-b border-border">100</td><td className="px-3 py-2 border-b border-border">Buffer capacity. <code>Send</code> blocks (or fails the context) when full.</td></tr>
            <tr><td className="px-3 py-2 border-b border-border"><code>WithWorkers(n)</code></td><td className="px-3 py-2 border-b border-border">1</td><td className="px-3 py-2 border-b border-border">Number of goroutines pulling from the queue.</td></tr>
            <tr><td className="px-3 py-2 border-b border-border"><code>WithAsyncLogger(l)</code></td><td className="px-3 py-2 border-b border-border">no-op</td><td className="px-3 py-2 border-b border-border">Logger for queue lifecycle events.</td></tr>
            <tr><td className="px-3 py-2"><code>WithErrorHandler(fn)</code></td><td className="px-3 py-2">no-op</td><td className="px-3 py-2">Called for every send that fails after retries.</td></tr>
          </tbody>
        </table>
      </div>

      <h3 id="async-sendwait" className="text-lg font-semibold text-text-heading mt-8 mb-2">Send vs SendWait</h3>
      <p className="text-text-muted mb-3">
        <code>Send</code> is fire-and-forget — it returns once the email is queued. Use it for the
        common case where the response should not depend on email delivery (signups, notifications).
      </p>
      <p className="text-text-muted mb-3">
        <code>SendWait</code> queues the email and blocks until a worker has finished sending it. Use
        it when you need the result inline — for example in a CLI tool or a synchronous test.
      </p>
      <CodeBlock code={`// Fire and forget — returns once queued.
if err := async.Send(ctx, msg); err != nil {
    return err // queue full or context canceled
}

// Block until delivered.
if err := async.SendWait(ctx, msg); err != nil {
    return err // delivery error
}`} />

      <h3 id="async-close" className="text-lg font-semibold text-text-heading mt-8 mb-2">Close</h3>
      <p className="text-text-muted mb-3">
        <code>Close</code> drains the queue, waits for in-flight workers to finish, then closes the
        underlying sender. Always defer it during shutdown so queued emails are not lost.
      </p>
      <CodeBlock code={`async := email.NewAsyncSender(sender, email.WithWorkers(3))
defer async.Close() // drain → wait → close inner sender`} />
    </section>
  );
}
