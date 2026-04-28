import CodeBlock from '../components/CodeBlock';

export default function DKIMDocs() {
  return (
    <section id="dkim" className="py-10 border-b border-border">
      <h2 className="text-2xl font-bold text-text-heading mb-4">DKIM Signing</h2>
      <p className="text-text-muted mb-4">
        DKIM (DomainKeys Identified Mail) lets receiving servers verify that an email actually came
        from your domain. Signed mail is far less likely to be marked spam, and modern providers
        (Gmail, Yahoo) require DKIM for bulk senders. goemail signs per RFC 6376 (RSA-SHA256) and RFC
        8463 (Ed25519-SHA256) with no external dependencies.
      </p>

      <h3 id="dkim-config" className="text-lg font-semibold text-text-heading mt-8 mb-2">DKIMConfig</h3>
      <CodeBlock code={`type DKIMConfig struct {
    // Required
    Domain     string         // e.g. "example.com"
    Selector   string         // DNS label, e.g. "mail" → mail._domainkey.example.com
    PrivateKey crypto.Signer  // RSA or Ed25519 private key

    // Optional
    HeaderCanonicalization Canonicalization // default: relaxed
    BodyCanonicalization   Canonicalization // default: relaxed
    Expiration             time.Duration    // signature lifetime
    SignedHeaders          []string         // override the default header set
}`} />
      <p className="text-text-muted mb-3 text-sm">
        Publish the public key as a TXT record at <code>{`{selector}._domainkey.{domain}`}</code> before
        sending — see your provider's docs for the exact format.
      </p>

      <h3 id="dkim-keys" className="text-lg font-semibold text-text-heading mt-8 mb-2">Loading Keys</h3>
      <p className="text-text-muted mb-3">
        <code>ParseDKIMPrivateKey</code> accepts PEM-encoded RSA or Ed25519 private keys, returning a
        <code> crypto.Signer</code> ready to plug into <code>DKIMConfig</code>.
      </p>
      <CodeBlock code={`pemData, err := os.ReadFile("dkim-private.pem")
if err != nil {
    log.Fatal(err)
}

privateKey, err := email.ParseDKIMPrivateKey(pemData)
if err != nil {
    log.Fatal(err)
}`} />

      <h3 id="dkim-smtp" className="text-lg font-semibold text-text-heading mt-8 mb-2">With SMTP</h3>
      <p className="text-text-muted mb-3">
        Set <code>DKIM</code> on <code>SMTPConfig</code>. Every message sent through that sender is
        automatically signed.
      </p>
      <CodeBlock code={`config := email.SMTPConfig{
    Host:     "smtp.example.com",
    Port:     587,
    Username: "user@example.com",
    Password: "password",
    UseTLS:   true,
    DKIM: &email.DKIMConfig{
        Domain:     "example.com",
        Selector:   "mail",
        PrivateKey: privateKey,
        Expiration: 24 * time.Hour,
    },
}

sender, _ := email.NewSMTPSender(config)`} />

      <h3 id="dkim-raw" className="text-lg font-semibold text-text-heading mt-8 mb-2">Raw Signing</h3>
      <p className="text-text-muted mb-3">
        For provider adapters or custom senders that build raw messages, use
        <code> BuildRawMessageWithDKIM</code> or <code>SignMessage</code> directly.
      </p>
      <CodeBlock code={`// Build a fully-formed RFC 5322 message with a DKIM-Signature header
msg, err := email.BuildRawMessageWithDKIM(e, dkimConfig)

// Or sign an already-built raw message
signed, err := email.SignMessage(rawMessage, dkimConfig)`} />
    </section>
  );
}
