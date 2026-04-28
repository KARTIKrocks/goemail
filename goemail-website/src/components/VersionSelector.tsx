import { useState, useEffect, useRef } from 'react';
import { useVersion } from '../hooks/useVersion';

export default function VersionSelector() {
  const { releases, selectedVersion, latestVersion, setSelectedVersion, loading, getReleaseUrl } = useVersion();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener('mousedown', handleClick);
    return () => document.removeEventListener('mousedown', handleClick);
  }, []);

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen((o) => !o)}
        className="flex items-center gap-1 text-xs font-medium bg-primary/20 text-primary px-2 py-0.5 rounded-full hover:bg-primary/30 transition-colors cursor-pointer"
      >
        {selectedVersion}
        <svg className={`w-3 h-3 transition-transform ${open ? 'rotate-180' : ''}`} fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
        </svg>
      </button>

      {open && (
        <div className="absolute top-full mt-2 left-0 w-72 bg-bg-card border border-border rounded-lg shadow-xl overflow-hidden z-50">
          <div className="px-3 py-2 border-b border-border">
            <p className="text-xs text-text-muted">Switch documentation version</p>
          </div>

          {loading && (
            <div className="px-3 py-4 text-xs text-text-muted text-center">Loading versions…</div>
          )}

          {!loading && releases.length === 0 && (
            <div className="px-3 py-3 text-xs text-text-muted text-center">
              Could not load versions.
              <a
                href="https://github.com/KARTIKrocks/goemail/releases"
                target="_blank"
                rel="noopener noreferrer"
                className="block mt-1 text-primary hover:underline"
              >
                View on GitHub
              </a>
            </div>
          )}

          <ul className="max-h-60 overflow-y-auto">
            {releases.map((r) => {
              const isSelected = r.tag_name === selectedVersion;
              const isLatest = r.tag_name === latestVersion;
              return (
                <li key={r.tag_name}>
                  <button
                    onClick={() => {
                      setSelectedVersion(r.tag_name);
                      setOpen(false);
                      window.scrollTo({ top: 0, behavior: 'smooth' });
                    }}
                    className={`w-full flex items-center justify-between px-3 py-2 hover:bg-overlay transition-colors text-left ${
                      isSelected ? 'bg-primary/10' : ''
                    }`}
                  >
                    <div className="flex items-center gap-2">
                      {isSelected && (
                        <svg className="w-3.5 h-3.5 text-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2.5} d="M5 13l4 4L19 7" />
                        </svg>
                      )}
                      <span className={`text-sm font-medium ${isSelected ? 'text-primary' : 'text-text-heading'}`}>
                        {r.tag_name}
                      </span>
                      {isLatest && (
                        <span className="text-[10px] font-medium bg-green-500/20 text-green-700 dark:text-green-400 px-1.5 py-0.5 rounded-full">
                          latest
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-xs text-text-muted">
                        {new Date(r.published_at).toLocaleDateString('en-US', {
                          month: 'short',
                          day: 'numeric',
                          year: 'numeric',
                        })}
                      </span>
                      <a
                        href={getReleaseUrl(r.tag_name)}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="text-text-muted hover:text-primary transition-colors"
                        title="View release notes"
                        onClick={(e) => e.stopPropagation()}
                      >
                        <svg className="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
                        </svg>
                      </a>
                    </div>
                  </button>
                </li>
              );
            })}
          </ul>

          <div className="px-3 py-2 border-t border-border">
            <a
              href="https://github.com/KARTIKrocks/goemail/releases"
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-primary hover:underline"
              onClick={() => setOpen(false)}
            >
              View all releases on GitHub →
            </a>
          </div>
        </div>
      )}
    </div>
  );
}
