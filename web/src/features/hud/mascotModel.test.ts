import { describe, expect, it } from 'vitest';
import type { AttentionGroup } from '../../types';
import type { HudAlert, HudState } from './hudModel';
import {
  mascotMood,
  mascotMotionAllowed,
  nextMascotBubble,
  projectMascotAttention,
  readMascotSnoozes,
  snoozeMascotAttention,
} from './mascotModel';

const state = (overrides: Partial<HudState> = {}): HudState => ({
  generated_at: '2026-10-24T00:00:00Z',
  providers: [],
  agents: [],
  missions: [],
  alerts: [],
  primary: { kind: 'NONE' },
  system: {
    capacity: 'READY',
    resources: 'READY',
    agents: 'READY',
    missions: 'READY',
    cost: 'READY',
    cache_ttl_millis: 15000,
  },
  ...overrides,
});

const attentionGroup = (needsYou: AttentionGroup['needs_you']): AttentionGroup => ({
  needs_you: needsYou,
  completed: [],
  failed: [],
  all_items: needsYou,
  total_needs: needsYou.length,
});

const missionAttention = (level: AttentionGroup['needs_you'][number]['level'], id: string) => ({
  mission_id: id,
  project_id: 'project-1',
  state: 'BLOCKED_NEEDS_USER',
  level,
  reason_code: 'INTERVENTION_REQUIRED',
  summary: 'A mission needs a decision',
  question: 'Which option should be used?',
  impact: 'Execution is paused',
  recommended_actions: ['Review the mission'],
  age_seconds: 4,
});

