import CodeBlock from '../components/CodeBlock';

export default function ReliabilityDocs() {
  return (
    <section id="reliability" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Retry & Rate Limiting</h2>
      <p className="text-text-muted mb-4">
        SMTP is not always reliable: connections drop, providers throttle, and downstream mail
        servers temporarily refuse delivery. <code>SMTPSender</code> handles these cases for you with
        configurable retries and a built-in token-bucket rate limiter.
      </p>

      <h3 id="reliability-retry" className="text-lg font-semibold text-text-heading mt-8 mb-2">Retry Logic</h3>
      <p className="text-text-muted mb-3">
        Retries use exponential backoff. With the defaults (<code>MaxRetries=3</code>,
        <code> RetryDelay=1s</code>, <code>RetryBackoff=2.0</code>), failed sends are retried after
        1s, 2s, and 4s before giving up.
      </p>
      <CodeBlock code={`config := email.SMTPConfig{
    Host:         "smtp.gmail.com",
    Port:         587,
    Username:     "you@gmail.com",
    Password:     "app-password",
    UseTLS:       true,
    MaxRetries:   5,                // up to 5 retries
    RetryDelay:   500 * time.Millisecond,
    RetryBackoff: 2.0,              // 0.5s, 1s, 2s, 4s, 8s
}`} />
      <p className="text-text-muted mb-3 text-sm">
        Only transient failures (network errors, 4xx SMTP responses) are retried. Permanent failures
        — invalid recipient, authentication failure, message rejected — fail immediately.
      </p>

      <h3 id="reliability-rate" className="text-lg font-semibold text-text-heading mt-8 mb-2">Rate Limiting</h3>
      <p className="text-text-muted mb-3">
        Most SMTP providers cap how many emails per second they accept from a single sender. Set
        <code> RateLimit</code> in messages-per-second; <code>SMTPSender</code> blocks each <code>Send</code>
        until a token is available.
      </p>
      <CodeBlock code={`config := email.SMTPConfig{
    Host:      "smtp.gmail.com",
    RateLimit: 5, // up to 5 messages/second
}`} />
      <p className="text-text-muted mb-3 text-sm">
        The default is <code>10/s</code>. A value of <code>0</code> disables rate limiting entirely.
      </p>

      <h3 id="reliability-context" className="text-lg font-semibold text-text-heading mt-8 mb-2">Context Timeouts</h3>
      <p className="text-text-muted mb-3">
        Every <code>Send</code> takes a <code>context.Context</code>. Use it to bound how long an
        individual send may take, or to cancel work when the request that triggered the send is done.
      </p>
      <CodeBlock code={`ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

if err := mailer.Send(ctx, to, subject, body); err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("send timed out")
    }
    return err
}`} />
    </section>
  );
}
