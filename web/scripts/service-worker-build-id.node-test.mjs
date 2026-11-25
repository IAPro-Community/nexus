import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import test from 'node:test';
import { serviceWorkerBuildId } from './service-worker-build-id.mjs';

test('service worker build ID is stable for a build and changes with emitted JavaScript', () => {
  const assets = [
    { name: 'bundle.js', content: 'main bundle' },
    { name: 'chunks/BrowserSurface.js', content: 'surface bundle' },
  ];

  assert.equal(serviceWorkerBuildId(assets), serviceWorkerBuildId([...assets].reverse()));
  assert.notEqual(
    serviceWorkerBuildId(assets),
    serviceWorkerBuildId([assets[0], { name: 'chunks/BrowserSurface.js', content: 'new build' }]),
  );
});

test('service worker activation deletes caches from previous builds', async () => {
  const source = await readFile(new URL('../public/sw.js', import.meta.url), 'utf8');
  const script = source.replace('__NEXUS_BUILD_ID__', 'build-current');
  const listeners = new Map();
  const deleted = [];
  let activation;

  const context = {
    self: {
      location: new URL('http://localhost/'),
      addEventListener: (type, listener) => listeners.set(type, listener),
      clients: { claim: async () => undefined },
    },
    caches: {
      keys: async () => ['nexus-shell-build-old', 'nexus-shell-build-current', 'unrelated'],
      delete: async (key) => deleted.push(key),
    },
    Promise,
    URL,
    Response,
  };
  await import('node:vm').then(({ runInNewContext }) => runInNewContext(script, context));
  listeners.get('activate')({ waitUntil: (promise) => (activation = promise) });
  await activation;

  assert.deepEqual(deleted, ['nexus-shell-build-old']);
});
