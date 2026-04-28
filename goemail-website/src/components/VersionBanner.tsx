import { useVersion } from '../hooks/useVersion';

export default function VersionBanner() {
  const { isLatest, selectedVersion, latestVersion, setSelectedVersion } = useVersion();

  if (isLatest) return null;

  return (
    <div className="bg-amber-500/10 border border-amber-500/30 text-amber-800 dark:text-amber-200 px-4 py-3 text-sm flex items-center justify-between gap-3 rounded-lg mb-6">
      <div className="flex items-center gap-2">
        <svg className="w-4 h-4 text-amber-600 dark:text-amber-400 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
        <span>
          You are viewing docs for <strong>{selectedVersion}</strong>. The latest version is <strong>{latestVersion}</strong>.
        </span>
      </div>
      <button
        onClick={() => {
          setSelectedVersion(latestVersion);
          window.scrollTo({ top: 0, behavior: 'smooth' });
        }}
        className="shrink-0 text-xs font-medium bg-amber-500/20 hover:bg-amber-500/30 text-amber-700 dark:text-amber-300 px-3 py-1 rounded-full transition-colors"
      >
        Switch to latest
      </button>
    </div>
  );
}
