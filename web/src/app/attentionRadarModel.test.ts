import { describe, expect, it } from 'vitest';
import type { RuntimeSession } from '../types';
import { buildAttentionRadar, planFocusAttention } from './attentionRadarModel';

function rt(partial: Partial<RuntimeSession>): RuntimeSession {
  return {
    runtime_id: 'rt-1',
    workspace: '/tmp',
    pid: 1,
    host_pid: 1,
    state: 'RUNNING',
    control_level: 'TERMINAL',
    control_endpoint: '',
    started_at: '',
    ...partial,
  };
}

describe('buildAttentionRadar', () => {
  it('groups runtimes by project and marks honest needs_user', () => {
    const groups = buildAttentionRadar([
      rt({
        runtime_id: 'a',
        project_id: 'p1',
        project_name: 'Alpha',
        state: 'WAITING',
        attention_kind: 'needs_user',
        prompt_kind: 'yn',
        attention_context: 'Continue?',
        attention_fingerprint: 'fp1',
        attention_reason: 'QUESTION',
      }),
      rt({
        runtime_id: 'b',
        project_id: 'p2',
        project_name: 'Beta',
        state: 'RUNNING',
        attention_kind: 'working',
        attention_reason: 'WORKING',
      }),
    ]);
    expect(groups[0].projectName).toBe('Alpha');
    expect(groups[0].needsUserCount).toBe(1);
    expect(groups[1].items[0].badge).toBe('working');
  });

  it('does not promote chrome-only waits without context', () => {
    const groups = buildAttentionRadar([
      rt({
        project_id: 'p1',
        project_name: 'Alpha',
        state: 'WAITING',
        attention_reason: 'QUESTION',
        attention_kind: 'needs_user',
        prompt_kind: 'none',
        attention_context: '',
      }),
    ]);
    expect(groups[0].needsUserCount).toBe(0);
    expect(groups[0].items[0].badge).toBe('idle');
  });

  it('shows identical questions from independent runtimes as separate actionable rows', () => {
    const groups = buildAttentionRadar([
      rt({
        runtime_id: 'a',
        project_id: 'p1',
        project_name: 'Alpha',
        state: 'WAITING',
        attention_kind: 'needs_user',
        prompt_kind: 'yn',
        attention_context: 'Continue?',
        attention_reason: 'QUESTION',
      }),
      rt({
        runtime_id: 'b',
        project_id: 'p1',
        project_name: 'Alpha',
        state: 'WAITING',
        attention_kind: 'needs_user',
        prompt_kind: 'yn',
        attention_context: 'Continue?',
        attention_reason: 'QUESTION',
      }),
      rt({
        runtime_id: 'c',
        project_id: 'p1',
        project_name: 'Alpha',
        state: 'WAITING',
        attention_kind: 'needs_user',
        prompt_kind: 'yn',
        attention_context: 'Continue?',
        attention_reason: 'QUESTION',
      }),
    ]);
    expect(groups).toHaveLength(1);
    expect(groups[0].needsUserCount).toBe(3);
    expect(groups[0].items).toHaveLength(3);
  });

  it('rejects replacement-character-only context as dishonest', () => {
    const groups = buildAttentionRadar([
      rt({
        project_id: 'p1',
        project_name: 'Alpha',
        state: 'WAITING',
        attention_kind: 'needs_user',
        prompt_kind: 'free_text',
        attention_context: '\uFFFD\uFFFD\uFFFD\uFFFD',
        attention_reason: 'QUESTION',
      }),
    ]);
    expect(groups[0].needsUserCount).toBe(0);
    expect(groups[0].items[0].badge).toBe('idle');
  });

  it('does not mark bare RUNNING as working on the radar', () => {
    const groups = buildAttentionRadar([
      rt({
        runtime_id: 'idle-run',
        project_id: 'p1',
        project_name: 'Alpha',
        state: 'RUNNING',
        attention_kind: 'idle',
      }),
    ]);
    expect(groups[0].items[0].badge).toBe('idle');
  });
});

describe('planFocusAttention', () => {
  it('switches project then opens agent terminal by agent_id', () => {
    const actions = planFocusAttention(
      { projectId: 'p2', agentId: 'ag-9', runtimeId: 'rt-9' },
      { currentProjectId: 'p1', agentName: 'Backend' },
    );
    expect(actions).toEqual([
      { type: 'switch-project', projectId: 'p2' },
      { type: 'open-agent-terminal', agentId: 'ag-9', title: 'Backend', runtimeId: 'rt-9' },
      { type: 'refresh-agents', projectId: 'p2' },
    ]);
  });

  it('opens project shell when agent_id is empty', () => {
    const actions = planFocusAttention(
      { projectId: 'p1', runtimeId: 'rt-shell' },
      {
        currentProjectId: 'p1',
        runtime: rt({ runtime_id: 'rt-shell', project_id: 'p1', title: 'Project Shell' }),
      },
    );
    expect(actions).toEqual([
      {
        type: 'open-project-shell',
        projectId: 'p1',
        runtimeId: 'rt-shell',
        title: 'Project Shell',
      },
      { type: 'refresh-agents', projectId: 'p1' },
    ]);
  });
});
