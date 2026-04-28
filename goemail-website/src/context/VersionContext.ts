import { createContext } from 'react';

export interface Release {
  tag_name: string;
  html_url: string;
  published_at: string;
  prerelease: boolean;
}

export interface VersionContextValue {
  releases: Release[];
  selectedVersion: string;
  latestVersion: string;
  setSelectedVersion: (v: string) => void;
  loading: boolean;
  isLatest: boolean;
  getReleaseUrl: (version: string) => string;
  getInstallCmd: (version: string) => string;
}

export const VersionContext = createContext<VersionContextValue | null>(null);
