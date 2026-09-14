import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import type { ReactNode } from 'react';

import styles from './index.module.css';

type Feature = {
  readonly title: string;
  readonly description: string;
};

// Kept in sync with the feature list in the repository README.
const FEATURES = [
  {
    title: 'SMTP Support',
    description:
      'TLS / STARTTLS, authentication, and per-message context cancellation',
  },
  {
    title: 'Templating',
    description: 'Go html/text templates for HTML and plain-text email bodies',
  },
  {
    title: 'Attachments',
    description: 'Send files with proper MIME encoding and content types',
  },
  {
    title: 'Retry Logic',
    description: 'Configurable exponential backoff for transient SMTP failures',
  },
  {
    title: 'Rate Limiting',
    description:
      'Built-in token-bucket limiter to avoid overwhelming providers',
  },
  {
    title: 'Connection Pooling',
    description: 'Reuse SMTP connections for high-throughput sending',
  },
  {
    title: 'Async Sending',
    description: 'Background queue with configurable workers and buffer size',
  },
  {
    title: 'Middleware Pipeline',
    description: 'Composable logging, metrics, recovery, and lifecycle hooks',
  },
  {
    title: 'Provider Adapters',
    description:
      'SendGrid, Mailgun, and AWS SES via HTTP APIs (separate modules)',
  },
  {
    title: 'DKIM Signing',
    description:
      'RSA-SHA256 / Ed25519 signing per RFC 6376 + 8463 — zero extra deps',
  },
  {
    title: 'Webhooks',
    description: 'Receive and normalize delivery events from any provider',
  },
  {
    title: 'Pluggable Logger',
    description: 'Bring your own logger — slog, zap, logrus, zerolog, anything',
  },
  {
    title: 'Mock Sender',
    description: 'Drop-in test double with assertions on sent messages',
  },
  {
    title: 'Header Safety',
    description: 'Automatic header-injection protection and address validation',
  },
] as const satisfies readonly Feature[];

// Everything goemail provides that raw net/smtp leaves to you. Kept in sync
// with the "Why goemail?" table in the repository README.
const CAPABILITIES = [
  'Retry with exponential backoff',
  'Connection pooling for high throughput',
  'Rate limiting',
  'DKIM signing (RSA-SHA256 / Ed25519-SHA256)',
  'HTML templates, attachments, batch sending',
  'Provider HTTP adapters (SendGrid, Mailgun, AWS SES)',
  'Middleware (logging, metrics, hooks, recovery)',
  'Header injection protection, address validation',
  'Async sending with a buffered worker queue',
] as const satisfies readonly string[];

const INSTALL_COMMAND = 'go get github.com/KARTIKrocks/goemail';

function Hero(): ReactNode {
  return (
    <header className={styles.hero}>
      <div className="container">
        <h1 className={styles.title}>Production-ready Go email package</h1>
        <p className={styles.subtitle}>
          SMTP, templating, retries, rate limiting, connection pooling, async
          sending, middleware, DKIM signing, and provider adapters for SendGrid,
          Mailgun, and AWS SES — composable, well-tested, and dependency-light.
        </p>

        <div className={styles.buttons}>
          <Link
            className="button button--primary button--lg"
            to="/docs/getting-started">
            Get Started
          </Link>
          <Link
            className="button button--secondary button--lg"
            to="https://pkg.go.dev/github.com/KARTIKrocks/goemail">
            API Reference
          </Link>
        </div>

        <div className={styles.install}>
          <span className={styles.prompt} aria-hidden="true">
            $
          </span>
          <code>{INSTALL_COMMAND}</code>
        </div>
      </div>
    </header>
  );
}

function Features(): ReactNode {
  return (
    <section className="container" aria-label="Features">
      <div className={styles.features}>
        {FEATURES.map((feature) => (
          <article key={feature.title} className={styles.card}>
            <h2>{feature.title}</h2>
            <p>{feature.description}</p>
          </article>
        ))}
      </div>
    </section>
  );
}

function WhyGoemail(): ReactNode {
  return (
    <section className={styles.section}>
      <div className="container">
        <h2 className={styles.sectionTitle}>Why goemail?</h2>
        <p className={styles.sectionLead}>
          <code>net/smtp</code> gets you a connection and a <code>DATA</code>{' '}
          command. Everything past that — the parts that turn “I can send one
          email” into “I can run this in production” — is what goemail provides.
          It is not a replacement for <code>net/smtp</code>; the default{' '}
          <code>Sender</code> is built on top of it.
        </p>

        <div className={styles.tableScroll}>
          <table className={styles.compare}>
            <thead>
              <tr>
                <th scope="col">Capability</th>
                <th scope="col">goemail</th>
                <th scope="col">
                  Raw <code>net/smtp</code>
                </th>
              </tr>
            </thead>
            <tbody>
              {CAPABILITIES.map((capability) => (
                <tr key={capability}>
                  <th scope="row">{capability}</th>
                  <td>
                    <span className={styles.check} aria-hidden="true">
                      ✓
                    </span>
                    <span className={styles.srOnly}>Included</span>
                  </td>
                  <td className={styles.diy}>You build it</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
}

export default function Home(): ReactNode {
  const { siteConfig } = useDocusaurusContext();

  return (
    <Layout
      title={siteConfig.tagline}
      description="A production-ready email package for Go with SMTP, templating, retries, connection pooling, async sending, middleware, DKIM signing, and provider adapters for SendGrid, Mailgun, and AWS SES.">
      <Hero />
      <main>
        <Features />
        <WhyGoemail />
      </main>
    </Layout>
  );
}
