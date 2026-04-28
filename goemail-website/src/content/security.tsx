import CodeBlock from '../components/CodeBlock';

export default function SecurityDocs() {
  return (
    <section id="security" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Security</h2>
      <p className="text-text-muted mb-4">
        A few small habits prevent the most common email-related security problems. goemail enforces
        the protocol-level ones automatically; the rest are operational.
      </p>

      <h3 id="security-env" className="text-lg font-semibold text-text-heading mt-8 mb-2">Environment Variables</h3>
      <p className="text-text-muted mb-3">
        Never check SMTP passwords or API keys into source control. Read them from the environment or
        a secret manager at startup.
      </p>
      <CodeBlock code={`config := email.SMTPConfig{
    Host:     os.Getenv("SMTP_HOST"),
    Port:     587,
    Username: os.Getenv("SMTP_USERNAME"),
    Password: os.Getenv("SMTP_PASSWORD"),
    From:     os.Getenv("SMTP_FROM"),
    UseTLS:   true,
}`} />

      <h3 id="security-injection" className="text-lg font-semibold text-text-heading mt-8 mb-2">Header Injection</h3>
      <p className="text-text-muted mb-3">
        If user input flows into a recipient list, subject, or custom header, an attacker who controls
        that input can try to inject extra headers (<code>Bcc:</code>, <code>Reply-To:</code>) by
        embedding CR/LF. goemail validates header names against RFC 5322 and strips CR/LF from header
        values automatically — <code>AddHeader</code> returns nothing for invalid input rather than
        silently corrupting the message.
      </p>
      <p className="text-text-muted mb-3">
        Recipient and subject validation happens in <code>Email.Build</code>:
      </p>
      <CodeBlock code={`built, err := email.NewEmail().
    SetFrom("noreply@example.com").
    AddTo(userSuppliedAddress). // validated below
    SetSubject(userSuppliedSubject).
    SetBody("...").
    Build()
if err != nil {
    // Reject the request — don't try to "clean up" the input
    return fmt.Errorf("invalid email: %w", err)
}`} />

      <h3 id="security-app-passwords" className="text-lg font-semibold text-text-heading mt-8 mb-2">App Passwords</h3>
      <p className="text-text-muted mb-3">
        Most consumer mail providers no longer accept account passwords for SMTP. Generate a
        scoped <strong>App Password</strong>:
      </p>
      <ul className="list-disc list-inside text-text-muted text-sm space-y-1 mb-3">
        <li><strong>Gmail:</strong> enable 2-Step Verification, then visit <a className="text-primary hover:underline" href="https://myaccount.google.com/apppasswords" target="_blank" rel="noopener noreferrer">myaccount.google.com/apppasswords</a>.</li>
        <li><strong>Yahoo / iCloud:</strong> generate from account security settings.</li>
        <li><strong>Microsoft 365:</strong> use OAuth 2.0 instead — basic SMTP auth is being phased out.</li>
      </ul>
      <p className="text-text-muted mb-3 text-sm">
        Treat App Passwords as production credentials: rotate them on suspected leak, scope them per
        environment, and store them only in your secret manager.
      </p>
    </section>
  );
}
