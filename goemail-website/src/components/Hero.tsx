import { useState } from 'react';
import { useVersion } from '../hooks/useVersion';

interface Feature {
  title: string;
  desc: string;
}

const features: Feature[] = [
  { title: 'SMTP Support', desc: 'TLS / STARTTLS, authentication, and per-message context cancellation' },
  { title: 'Templating', desc: 'Go html/text templates for HTML and plain-text email bodies' },
  { title: 'Attachments', desc: 'Send files with proper MIME encoding and content types' },
  { title: 'Retry Logic', desc: 'Configurable exponential backoff for transient SMTP failures' },
  { title: 'Rate Limiting', desc: 'Built-in token-bucket limiter to avoid overwhelming providers' },
  { title: 'Connection Pooling', desc: 'Reuse SMTP connections for high-throughput sending' },
  { title: 'Async Sending', desc: 'Background queue with configurable workers and buffer size' },
  { title: 'Middleware Pipeline', desc: 'Composable logging, metrics, recovery, and lifecycle hooks' },
  { title: 'Provider Adapters', desc: 'SendGrid, Mailgun, and AWS SES via HTTP APIs (separate modules)' },
  { title: 'DKIM Signing', desc: 'RSA-SHA256 / Ed25519 signing per RFC 6376 + 8463 — zero extra deps' },
  { title: 'Webhooks', desc: 'Receive and normalize delivery events from any provider' },
  { title: 'Pluggable Logger', desc: 'Bring your own logger — slog, zap, logrus, zerolog, anything' },
  { title: 'Mock Sender', desc: 'Drop-in test double with assertions on sent messages' },
  { title: 'Header Safety', desc: 'Automatic header-injection protection and address validation' },
];

export default function Hero() {
  const [copied, setCopied] = useState(false);
  const { selectedVersion, getInstallCmd } = useVersion();
  const installCmd = getInstallCmd(selectedVersion);

  const handleCopy = () => {
    navigator.clipboard.writeText(installCmd);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <section id="top" className="py-16 border-b border-border">
      <h1 className="text-4xl md:text-5xl font-bold text-text-heading mb-4">
        Production-ready Go email package
      </h1>
      <p className="text-lg text-text-muted max-w-2xl mb-8">
        SMTP, templating, retries, rate limiting, connection pooling, async sending, middleware,
        DKIM signing, and provider adapters for SendGrid, Mailgun, and AWS SES — composable, well-tested,
        and dependency-light.
      </p>

      <div className="flex items-center gap-2 bg-bg-card border border-border rounded-lg px-4 py-3 max-w-lg mb-10">
        <span className="text-text-muted select-none">$</span>
        <code className="flex-1 text-sm font-mono text-accent">{installCmd}</code>
        <button
          onClick={handleCopy}
          className="text-xs text-text-muted hover:text-text px-2 py-1 rounded bg-overlay hover:bg-overlay-hover transition-colors"
        >
          {copied ? 'Copied!' : 'Copy'}
        </button>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        {features.map((f) => (
          <div key={f.title} className="bg-bg-card border border-border rounded-lg p-4">
            <h3 className="text-sm font-semibold text-text-heading mb-1">{f.title}</h3>
            <p className="text-xs text-text-muted">{f.desc}</p>
          </div>
        ))}
      </div>
    </section>
  );
}
