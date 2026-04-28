import CodeBlock from '../components/CodeBlock';

export default function ProvidersDocs() {
  return (
    <section id="providers" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Provider Adapters</h2>
      <p className="text-text-muted mb-4">
        Provider adapters send mail through an HTTP API instead of SMTP — usually faster, more
        reliable, and with richer delivery feedback. Each adapter lives in its own Go module under
        <code> providers/</code> so the core library stays dependency-free. Every adapter implements
        the standard <code>email.Sender</code> interface, so you can swap providers, wrap them in
        middleware, or feed them to <code>AsyncSender</code> with no other code changes.
      </p>

      <h3 id="providers-sendgrid" className="text-lg font-semibold text-text-heading mt-8 mb-2">SendGrid</h3>
      <CodeBlock lang="bash" code={`go get github.com/KARTIKrocks/goemail/providers/sendgrid`} />
      <CodeBlock code={`import (
    email "github.com/KARTIKrocks/goemail"
    "github.com/KARTIKrocks/goemail/providers/sendgrid"
)

sender, err := sendgrid.New(sendgrid.Config{
    APIKey: os.Getenv("SENDGRID_API_KEY"),
})
if err != nil {
    log.Fatal(err)
}

mailer := email.NewMailer(sender, "no-reply@example.com")`} />

      <h3 id="providers-mailgun" className="text-lg font-semibold text-text-heading mt-8 mb-2">Mailgun</h3>
      <CodeBlock lang="bash" code={`go get github.com/KARTIKrocks/goemail/providers/mailgun`} />
      <CodeBlock code={`import "github.com/KARTIKrocks/goemail/providers/mailgun"

sender, err := mailgun.New(mailgun.Config{
    Domain: "mg.example.com",
    APIKey: os.Getenv("MAILGUN_API_KEY"),
    // BaseURL: "https://api.eu.mailgun.net", // for EU accounts
})`} />

      <h3 id="providers-ses" className="text-lg font-semibold text-text-heading mt-8 mb-2">AWS SES</h3>
      <p className="text-text-muted mb-3">
        The SES adapter uses the AWS SDK v2 SES v2 client. AWS credentials are picked up from the
        ambient environment (env vars, EC2 instance role, IAM role for service accounts, etc.).
      </p>
      <CodeBlock lang="bash" code={`go get github.com/KARTIKrocks/goemail/providers/ses`} />
      <CodeBlock code={`import "github.com/KARTIKrocks/goemail/providers/ses"

sender, err := ses.New(context.Background(), ses.Config{
    Region: "us-east-1",
})`} />

      <h3 id="providers-otel" className="text-lg font-semibold text-text-heading mt-8 mb-2">OpenTelemetry</h3>
      <p className="text-text-muted mb-3">
        <code>otelmail</code> is a middleware, not a sender — it adds a distributed-tracing span around
        every send. Each span carries <code>email.from</code>, <code>email.to</code>,
        <code> email.subject</code>, and <code>email.recipients.count</code> attributes; failures
        record the error and set status to <code>Error</code>.
      </p>
      <CodeBlock lang="bash" code={`go get github.com/KARTIKrocks/goemail/providers/otelmail`} />
      <CodeBlock code={`import (
    email "github.com/KARTIKrocks/goemail"
    "github.com/KARTIKrocks/goemail/providers/otelmail"
)

wrapped := email.Chain(sender,
    otelmail.WithTracing(),       // creates a span per Send
    email.WithLogging(logger),
)

// Optional: customize the tracer or span name
// otelmail.WithTracing(otelmail.WithTracerName("myapp.email"))`} />
    </section>
  );
}