describe('NEX Mascot projection', () => {
  it('treats the Attention Center as authoritative and includes only REQUIRE_USER missions', () => {
    const projected = projectMascotAttention(
      state({
        missions: [
          {
            run_id: 'legacy-run',
            project_id: 'project-1',
            state: 'BLOCKED_NEEDS_USER',
            current_stage: 'build',
            completed_packages: 0,
            total_packages: 1,
            needs_user: true,
            updated_at: '2026-10-24T00:00:00Z',
          },
        ],
      }),
      attentionGroup([
        missionAttention('IN_APP', 'not-actionable'),
        missionAttention('REQUIRE_USER', 'actionable'),
      ]),
    );

    expect(projected).toHaveLength(1);
    expect(projected[0].recordId).toBe('actionable');
    expect(projected[0].severity).toBe('critical');
  });

  it('deduplicates runtime attention against the matching waiting Agent', () => {
    const projected = projectMascotAttention(
      state({
        agents: [
          {
            id: 'agent-1',
            project_id: 'project-1',
            name: 'Builder',
            status: 'AWAITING_INPUT',
          },
        ],
        runtime_attention: [
          {
            runtime_id: 'runtime-1',
            agent_id: 'agent-1',
            project_id: 'project-1',
            provider: 'codex',
            reason: 'APPROVAL',
            prompt_kind: 'yn',
            fingerprint: 'approval-1',
            updated_at: '2026-10-24T00:00:00Z',
          },
        ],
      }),
    );

    expect(projected).toHaveLength(1);
    expect(projected[0].id).toBe('runtime:approval-1');
  });

  it('surfaces a failed Mission without turning it into a request for human intervention', () => {
    const failed = missionAttention('IN_APP', 'failed-run');
    const projected = projectMascotAttention(state(), {
      ...attentionGroup([]),
      failed: [failed],
      all_items: [failed],
    });

    expect(projected).toHaveLength(1);
    expect(projected[0].severity).toBe('critical');
    expect(projected[0].needsUser).toBe(false);
    expect(projected[0].titleKey).toBe('hud.mascot.missionFailedTitle');
  });

  it('maps attention severity without celebrating unverified completion', () => {
    const criticalAttention = {
      id: 'critical',
      titleKey: 'hud.mascot.missionTitle',
      detailKey: 'hud.mascot.missionDetail',
      severity: 'critical' as const,
      target: 'missions' as const,
      needsUser: true,
    };
    const warningAttention = { ...criticalAttention, severity: 'warning' as const };
    expect(mascotMood(state(), [], false)).toBe('idle');
    const completedMission = {
      run_id: 'done',
      project_id: 'project-1',
      state: 'COMPLETED',
      current_stage: 'build',
      completed_packages: 1,
      total_packages: 1,
      needs_user: false,
      updated_at: '2026-10-24T00:00:00Z',
    };
    expect(mascotMood(state({ missions: [completedMission] }), [], false)).toBe('idle');
    expect(
      mascotMood(
        state({ missions: [{ ...completedMission, state: 'COMPLETED_VERIFIED' }] }),
        [],
        false,
      ),
    ).toBe('idle');
    expect(mascotMood(state(), [criticalAttention], false)).toBe('critical');
    expect(mascotMood(state(), [warningAttention], false)).toBe('attention');
    expect(mascotMood(state(), [], true)).toBe('offline');
  });

  it('does not animate when motion is disabled or Nexus is offline', () => {
    expect(mascotMotionAllowed('idle', true)).toBe(true);
    expect(mascotMotionAllowed('working', false)).toBe(false);
    expect(mascotMotionAllowed('offline', true)).toBe(false);
  });

  it('uses offline only when every configured provider is unavailable', () => {
    const unavailable = {
      provider: 'codex',
      profile: 'default',
      health: 'UNAVAILABLE',
      availability: 'UNAVAILABLE',
      capacity_state: 'UNKNOWN',
      freshness: 'UNKNOWN',
      rate_limited: false,
      windows: [],
    };
    const available = { ...unavailable, availability: 'AVAILABLE' };

    expect(mascotMood(state({ providers: [unavailable] }), [], false)).toBe('offline');
    expect(mascotMood(state({ providers: [unavailable, available] }), [], false)).toBe('idle');
  });

  it('persists bounded attention snoozes and chooses the next unsnoozed item', () => {
    const storageValues = new Map<string, string>();
    const storage = {
      getItem: (key: string) => storageValues.get(key) ?? null,
      setItem: (key: string, value: string) => storageValues.set(key, value),
    };
    const items = [
      {
        id: 'first',
        titleKey: 'hud.mascot.missionTitle',
        detailKey: 'hud.mascot.missionDetail',
        severity: 'critical' as const,
        target: 'missions' as const,
        needsUser: true,
      },
      {
        id: 'second',
        titleKey: 'hud.mascot.agentTitle',
        detailKey: 'hud.mascot.agentDetail',
        severity: 'warning' as const,
        target: 'agents' as const,
        needsUser: true,
      },
    ];

    snoozeMascotAttention(storage, 'first', 1000, 5000);
    const snoozes = readMascotSnoozes(storage, 1001);

    expect(nextMascotBubble(items, snoozes, 1001)?.id).toBe('second');
    expect(nextMascotBubble(items, snoozes, 6001)?.id).toBe('first');
    expect(nextMascotBubble(items, {}, 1001, { first: true })?.id).toBe('second');
  });

  it('continues reading snoozes written under the legacy companion key', () => {
    const storage = {
      getItem: (key: string) =>
        key === 'iapro:nexus:companion:snooze:v1' ? JSON.stringify({ first: 5000 }) : null,
      setItem: () => undefined,
    };

    expect(readMascotSnoozes(storage, 1000)).toEqual({ first: 5000 });
  });

  it('ignores informational and acknowledged capacity alerts', () => {
    const alert = (patch: Partial<HudAlert>): HudAlert => ({
      fingerprint: 'capacity-1',
      category: 'quota',
      severity: 'WARNING',
      acked: false,
      last_observed_at: '2026-10-24T00:00:00Z',
      ...patch,
    });
    const projected = projectMascotAttention(
      state({
        alerts: [alert({ severity: 'INFO' }), alert({ fingerprint: 'acked', acked: true })],
      }),
    );

    expect(projected).toEqual([]);
  });

  it('treats null arrays from Go JSON as empty at the projection boundary', () => {
    const nullableState = {
      ...state(),
      providers: null,
      agents: null,
      missions: null,
      alerts: null,
      runtime_attention: null,
    } as unknown as HudState;
    const nullableAttention = {
      needs_you: null,
      completed: null,
      failed: null,
      all_items: null,
      total_needs: 0,
    } as unknown as AttentionGroup;

    expect(projectMascotAttention(nullableState, nullableAttention)).toEqual([]);
    expect(mascotMood(nullableState, [], false)).toBe('idle');
  });
});
