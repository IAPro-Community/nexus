import { beforeEach, describe, expect, it, vi } from 'vitest';
import { NexusAPIError, nexusApi, setNexusCSRF } from './api';

describe('nexus API request errors', () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
    vi.stubGlobal('fetch', vi.fn());
    setNexusCSRF('');
  });

  it('preserves status and structured clarification payload on 409', async () => {
    const payload = {
      error: 'clarification_required',
      code: 'CONFLICT',
      clarification: {
        id: 'clr-1',
        project_id: 'proj-1',
        goal: 'Build product',
        status: 'PENDING',
        intent: { intent: 'Build product', scope: 'product', risk_level: 'MEDIUM' },
        unknowns: [
          {
            key: 'target',
            question: 'Which target?',
            level: 'BLOCKING',
            options: ['web', 'mobile'],
          },
        ],
        facts: {},
      },
    };
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 409,
        statusText: 'Conflict',
        json: () => Promise.resolve(payload),
      }),
    );

    try {
      await nexusApi.createPlan('proj-1', { goal: 'Build product', auto_plan: true });
      throw new Error('expected request to fail');
    } catch (error) {
      expect(error).toBeInstanceOf(NexusAPIError);
      const apiError = error as NexusAPIError<typeof payload>;
      expect(apiError.status).toBe(409);
      expect(apiError.payload.clarification.id).toBe('clr-1');
      expect(apiError.message).toBe('clarification_required');
      expect(apiError.code).toBe('CONFLICT');
    }
  });

  it('uses the shared transport CSRF state', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ status: 'ok' }),
    });
    vi.stubGlobal('fetch', fetchMock);
    setNexusCSRF('nexus-shared-csrf');

    await nexusApi.prepareContext('project-1', true);

    const [, init] = fetchMock.mock.calls[0];
    expect(init.headers.get('X-CSRF-Token')).toBe('nexus-shared-csrf');
  });
});

describe('system metadata API', () => {
  it('loads version and capability metadata through the Nexus client boundary', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          apiVersion: 'v1',
          serverVersion: 'dev',
          build: {},
          providers: ['codex'],
          capabilities: { codex: { terminal: true } },
        }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(nexusApi.getSystemInfo()).resolves.toMatchObject({
      apiVersion: 'v1',
      providers: ['codex'],
    });
    expect(String(fetchMock.mock.calls[0][0])).toBe('/api/v1/system/info');
  });
});

describe('WorkPlan response normalization', () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it('normalizes nullable phases and packages at the API boundary', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          plan: { id: 'plan-1', phases: [{ id: 'phase-1', packages: null }] },
          revisions: [],
        }),
    });
    vi.stubGlobal('fetch', fetchMock);

    const detail = await nexusApi.getPlan('plan-1');

    expect(detail.plan.phases).toEqual([{ id: 'phase-1', packages: [] }]);
  });

  it('turns a malformed plan list into an empty list instead of crashing', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({ ok: true, json: () => Promise.resolve(null) }),
    );

    await expect(nexusApi.getPlans('project-1')).resolves.toEqual([]);
  });
});

describe('mission manual-control API routes', () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it('uses dedicated take-control endpoint instead of generic pause', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ id: 'run-1', state: 'PAUSED' }),
    });
    vi.stubGlobal('fetch', fetchMock);
    await nexusApi.takeControlRun('run-1', 'manual');
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/runs/run-1/take-control');
  });

  it('uses dedicated return-to-mission endpoint instead of generic resume', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ id: 'run-1', state: 'EXECUTING' }),
    });
    vi.stubGlobal('fetch', fetchMock);
    await nexusApi.returnToMission('run-1');
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/runs/run-1/return-to-mission');
  });
});

describe('resource allocation API contracts', () => {
  it('preserves the scheduler decision and persistence result', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          decision: { policy: 'MANUAL', score: 1, explain_path: [] },
          persisted: true,
        }),
    });
    vi.stubGlobal('fetch', fetchMock);

    await expect(nexusApi.selectResource('agent-1', 'codex', 'work')).resolves.toMatchObject({
      decision: { policy: 'MANUAL', score: 1 },
      persisted: true,
    });
  });
});

describe('direct Agent and Project Shell API routes', () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it('submits Ask to the existing Agent endpoint with explicit start policy', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () =>
        Promise.resolve({
          agent_id: 'agt-1',
          runtime_id: 'rt-1',
          started: false,
          accepted: true,
        }),
    });
    vi.stubGlobal('fetch', fetchMock);
    await nexusApi.askAgent('agt-1', 'fix the tests', false);
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/agents/agt-1/ask');
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(JSON.parse(String(init.body))).toEqual({
      prompt: 'fix the tests',
      start_if_needed: false,
    });

    // With skills
    await nexusApi.askAgent('agt-1', 'fix the tests', true, ['skill-sec']);
    const init2 = fetchMock.mock.calls[1][1] as RequestInit;
    expect(JSON.parse(String(init2.body))).toEqual({
      prompt: 'fix the tests',
      start_if_needed: true,
      skill_ids: ['skill-sec'],
      scope: 'NEXT_PROMPT',
    });
  });

  it('starts an independent Project Shell through the project endpoint', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ runtime: { runtime_id: 'rt-shell' } }),
    });
    vi.stubGlobal('fetch', fetchMock);
    await nexusApi.startProjectShell('prj-1');
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/projects/prj-1/shell');
  });
});

describe('Flow Run evidence API', () => {
  beforeEach(() => {
    vi.unstubAllGlobals();
  });

  it('loads typed evidence from the run-scoped endpoint without a mutation request', async () => {
    const fetchMock = vi.fn().mockResolvedValue({
      ok: true,
      json: () => Promise.resolve({ run_id: 'run-1', capsules: [], receipts: [] }),
    });
    vi.stubGlobal('fetch', fetchMock);
    await nexusApi.getRunEvidence('run-1');
    expect(String(fetchMock.mock.calls[0][0])).toContain('/api/v1/runs/run-1/evidence');
    const init = (fetchMock.mock.calls[0][1] || {}) as RequestInit;
    expect(init.method).toBeUndefined();
  });
});
