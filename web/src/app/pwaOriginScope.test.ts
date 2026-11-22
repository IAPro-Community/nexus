import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';

/**
 * The PWA only survives a Nexus restart if its origin is derived from the
 * host that served it. A Named Tunnel keeps the hostname stable; a hardcoded
 * origin in the manifest or the service worker registration would pin the
 * installed app to a dead hostname the moment the tunnel changes.
 */
const manifest = JSON.parse(
  readFileSync(new URL('../../public/manifest.webmanifest', import.meta.url), 'utf8'),
);
const serviceWorker = readFileSync(new URL('../../public/sw.js', import.meta.url), 'utf8');
const entrypoint = readFileSync(new URL('../index.tsx', import.meta.url), 'utf8');

describe('PWA origin portability', () => {
  it('keeps the manifest origin-relative', () => {
    expect(manifest.id).toBe('/');
    expect(manifest.start_url).toBe('/');
    expect(manifest.scope).toBe('/');
  });

  it('keeps every manifest asset origin-relative', () => {
    for (const icon of manifest.icons) {
      expect(icon.src.startsWith('/')).toBe(false);
      expect(icon.src.startsWith('./')).toBe(true);
      expect(icon.src).not.toMatch(/^https?:\/\//);
    }
  });

  it('registers the service worker at the root scope of the current origin', () => {
    expect(entrypoint).toContain("navigator.serviceWorker.register('/sw.js')");
  });

  it('never pins an absolute origin in the service worker', () => {
    expect(serviceWorker).not.toMatch(/https?:\/\/(?!www\.w3\.org)/);
  });
});
