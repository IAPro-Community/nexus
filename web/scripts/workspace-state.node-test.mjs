import assert from 'node:assert/strict';
import { test } from 'node:test';
import { dumpWorkspaceStateInPage } from './workspace-state.mjs';

test('workspace diagnostics summarize local and server layouts in the page context', async () => {
  const originalStorage = Object.getOwnPropertyDescriptor(globalThis, 'localStorage');
  const originalFetch = Object.getOwnPropertyDescriptor(globalThis, 'fetch');
  const localLayout = {
    root: {
      kind: 'stack',
      tabs: [{ id: 'terminal-tab', type: 'terminal', data: { runtimeId: 'shell-123' } }],
      activeId: 'terminal-tab',
    },
  };
  const serverLayout = {
    root: {
      kind: 'stack',
      tabs: [{ id: 'server-tab', type: 'overview', data: {} }],
      activeId: 'server-tab',
    },
  };
  const storage = new Map([['nexus.workspace.layout', JSON.stringify(localLayout)]]);
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      get length() {
        return storage.size;
      },
      key: (index) => [...storage.keys()][index] ?? null,
      getItem: (key) => storage.get(key) ?? null,
    },
  });
  globalThis.fetch = async (url) => ({
    json: async () =>
      String(url) === '/api/v1/projects'
        ? { projects: [{ id: 'project-123' }] }
        : { revision: 7, layout: JSON.stringify(serverLayout) },
  });

  try {
    const result = await dumpWorkspaceStateInPage();
    assert.deepEqual(result.localStorage['nexus.workspace.layout'], {
      root: 'stack',
      tabs: [{ id: 'terminal-tab', type: 'terminal', runtimeId: 'shell-123' }],
      activeId: 'terminal-tab',
    });
    assert.equal(result.projectId, 'project-123');
    assert.equal(result.layoutRevision, 7);
    assert.deepEqual(result.serverLayout, {
      root: 'stack',
      tabs: [{ id: 'server-tab', type: 'overview', runtimeId: '' }],
      activeId: 'server-tab',
    });
    assert.equal(result.layoutError, undefined);
  } finally {
    if (originalStorage) Object.defineProperty(globalThis, 'localStorage', originalStorage);
    else delete globalThis.localStorage;
    if (originalFetch) Object.defineProperty(globalThis, 'fetch', originalFetch);
    else delete globalThis.fetch;
  }
});
