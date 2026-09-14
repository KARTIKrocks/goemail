import CodeBlock from '../components/CodeBlock';

export default function EmailBuilderDocs() {
  return (
    <section id="email-builder" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Email Builder</h2>
      <p className="text-text-muted mb-4">
        For messages that need anything beyond the four-argument <code>Send</code> helper —
        multiple recipients, CC/BCC, custom headers, or attachments — use the chainable
        <code> Email</code> builder. Every setter returns the same <code>*Email</code> so calls
        can be fluently composed.
      </p>

      <h3 id="builder-fields" className="text-lg font-semibold text-text-heading mt-8 mb-2">Fields & Recipients</h3>
      <CodeBlock code={`msg := email.NewEmail().
    SetFrom("sender@example.com").
    AddTo("user1@example.com", "user2@example.com").
    AddCc("manager@example.com").
    AddBcc("archive@example.com").
    SetReplyTo("support@example.com").
    SetSubject("Important Update")`} />

      <p className="text-text-muted mb-3">
        Recipients can be passed as bare addresses (<code>"alice@example.com"</code>) or with display
        names (<code>"Alice &lt;alice@example.com&gt;"</code>). Each <code>AddTo</code>, <code>AddCc</code>,
        and <code>AddBcc</code> accepts variadic arguments and is additive.
      </p>

      <h3 id="builder-bodies" className="text-lg font-semibold text-text-heading mt-8 mb-2">Plain Text & HTML</h3>
      <p className="text-text-muted mb-3">
        Set either or both. When both are present goemail produces a multipart/alternative message and
        the recipient's mail client picks the best fit.
      </p>
      <CodeBlock code={`msg.
    SetBody("Plain-text fallback for clients without HTML support.").
    SetHTMLBody("<h1>Hello</h1><p>This is the rich version.</p>")`} />

      <h3 id="builder-attachments" className="text-lg font-semibold text-text-heading mt-8 mb-2">Attachments</h3>
      <CodeBlock code={`pdfData, err := os.ReadFile("invoice.pdf")
if err != nil {
    log.Fatal(err)
}

msg := email.NewEmail().
    SetFrom("billing@example.com").
    AddTo("customer@example.com").
    SetSubject("Your invoice").
    SetBody("Please find your invoice attached.").
    AddAttachment("invoice.pdf", "application/pdf", pdfData)`} />

      <p className="text-text-muted mb-3">
        Use the correct MIME type for each attachment (<code>image/png</code>, <code>text/csv</code>,
        <code> application/pdf</code>, etc.) — clients use it to render previews and pick icons.
      </p>

      <h3 id="builder-headers" className="text-lg font-semibold text-text-heading mt-8 mb-2">Custom Headers</h3>
      <p className="text-text-muted mb-3">
        Add any RFC 5322 header. goemail validates header names and strips CR/LF to prevent header
        injection from untrusted input.
      </p>
      <CodeBlock code={`msg.
    AddHeader("X-Priority", "1").
    AddHeader("X-Campaign-ID", campaignID).
    AddHeader("List-Unsubscribe", "<https://example.com/unsubscribe?u=123>")`} />

      <h3 id="builder-validation" className="text-lg font-semibold text-text-heading mt-8 mb-2">Build & Validate</h3>
      <p className="text-text-muted mb-3">
        Call <code>Build</code> to validate the email before sending. <code>Build</code> checks that
        the sender, at least one recipient, and a subject are present, and that all addresses parse.
        <code> Mailer.SendEmail</code> calls <code>Build</code> for you.
      </p>
      <CodeBlock code={`built, err := msg.Build()
if err != nil {
    return fmt.Errorf("invalid email: %w", err)
}

if err := mailer.SendEmail(ctx, built); err != nil {
    return err
}`} />
    </section>
  );
}
