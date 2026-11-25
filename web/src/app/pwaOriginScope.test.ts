import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { runInNewContext } from 'node:vm';

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

  it('bypasses dynamic workspace routes and resolves offline asset misses as 503', async () => {
    const listeners = new Map<string, (event: unknown) => void>();
    runInNewContext(serviceWorker, {
      self: {
        location: new URL('http://127.0.0.1:13000/'),
        addEventListener: (type: string, listener: (event: unknown) => void) =>
          listeners.set(type, listener),
      },
      caches: { match: async () => null },
      fetch: async () => {
        throw new Error('offline');
      },
      Promise,
      URL,
      Response,
    });

    const handleFetch = listeners.get('fetch');
    expect(handleFetch).toBeDefined();
    let workspaceResponse: Promise<Response> | undefined;
    handleFetch?.({
      request: {
        method: 'GET',
        url: 'http://127.0.0.1:13000/p/prj_1/projects',
        mode: 'cors',
      },
      respondWith: (response: Promise<Response>) => {
        workspaceResponse = response;
      },
    });
    expect(workspaceResponse).toBeUndefined();

    let assetResponse: Promise<Response> | undefined;
    handleFetch?.({
      request: {
        method: 'GET',
        url: 'http://127.0.0.1:13000/assets/app.js',
        mode: 'no-cors',
      },
      respondWith: (response: Promise<Response>) => {
        assetResponse = response;
      },
    });
    await expect(assetResponse).resolves.toMatchObject({ status: 503, statusText: 'Offline' });
  });
});
