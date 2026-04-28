import { useEffect, useRef, useState, useMemo } from 'react';
import { useVersion } from '../hooks/useVersion';

interface SidebarProps {
  open: boolean;
  onClose: () => void;
}

interface ChildItem {
  id: string;
  label: string;
  minVersion?: string;
}

interface SectionItem {
  id: string;
  label: string;
  minVersion?: string;
  children?: ChildItem[];
}

const sections: SectionItem[] = [
  { id: 'top', label: 'Overview' },
  { id: 'getting-started', label: 'Getting Started' },
  {
    id: 'mailer',
    label: 'Mailer',
    children: [
      { id: 'mailer-creating', label: 'Creating a Mailer' },
      { id: 'mailer-send', label: 'Send & SendHTML' },
      { id: 'mailer-template', label: 'SendTemplate' },
      { id: 'mailer-batch', label: 'Batch Sending' },
      { id: 'mailer-close', label: 'Close' },
    ],
  },
  {
    id: 'email-builder',
    label: 'Email Builder',
    children: [
      { id: 'builder-fields', label: 'Fields & Recipients' },
      { id: 'builder-bodies', label: 'Plain Text & HTML' },
      { id: 'builder-attachments', label: 'Attachments' },
      { id: 'builder-headers', label: 'Custom Headers' },
      { id: 'builder-validation', label: 'Build & Validate' },
    ],
  },
  {
    id: 'smtp',
    label: 'SMTP',
    children: [
      { id: 'smtp-config', label: 'SMTPConfig' },
      { id: 'smtp-providers', label: 'Common Providers' },
      { id: 'smtp-pool', label: 'Connection Pooling' },
    ],
  },
  {
    id: 'templates',
    label: 'Templates',
    children: [
      { id: 'templates-creating', label: 'Creating Templates' },
      { id: 'templates-files', label: 'Loading from Files' },
      { id: 'templates-rendering', label: 'Rendering' },
    ],
  },
  {
    id: 'middleware',
    label: 'Middleware',
    children: [
      { id: 'middleware-chain', label: 'Chain' },
      { id: 'middleware-builtin', label: 'Built-in Middleware' },
      { id: 'middleware-hooks', label: 'Send Hooks' },
      { id: 'middleware-custom', label: 'Custom Middleware' },
    ],
  },
  {
    id: 'async',
    label: 'Async Sending',
    children: [
      { id: 'async-creating', label: 'Creating an AsyncSender' },
      { id: 'async-options', label: 'Options' },
      { id: 'async-sendwait', label: 'Send vs SendWait' },
      { id: 'async-close', label: 'Close' },
    ],
  },
  {
    id: 'providers',
    label: 'Provider Adapters',
    children: [
      { id: 'providers-sendgrid', label: 'SendGrid' },
      { id: 'providers-mailgun', label: 'Mailgun' },
      { id: 'providers-ses', label: 'AWS SES' },
      { id: 'providers-otel', label: 'OpenTelemetry' },
    ],
  },
  {
    id: 'dkim',
    label: 'DKIM Signing',
    children: [
      { id: 'dkim-config', label: 'DKIMConfig' },
      { id: 'dkim-keys', label: 'Loading Keys' },
      { id: 'dkim-smtp', label: 'With SMTP' },
      { id: 'dkim-raw', label: 'Raw Signing' },
    ],
  },
  {
    id: 'webhooks',
    label: 'Webhooks',
    children: [
      { id: 'webhooks-events', label: 'Event Types' },
      { id: 'webhooks-receiver', label: 'WebhookReceiver' },
      { id: 'webhooks-parsers', label: 'Provider Parsers' },
    ],
  },
  {
    id: 'reliability',
    label: 'Retry & Rate Limit',
    children: [
      { id: 'reliability-retry', label: 'Retry Logic' },
      { id: 'reliability-rate', label: 'Rate Limiting' },
      { id: 'reliability-context', label: 'Context Timeouts' },
    ],
  },
  {
    id: 'metrics',
    label: 'Metrics',
    children: [
      { id: 'metrics-interface', label: 'MetricsCollector' },
      { id: 'metrics-prometheus', label: 'Prometheus Example' },
    ],
  },
  {
    id: 'logging',
    label: 'Logging',
    children: [
      { id: 'logging-interface', label: 'Logger Interface' },
      { id: 'logging-slog', label: 'slog' },
      { id: 'logging-custom', label: 'Custom Loggers' },
    ],
  },
  {
    id: 'testing',
    label: 'Testing',
    children: [
      { id: 'testing-mock', label: 'MockSender' },
      { id: 'testing-assertions', label: 'Inspecting Sent Mail' },
    ],
  },
  {
    id: 'security',
    label: 'Security',
    children: [
      { id: 'security-env', label: 'Environment Variables' },
      { id: 'security-injection', label: 'Header Injection' },
      { id: 'security-app-passwords', label: 'App Passwords' },
    ],
  },
];

