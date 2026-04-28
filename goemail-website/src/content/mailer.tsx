import CodeBlock from '../components/CodeBlock';

export default function MailerDocs() {
  return (
    <section id="mailer" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Mailer</h2>
      <p className="text-text-muted mb-4">
        <code>Mailer</code> is the high-level convenience type. It wraps a <code>Sender</code> with
        helpers for plain text, HTML, templates, and batch sending. The default <code>From</code>
        address you pass to <code>NewMailer</code> is used whenever an email does not specify its own
        sender.
      </p>

      <h3 id="mailer-creating" className="text-lg font-semibold text-text-heading mt-8 mb-2">Creating a Mailer</h3>
      <CodeBlock code={`sender, err := email.NewSMTPSender(config)
if err != nil {
    log.Fatal(err)
}

mailer := email.NewMailer(sender, "no-reply@example.com")
defer mailer.Close() // closes the underlying sender`} />

      <p className="text-text-muted mb-3">
        For senders wrapped in middleware, use <code>NewMailerWithOptions</code>:
      </p>
      <CodeBlock code={`mailer := email.NewMailerWithOptions(sender, "no-reply@example.com",
    email.WithMiddleware(
        email.WithRecovery(),
        email.WithLogging(logger),
    ),
)`} />

      <h3 id="mailer-send" className="text-lg font-semibold text-text-heading mt-8 mb-2">Send & SendHTML</h3>
      <p className="text-text-muted mb-3">
        For one-off messages, the <code>Send</code> and <code>SendHTML</code> helpers skip the builder
        and accept <code>(ctx, to, subject, body)</code> directly.
      </p>
      <CodeBlock code={`ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Plain text
err := mailer.Send(ctx,
    []string{"alice@example.com", "bob@example.com"},
    "Daily report",
    "Today's numbers are looking good.",
)

// HTML
err = mailer.SendHTML(ctx,
    []string{"alice@example.com"},
    "Welcome!",
    "<h1>Welcome to MyApp</h1><p>Click <a href=\\"https://example.com\\">here</a> to get started.</p>",
)`} />

      <h3 id="mailer-template" className="text-lg font-semibold text-text-heading mt-8 mb-2">SendTemplate</h3>
      <p className="text-text-muted mb-3">
        Render a registered template with arbitrary data. See the <a href="#templates" className="text-primary hover:underline">Templates</a> section for how to register them.
      </p>
      <CodeBlock code={`data := map[string]any{
    "Name":       "Alice",
    "VerifyLink": "https://example.com/verify/abc123",
}

err := mailer.SendTemplate(ctx,
    []string{"alice@example.com"},
    "welcome",
    data,
)`} />

      <h3 id="mailer-batch" className="text-lg font-semibold text-text-heading mt-8 mb-2">Batch Sending</h3>
      <p className="text-text-muted mb-3">
        <code>SendBatch</code> sends many emails concurrently with a bounded worker pool.
        Failures are aggregated and returned as a single joined error — successful sends still go through.
      </p>
      <CodeBlock code={`emails := []*email.Email{
    email.NewEmail().SetFrom("no-reply@example.com").AddTo("alice@example.com").
        SetSubject("Hi Alice").SetBody("Hello"),
    email.NewEmail().SetFrom("no-reply@example.com").AddTo("bob@example.com").
        SetSubject("Hi Bob").SetBody("Hello"),
    email.NewEmail().SetFrom("no-reply@example.com").AddTo("carol@example.com").
        SetSubject("Hi Carol").SetBody("Hello"),
}

// Concurrency limit of 5
if err := mailer.SendBatch(ctx, emails, 5); err != nil {
    log.Printf("some sends failed: %v", err)
}`} />

      <h3 id="mailer-close" className="text-lg font-semibold text-text-heading mt-8 mb-2">Close</h3>
      <p className="text-text-muted mb-3">
        Always <code>defer mailer.Close()</code>. It releases pooled SMTP connections, drains the async
        queue (if wrapping <code>AsyncSender</code>), and flushes any internal buffers.
      </p>
      <CodeBlock code={`mailer := email.NewMailer(sender, config.From)
defer mailer.Close()`} />
    </section>
  );
}
