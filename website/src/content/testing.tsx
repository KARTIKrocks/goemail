import CodeBlock from '../components/CodeBlock';

export default function TestingDocs() {
  return (
    <section id="testing" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Testing</h2>
      <p className="text-text-muted mb-4">
        Testing email-sending code without a real SMTP server is one of the main reasons goemail keeps
        the <code>Sender</code> interface small. <code>NewMockSender</code> returns an in-memory
        sender that records every email it receives, with helpers to inspect and assert on them.
      </p>

      <h3 id="testing-mock" className="text-lg font-semibold text-text-heading mt-8 mb-2">MockSender</h3>
      <CodeBlock code={`func TestSendWelcomeEmail(t *testing.T) {
    mock := email.NewMockSender()
    mailer := email.NewMailer(mock, "test@example.com")

    err := mailer.Send(context.Background(),
        []string{"alice@example.com"},
        "Welcome",
        "Welcome to MyApp!",
    )
    if err != nil {
        t.Fatalf("send failed: %v", err)
    }

    if got := mock.GetEmailCount(); got != 1 {
        t.Fatalf("got %d emails, want 1", got)
    }
}`} />

      <p className="text-text-muted mb-3">
        For tests that need to simulate failure, configure the mock to return an error:
      </p>
      <CodeBlock code={`mock := email.NewMockSender()
mock.SetError(errors.New("simulated SMTP failure"))

err := mailer.Send(ctx, to, subject, body)
// err is non-nil — assert your retry / fallback logic`} />

      <h3 id="testing-assertions" className="text-lg font-semibold text-text-heading mt-8 mb-2">Inspecting Sent Mail</h3>
      <p className="text-text-muted mb-3">
        Use <code>GetLastEmail</code>, <code>GetEmails</code>, or <code>GetEmailCount</code> to make
        assertions about subject, recipients, body, attachments, or headers:
      </p>
      <CodeBlock code={`sent := mock.GetLastEmail()

if sent.Subject != "Welcome" {
    t.Errorf("Subject = %q, want %q", sent.Subject, "Welcome")
}

if len(sent.To) != 1 || sent.To[0] != "alice@example.com" {
    t.Errorf("To = %v, want [alice@example.com]", sent.To)
}

if !strings.Contains(sent.Body, "Welcome") {
    t.Errorf("Body missing greeting: %s", sent.Body)
}

// Reset between subtests
mock.Reset()`} />
    </section>
  );
}
