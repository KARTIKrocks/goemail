import type { SidebarsConfig } from '@docusaurus/plugin-content-docs';

// This runs in Node.js - Don't use client-side code here (browser APIs, JSX...)

/**
 * Mirrors the original single-page site's navigation order: how to build a
 * message and send it (Core), how to compose cross-cutting behaviour around
 * a Sender (Composition), where mail actually goes (Delivery), and
 * operational concerns (Operations).
 */
const sidebars: SidebarsConfig = {
  docsSidebar: [
    'intro',
    'getting-started',
    {
      type: 'category',
      label: 'Core',
      collapsed: false,
      items: ['mailer', 'email-builder', 'smtp', 'templates'],
    },
    {
      type: 'category',
      label: 'Composition',
      collapsed: false,
      items: ['middleware', 'async'],
    },
    {
      type: 'category',
      label: 'Delivery',
      collapsed: false,
      items: ['providers', 'dkim', 'webhooks'],
    },
    {
      type: 'category',
      label: 'Operations',
      collapsed: false,
      items: ['reliability', 'metrics', 'logging'],
    },
    {
      type: 'category',
      label: 'Quality & Security',
      collapsed: false,
      items: ['testing', 'security'],
    },
  ],
};

export default sidebars;
