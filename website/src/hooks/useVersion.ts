import { useContext, useCallback } from 'react';
import { VersionContext, type VersionContextValue } from '../context/VersionContext';
import { isAtLeast } from '../lib/version';

export interface UseVersionReturn extends VersionContextValue {
  /** Returns true if selectedVersion >= the given version string. */
  minVersion: (v: string) => boolean;
}

export function useVersion(): UseVersionReturn {
  const ctx = useContext(VersionContext);
  if (!ctx) throw new Error('useVersion must be used within VersionProvider');

  const { selectedVersion } = ctx;
  const minVersion = useCallback(
    (v: string) => isAtLeast(selectedVersion, v),
    [selectedVersion],
  );

  return { ...ctx, minVersion };
}
