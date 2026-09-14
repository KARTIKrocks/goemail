import CodeBlock from '../components/CodeBlock';
import { useVersion } from '../hooks/useVersion';

export default function GettingStarted() {
  const { selectedVersion, getInstallCmd } = useVersion();

  return (
    <section id="getting-started" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">Getting Started</h2>

      <h3 className="text-lg font-semibold text-text-heading mt-6 mb-2">Installation</h3>
      <p className="text-text-muted mb-3">Requires <strong>Go 1.22+</strong>. The core module has no third-party dependencies beyond <code>golang.org/x/sync</code> and <code>golang.org/x/time</code>.</p>
      <CodeBlock lang="bash" code={getInstallCmd(selectedVersion)} />

      <h3 className="text-lg font-semibold text-text-heading mt-8 mb-2">Quick Start</h3>
      <p className="text-text-muted mb-3">
        A minimal program that sends a plain-text email over SMTP with a context timeout:
      </p>
      <CodeBlock code={`package main

import (
    "context"
    "log"
    "time"

    email "github.com/KARTIKrocks/goemail"
)

func main() {
    config := email.SMTPConfig{
        Host:     "smtp.gmail.com",
        Port:     587,
        Username: "your-email@gmail.com",
        Password: "your-app-password",
        From:     "your-email@gmail.com",
        UseTLS:   true,
    }

    sender, err := email.NewSMTPSender(config)
    if err != nil {
        log.Fatal(err)
    }
    mailer := email.NewMailer(sender, config.From)
    defer mailer.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    err = mailer.Send(ctx,
        []string{"recipient@example.com"},
        "Hello from goemail",
        "This is a test email.",
    )
    if err != nil {
        log.Fatal(err)
    }
}`} />

      <h3 className="text-lg font-semibold text-text-heading mt-8 mb-2">Two layers: Sender and Mailer</h3>
      <p className="text-text-muted mb-3">
        goemail is built around a single <code>Sender</code> interface. Concrete senders include
        <code> NewSMTPSender</code>, <code>NewMockSender</code>, and provider adapters
        (<code>sendgrid.New</code>, <code>mailgun.New</code>, <code>ses.New</code>).
        A <code>Mailer</code> wraps any sender with helpers for plain-text, HTML, templates, and batch sending.
      </p>
      <p className="text-text-muted mb-3">
        Middleware (logging, metrics, hooks, recovery) is applied to senders via <code>email.Chain</code>,
        so you can compose any sender with any middleware without touching the <code>Mailer</code>.
      </p>
    </section>
  );
}
