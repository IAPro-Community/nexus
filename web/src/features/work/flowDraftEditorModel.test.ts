import { beforeEach, describe, expect, it } from 'vitest';
import { flowDraftSessionKey, loadStoredFlowDraft } from './flowDraftEditorModel';

// Node-environment vitest: mock sessionStorage like WorkspaceLayoutService.test.ts.
const store = new Map<string, string>();
const mockSessionStorage = {
  getItem: (key: string) => store.get(key) ?? null,
  setItem: (key: string, val: string) => {
    store.set(key, String(val));
  },
  removeItem: (key: string) => {
    store.delete(key);
  },
  clear: () => {
    store.clear();
  },
};

(globalThis as unknown as { window: unknown }).window = globalThis;
(globalThis as unknown as { sessionStorage: unknown }).sessionStorage = mockSessionStorage;

beforeEach(() => {
  store.clear();
});

describe('flowDraftSessionKey', () => {
  it('scopes keys per plan so drafts cannot cross-contaminate', () => {
    expect(flowDraftSessionKey('p1', 'plan-a')).toBe('nexus:flow:draft-edits:p1:plan-a');
    expect(flowDraftSessionKey('p1', 'plan-b')).toBe('nexus:flow:draft-edits:p1:plan-b');
    expect(flowDraftSessionKey('p1', 'plan-a')).not.toBe(flowDraftSessionKey('p1', 'plan-b'));
  });
});

describe('loadStoredFlowDraft', () => {
  const read = (stored: unknown) => {
    sessionStorage.setItem('k', JSON.stringify(stored));
    return loadStoredFlowDraft('k');
  };

  it('rejects payloads without numeric revision', () => {
    expect(read({ planId: 'a', revision: '3', draft: { steps: [] } })).toBeNull();
    expect(read({ planId: 'a', draft: { steps: [] } })).toBeNull();
  });

  it('passes drafts through; step-shape validation lives in the hook', () => {
    const payload = { planId: 'a', revision: 3, draft: { steps: [{ id: 1 }] } };
    expect(read(payload)).toEqual(payload);
  });

  it('accepts well-formed drafts', () => {
    const good = {
      planId: 'a',
      revision: 3,
      draft: { steps: [{ id: 's1', title: 'Step' }] },
    };
    expect(read(good)).toEqual(good);
  });

  it('returns null for missing or corrupt storage', () => {
    sessionStorage.removeItem('k');
    expect(loadStoredFlowDraft('k')).toBeNull();
    sessionStorage.setItem('k', '{broken');
    expect(loadStoredFlowDraft('k')).toBeNull();
  });
});
