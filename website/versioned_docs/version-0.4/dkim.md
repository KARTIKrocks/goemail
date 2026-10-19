---
id: dkim
title: DKIM Signing
description: Sign outgoing mail with DKIM (RSA-SHA256 / Ed25519) with no external dependencies.
---

# DKIM Signing

DKIM (DomainKeys Identified Mail) lets receiving servers verify that an
email actually came from your domain. Signed mail is far less likely to
be marked spam, and modern providers (Gmail, Yahoo) require DKIM for bulk
senders. goemail signs per RFC 6376 (RSA-SHA256) and RFC 8463
(Ed25519-SHA256) with no external dependencies.

## DKIMConfig

```go
type DKIMConfig struct {
    // Required
    Domain     string         // e.g. "example.com"
    Selector   string         // DNS label, e.g. "mail" → mail._domainkey.example.com
    PrivateKey crypto.Signer  // RSA or Ed25519 private key

    // Optional
    HeaderCanonicalization Canonicalization // default: relaxed
    BodyCanonicalization   Canonicalization // default: relaxed
    Expiration             time.Duration    // signature lifetime
    SignedHeaders          []string         // override the default header set
}
```

Publish the public key as a TXT record at `{selector}._domainkey.{domain}`
before sending — see your provider's docs for the exact format.

## Loading Keys

`ParseDKIMPrivateKey` accepts PEM-encoded RSA or Ed25519 private keys,
returning a `crypto.Signer` ready to plug into `DKIMConfig`.

```go
pemData, err := os.ReadFile("dkim-private.pem")
if err != nil {
    log.Fatal(err)
}

privateKey, err := email.ParseDKIMPrivateKey(pemData)
if err != nil {
    log.Fatal(err)
}
```

## With SMTP

Set `DKIM` on `SMTPConfig`. Every message sent through that sender is
automatically signed.

```go
config := email.SMTPConfig{
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

sender, _ := email.NewSMTPSender(config)
```

## Raw Signing

For provider adapters or custom senders that build raw messages, use
`BuildRawMessageWithDKIM` or `SignMessage` directly.

```go
// Build a fully-formed RFC 5322 message with a DKIM-Signature header
msg, err := email.BuildRawMessageWithDKIM(e, dkimConfig)

// Or sign an already-built raw message
signed, err := email.SignMessage(rawMessage, dkimConfig)
```