function updateHash(id: string) {
  const url = new URL(window.location.href);
  if (id === 'top') {
    url.hash = '';
  } else {
    url.hash = id;
  }
  if (window.location.hash !== url.hash) {
    history.replaceState(null, '', url.toString());
  }
}

export default function Sidebar({ open, onClose }: SidebarProps) {
  const { minVersion } = useVersion();

  const filteredSections = useMemo(() => {
    return sections
      .filter((s) => !s.minVersion || minVersion(s.minVersion))
      .map((s) => {
        if (!s.children) return s;
        const filteredChildren = s.children.filter(
          (c) => !c.minVersion || minVersion(c.minVersion),
        );
        return { ...s, children: filteredChildren.length > 0 ? filteredChildren : undefined };
      });
  }, [minVersion]);

  const allIds = useMemo(
    () =>
      filteredSections.flatMap((s) =>
        s.children ? [s.id, ...s.children.map((c) => c.id)] : [s.id],
      ),
    [filteredSections],
  );

  const parentMap = useMemo(() => {
    const map = new Map<string, string>();
    for (const s of filteredSections) {
      if (s.children) {
        for (const c of s.children) {
          map.set(c.id, s.id);
        }
      }
    }
    return map;
  }, [filteredSections]);

  const [active, setActive] = useState(() => {
    const hash = window.location.hash.slice(1);
    return hash && document.getElementById(hash) ? hash : 'top';
  });
  const [expanded, setExpanded] = useState<string | null>(() => {
    const hash = window.location.hash.slice(1);
    const parent = parentMap.get(hash);
    if (parent) return parent;
    const section = filteredSections.find((s) => s.id === hash);
    return section?.children ? hash : null;
  });

  const visibleSet = useRef(new Set<string>());
  const isScrollingTo = useRef<string | null>(null);
  const scrollTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    const hash = window.location.hash.slice(1);
    if (hash) {
      setTimeout(() => {
        document.getElementById(hash)?.scrollIntoView({ behavior: 'smooth' });
      }, 100);
    }
  }, []);

  const setActiveAndHash = (id: string) => {
    setActive(id);
    updateHash(id);

    const parent = parentMap.get(id);
    if (parent) {
      setExpanded(parent);
    } else {
      const section = filteredSections.find((s) => s.id === id);
      if (section?.children) {
        setExpanded(id);
      }
    }
  };

  useEffect(() => {
    const updateActive = () => {
      if (isScrollingTo.current) return;
      if (visibleSet.current.size === 0) return;

      const navbarOffset = 80;
      let bestId: string | null = null;
      let bestDistance = Infinity;

      for (const id of visibleSet.current) {
        const el = document.getElementById(id);
        if (!el) continue;
        const rect = el.getBoundingClientRect();
        const distance = Math.abs(rect.top - navbarOffset);
        if (distance < bestDistance) {
          bestDistance = distance;
          bestId = id;
        }
      }

      if (bestId) {
        setActiveAndHash(bestId);
      }
    };

    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) {
            visibleSet.current.add(entry.target.id);
          } else {
            visibleSet.current.delete(entry.target.id);
          }
        }
        updateActive();
      },
      { rootMargin: '-80px 0px -40% 0px', threshold: 0.1 }
    );

    allIds.forEach((id) => {
      const el = document.getElementById(id);
      if (el) observer.observe(el);
    });

    const onScroll = () => {
      if (!isScrollingTo.current) return;
      if (scrollTimer.current) clearTimeout(scrollTimer.current);
      scrollTimer.current = setTimeout(() => {
        isScrollingTo.current = null;
        updateActive();
      }, 150);
    };

    window.addEventListener('scroll', onScroll, { passive: true });

    return () => {
      observer.disconnect();
      window.removeEventListener('scroll', onScroll);
      if (scrollTimer.current) clearTimeout(scrollTimer.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [allIds]);

  const handleClick = (id: string) => {
    isScrollingTo.current = id;
    setActiveAndHash(id);
    document.getElementById(id)?.scrollIntoView({ behavior: 'smooth' });
    onClose();
  };

  const toggleExpand = (id: string) => {
    setExpanded((prev) => (prev === id ? null : id));
  };

  const isActive = (id: string) => active === id;

  const isSectionActive = (section: SectionItem) =>
    active === section.id ||
    (section.children?.some((c) => c.id === active) ?? false);

  const sectionClass = (section: SectionItem) =>
    `flex items-center justify-between w-full px-3 py-1.5 rounded-md text-sm transition-colors cursor-pointer ${
      isSectionActive(section)
        ? 'text-primary font-medium'
        : 'text-text-muted hover:text-text hover:bg-bg-card'
    }`;

  const subItemClass = (id: string) =>
    `block w-full text-left pl-6 pr-3 py-1 rounded-md text-xs transition-colors cursor-pointer ${
      isActive(id)
        ? 'bg-primary/10 text-primary font-medium'
        : 'text-text-muted hover:text-text hover:bg-bg-card'
    }`;

  return (
    <>
      {open && (
        <div
          className="fixed inset-0 bg-black/50 z-30 md:hidden"
          onClick={onClose}
        />
      )}

      <aside
        className={`fixed top-16 left-0 bottom-0 w-64 bg-bg-sidebar border-r border-border overflow-y-auto z-40 transition-transform ${
          open ? 'translate-x-0' : '-translate-x-full'
        } md:translate-x-0`}
      >
        <nav className="p-4 space-y-0.5">
          {filteredSections.map((section) =>
            section.children ? (
              <div key={section.id}>
                <button
                  onClick={() => {
                    if (expanded === section.id) {
                      toggleExpand(section.id);
                    } else {
                      handleClick(section.id);
                    }
                  }}
                  className={sectionClass(section)}
                >
                  <span>{section.label}</span>
                  <svg
                    className={`w-3.5 h-3.5 transition-transform ${
                      expanded === section.id ? 'rotate-90' : ''
                    }`}
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M9 5l7 7-7 7"
                    />
                  </svg>
                </button>
                {expanded === section.id && (
                  <div className="mt-0.5 space-y-0.5">
                    {section.children.map((child) => (
                      <button
                        key={child.id}
                        onClick={() => handleClick(child.id)}
                        className={subItemClass(child.id)}
                      >
                        {child.label}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <button
                key={section.id}
                onClick={() => handleClick(section.id)}
                className={`block w-full text-left px-3 py-1.5 rounded-md text-sm transition-colors cursor-pointer ${
                  isActive(section.id)
                    ? 'bg-primary/10 text-primary font-medium'
                    : 'text-text-muted hover:text-text hover:bg-bg-card'
                }`}
              >
                {section.label}
              </button>
            )
          )}
        </nav>
      </aside>
    </>
  );
}
