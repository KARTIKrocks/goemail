import { useState } from 'react';
import ThemeProvider from './components/ThemeProvider';
import VersionProvider from './components/VersionProvider';
import Navbar from './components/Navbar';
import Sidebar from './components/Sidebar';
import VersionBanner from './components/VersionBanner';
import Hero from './components/Hero';
import GettingStarted from './content/getting-started';
import MailerDocs from './content/mailer';
import EmailBuilderDocs from './content/email-builder';
import SMTPDocs from './content/smtp';
import TemplatesDocs from './content/templates';
import MiddlewareDocs from './content/middleware';
import AsyncDocs from './content/async';
import ProvidersDocs from './content/providers';
import DKIMDocs from './content/dkim';
import WebhooksDocs from './content/webhooks';
import ReliabilityDocs from './content/reliability';
import MetricsDocs from './content/metrics';
import LoggingDocs from './content/logging';
import TestingDocs from './content/testing';
import SecurityDocs from './content/security';

function DocsContent() {
  return (
    <>
      <Hero />
      <GettingStarted />
      <MailerDocs />
      <EmailBuilderDocs />
      <SMTPDocs />
      <TemplatesDocs />
      <MiddlewareDocs />
      <AsyncDocs />
      <ProvidersDocs />
      <DKIMDocs />
      <WebhooksDocs />
      <ReliabilityDocs />
      <MetricsDocs />
      <LoggingDocs />
      <TestingDocs />
      <SecurityDocs />
    </>
  );
}

export default function App() {
  const [menuOpen, setMenuOpen] = useState(false);

  return (
    <ThemeProvider>
    <VersionProvider>
      <div className="min-h-screen">
        <Navbar onMenuToggle={() => setMenuOpen((o) => !o)} menuOpen={menuOpen} />
        <Sidebar open={menuOpen} onClose={() => setMenuOpen(false)} />

        <main className="pt-16 md:pl-64">
          <div className="max-w-4xl mx-auto px-4 md:px-8 pb-20">
            <div className="pt-4">
              <VersionBanner />
            </div>
            <DocsContent />

            <footer className="py-10 text-center text-sm text-text-muted border-t border-border mt-10">
              <p>
                goemail is open source under the{' '}
                <a
                  href="https://github.com/KARTIKrocks/goemail/blob/main/LICENSE"
                  className="text-primary hover:underline"
                  target="_blank"
                  rel="noopener noreferrer"
                >
                  MIT License
                </a>
              </p>
            </footer>
          </div>
        </main>
      </div>
    </VersionProvider>
    </ThemeProvider>
  );
}
