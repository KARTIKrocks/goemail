import CodeBlock from '../components/CodeBlock';

export default function WebhooksDocs() {
  return (
    <section id="webhooks" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Webhooks</h2>
      <p className="text-text-muted mb-4">
        Once an email leaves your process, the only way to know what happened — delivered, bounced,
        opened, marked as spam — is for the provider to call back. goemail provides a small
        provider-agnostic webhook layer: a normalized <code>WebhookEvent</code> type, a
        <code> WebhookReceiver</code> HTTP handler, and parser plugins per provider.
      </p>

      <h3 id="webhooks-events" className="text-lg font-semibold text-text-heading mt-8 mb-2">Event Types</h3>
      <p className="text-text-muted mb-3">
        Provider-specific payloads are normalized into a single <code>EventType</code> enum so your
        application code does not branch on provider:
      </p>
      <div className="overflow-x-auto my-4">
        <table className="min-w-full text-sm border border-border rounded-lg">
          <thead className="bg-bg-card text-text-heading">
            <tr>
              <th className="text-left px-3 py-2 border-b border-border">EventType</th>
              <th className="text-left px-3 py-2 border-b border-border">Meaning</th>
            </tr>
          </thead>
          <tbody className="text-text-muted">
            <tr><td className="px-3 py-2 border-b border-border"><code>EventDelivered</code></td><td className="px-3 py-2 border-b border-border">Accepted by recipient mail server</td></tr>
            <tr><td className="px-3 py-2 border-b border-border"><code>EventBounced</code></td><td className="px-3 py-2 border-b border-border">Hard bounce (permanent failure)</td></tr>
            <tr><td className="px-3 py-2 border-b border-border"><code>EventDeferred</code></td><td className="px-3 py-2 border-b border-border">Soft bounce (temporary failure)</td></tr>
            <tr><td className="px-3 py-2 border-b border-border"><code>EventOpened</code></td><td className="px-3 py-2 border-b border-border">Recipient opened the email</td></tr>
            <tr><td className="px-3 py-2 border-b border-border"><code>EventClicked</code></td><td className="px-3 py-2 border-b border-border">Recipient clicked a tracked link</td></tr>
            <tr><td className="px-3 py-2 border-b border-border"><code>EventComplained</code></td><td className="px-3 py-2 border-b border-border">Recipient marked the email as spam</td></tr>
            <tr><td className="px-3 py-2 border-b border-border"><code>EventUnsubscribed</code></td><td className="px-3 py-2 border-b border-border">Recipient unsubscribed</td></tr>
            <tr><td className="px-3 py-2"><code>EventDropped</code></td><td className="px-3 py-2">Provider rejected the message before sending</td></tr>
          </tbody>
        </table>
      </div>

      <h3 id="webhooks-receiver" className="text-lg font-semibold text-text-heading mt-8 mb-2">WebhookReceiver</h3>
      <p className="text-text-muted mb-3">
        <code>WebhookReceiver</code> is an <code>http.Handler</code>. Mount it on the route your
        provider posts to, and pass it a <code>WebhookHandler</code> that processes normalized events.
        Returning an error causes the receiver to respond <code>500</code> so the provider retries.
      </p>
      <CodeBlock code={`handler := email.WebhookHandlerFunc(func(ctx context.Context, ev email.WebhookEvent) error {
    switch ev.Type {
    case email.EventBounced:
        return suppressionList.Add(ev.Recipient, ev.Reason)
    case email.EventComplained:
        return spamComplaints.Record(ev.Recipient)
    case email.EventOpened, email.EventClicked:
        analytics.Track(ev)
    }
    return nil
})

receiver := email.NewWebhookReceiver(parser, handler,
    email.WithWebhookLogger(logger),
    email.WithEventFilter(
        email.EventBounced,
        email.EventComplained,
        email.EventDelivered,
    ),
)

http.Handle("/webhooks/email", receiver)`} />

      <h3 id="webhooks-parsers" className="text-lg font-semibold text-text-heading mt-8 mb-2">Provider Parsers</h3>
      <p className="text-text-muted mb-3">
        Each provider has its own webhook payload format and signature scheme. Parser submodules
        validate the signature and convert the payload into <code>[]WebhookEvent</code>. Pass the
        parser to <code>NewWebhookReceiver</code> as shown above.
      </p>
      <p className="text-text-muted mb-3 text-sm">
        Provider parser modules are released under <code>providers/webhook&lt;provider&gt;</code> as they
        become available — track <a className="text-primary hover:underline" href="https://github.com/KARTIKrocks/goemail" target="_blank" rel="noopener noreferrer">the repository</a> for status.
      </p>
    </section>
  );
}
