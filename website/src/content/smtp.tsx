import CodeBlock from '../components/CodeBlock';

export default function SMTPDocs() {
  return (
    <section id="smtp" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">SMTP</h2>
      <p className="text-text-muted mb-4">
        <code>NewSMTPSender</code> opens a connection to your SMTP server with TLS or STARTTLS,
        authenticates, and implements the <code>Sender</code> interface. All knobs — retries,
        timeouts, rate limits, pooling, DKIM — live in <code>SMTPConfig</code>.
      </p>

      <h3 id="smtp-config" className="text-lg font-semibold text-text-heading mt-8 mb-2">SMTPConfig</h3>
      <p className="text-text-muted mb-3">
        Only <code>Host</code>, <code>Port</code>, and credentials are required. Defaults are sensible for
        most providers; tune them for your throughput and reliability needs.
      </p>
      <CodeBlock code={`type SMTPConfig struct {
    // Required
    Host     string // SMTP server hostname
    Port     int    // 587 for STARTTLS, 465 for implicit TLS, 25 for plain
    Username string
    Password string
    From     string // Default sender for Mailer.Send

    // TLS
    UseTLS bool // STARTTLS upgrade after EHLO

    // Reliability
    Timeout      time.Duration // 30s
    MaxRetries   int           // 3
    RetryDelay   time.Duration // 1s
    RetryBackoff float64       // 2.0
    RateLimit    int           // emails/sec, default 10

    // Pooling (PoolSize > 0 enables it)
    PoolSize        int
    MaxIdleConns    int           // 2
    PoolMaxLifetime time.Duration // 30m
    PoolMaxIdleTime time.Duration // 5m
    MaxMessages     int           // per connection, default 100
    PoolWaitTimeout time.Duration // 5s

    // Optional integrations
    Logger Logger
    DKIM   *DKIMConfig
}`} />

      <h3 id="smtp-providers" className="text-lg font-semibold text-text-heading mt-8 mb-2">Common Providers</h3>
      <p className="text-text-muted mb-3">SMTP settings for the most common transactional providers:</p>

      <p className="text-text-muted text-sm font-medium mt-4 mb-1">Gmail</p>
      <p className="text-text-muted text-sm mb-2">
        Use an <a className="text-primary hover:underline" href="https://myaccount.google.com/apppasswords" target="_blank" rel="noopener noreferrer">App Password</a> with 2-Step Verification enabled.
      </p>
      <CodeBlock code={`email.SMTPConfig{
    Host:     "smtp.gmail.com",
    Port:     587,
    Username: "you@gmail.com",
    Password: "your-app-password",
    UseTLS:   true,
}`} />

      <p className="text-text-muted text-sm font-medium mt-4 mb-1">SendGrid</p>
      <CodeBlock code={`email.SMTPConfig{
    Host:     "smtp.sendgrid.net",
    Port:     587,
    Username: "apikey",
    Password: os.Getenv("SENDGRID_API_KEY"),
    UseTLS:   true,
}`} />

      <p className="text-text-muted text-sm font-medium mt-4 mb-1">AWS SES</p>
      <CodeBlock code={`email.SMTPConfig{
    Host:     "email-smtp.us-east-1.amazonaws.com",
    Port:     587,
    Username: os.Getenv("SES_SMTP_USERNAME"),
    Password: os.Getenv("SES_SMTP_PASSWORD"),
    UseTLS:   true,
}`} />

      <p className="text-text-muted text-sm font-medium mt-4 mb-1">Mailgun</p>
      <CodeBlock code={`email.SMTPConfig{
    Host:     "smtp.mailgun.org",
    Port:     587,
    Username: "postmaster@mg.example.com",
    Password: os.Getenv("MAILGUN_SMTP_PASSWORD"),
    UseTLS:   true,
}`} />

      <h3 id="smtp-pool" className="text-lg font-semibold text-text-heading mt-8 mb-2">Connection Pooling</h3>
      <p className="text-text-muted mb-3">
        For sustained throughput, set <code>PoolSize &gt; 0</code> to reuse SMTP connections. Without
        pooling, every send pays the TCP + TLS + AUTH handshake cost (often 200–500ms per message).
      </p>
      <CodeBlock code={`config := email.SMTPConfig{
    Host:            "smtp.gmail.com",
    Port:            587,
    Username:        "you@gmail.com",
    Password:        "app-password",
    UseTLS:          true,
    PoolSize:        5,                // up to 5 open connections
    MaxIdleConns:    2,                // keep 2 warm when idle
    PoolMaxLifetime: 30 * time.Minute, // recycle long-lived conns
    MaxMessages:     100,              // rotate after 100 messages
}

sender, err := email.NewSMTPSender(config)
if err != nil {
    log.Fatal(err)
}
defer sender.Close() // closes every pooled connection`} />
      <p className="text-text-muted mb-3 text-sm">
        <strong>Tip:</strong> always <code>defer sender.Close()</code> when pooling is enabled —
        otherwise idle connections leak until the process exits.
      </p>
    </section>
  );
}
