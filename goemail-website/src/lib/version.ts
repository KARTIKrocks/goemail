/**
 * Compare two semver version strings (e.g., "v0.1.0", "v1.0.0").
 * Returns -1 if a < b, 0 if equal, 1 if a > b.
 */
export function compareVersions(a: string, b: string): number {
  const parse = (v: string) =>
    v.replace(/^v/, '').split('.').map(Number);
  const pa = parse(a);
  const pb = parse(b);
  for (let i = 0; i < 3; i++) {
    const va = pa[i] ?? 0;
    const vb = pb[i] ?? 0;
    if (va < vb) return -1;
    if (va > vb) return 1;
  }
  return 0;
}

/**
 * Returns true if `selected` >= `minVersion`.
 */
export function isAtLeast(selected: string, minVersion: string): boolean {
  return compareVersions(selected, minVersion) >= 0;
}
