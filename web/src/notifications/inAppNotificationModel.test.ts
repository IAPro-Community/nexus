import { describe, expect, it } from 'vitest';
import { notificationFromQuotaEvent, notificationFromRuntime } from './inAppNotificationModel';

describe('notificationFromRuntime', () => {
  const runtime = {
    runtime_id: 'runtime-1',
    workspace: '/tmp/project',
    pid: 1,
    host_pid: 1,
    state: 'RUNNING' as const,
    control_level: 'FULL',
    control_endpoint: '',
    started_at: '',
    project_name: 'Projeto',
  };

  it('creates a transient notification for completion', () => {
    expect(
      notificationFromRuntime({
        ...runtime,
        attention_reason: 'TASK_COMPLETED',
        attention_context: 'Build finalizado.',
      }),
    ).toMatchObject({ title: 'Tarefa concluída', tone: 'success', message: 'Build finalizado.' });
  });

  it('creates a warning notification for interactive attention', () => {
    expect(
      notificationFromRuntime({
        ...runtime,
        attention_reason: 'QUESTION',
        attention_context: 'Continue?',
      }),
    ).toMatchObject({ title: 'Confirmação pendente', tone: 'warning', message: 'Continue?' });
  });
});

describe('notificationFromQuotaEvent', () => {
  it('creates a warning notification for QUOTA_LOW', () => {
    const notif = notificationFromQuotaEvent({
      id: 'ev-1',
      runtime_id: '',
      type: 'QUOTA_LOW',
      summary: 'Alerta de consumo: Claude Pro atingiu 20% de quota restante.',
      data: {
        provider: 'claude',
        profile: 'work',
        display_name: 'Claude Pro Work',
        remaining_percent: 20,
      },
      timestamp: '2026-09-06T04:00:00Z',
    });

    expect(notif).toMatchObject({
      title: 'Consumo de quota: Claude Pro Work (20%)',
      tone: 'warning',
      projectName: 'Nexus Quota',
      message: 'Alerta de consumo: Claude Pro atingiu 20% de quota restante.',
    });
  });

  it('creates a danger notification for QUOTA_EXHAUSTED', () => {
    const notif = notificationFromQuotaEvent({
      id: 'ev-2',
      runtime_id: '',
      type: 'QUOTA_EXHAUSTED',
      summary: 'Quota de Claude Pro Work esgotada (0%). Notificações pausadas até renovação.',
      data: {
        provider: 'claude',
        profile: 'work',
        display_name: 'Claude Pro Work',
        remaining_percent: 0,
      },
      timestamp: '2026-09-06T04:05:00Z',
    });

    expect(notif).toMatchObject({
      title: 'Quota esgotada: Claude Pro Work',
      tone: 'danger',
      projectName: 'Nexus Quota',
      message: 'Quota de Claude Pro Work esgotada (0%). Notificações pausadas até renovação.',
    });
  });

  it('returns null for non-quota events', () => {
    const notif = notificationFromQuotaEvent({
      id: 'ev-3',
      runtime_id: 'rt-1',
      type: 'TASK_COMPLETED',
      summary: 'Done',
      data: {},
      timestamp: '2026-09-06T04:00:00Z',
    });

    expect(notif).toBeNull();
  });

  it('creates an intra-provider account_handoff action when same provider recommended', () => {
    const notif = notificationFromQuotaEvent({
      id: 'ev-4',
      runtime_id: 'rt-123',
      type: 'QUOTA_EXHAUSTED',
      summary: 'Quota esgotada.',
      data: {
        provider: 'claude',
        profile: 'primary',
        display_name: 'Claude Primary',
        recommended_provider: 'claude',
        recommended_profile: 'secondary',
        recommended_display_name: 'Claude Secondary',
        affected_runtime_id: 'rt-123',
      },
      timestamp: '2026-09-06T04:10:00Z',
    });

    expect(notif).not.toBeNull();
    expect(notif?.runtimeId).toBe('rt-123');
    expect(notif?.action).toEqual({
      type: 'account_handoff',
      label: 'Alternar para Claude Secondary',
      sourceRuntimeId: 'rt-123',
      targetProvider: 'claude',
      targetProfile: 'secondary',
      targetDisplayName: 'Claude Secondary',
      isCrossProvider: false,
    });
  });

  it('creates a cross-provider context_continue action when different provider recommended', () => {
    const notif = notificationFromQuotaEvent({
      id: 'ev-5',
      runtime_id: 'rt-456',
      type: 'QUOTA_EXHAUSTED',
      summary: 'Quota esgotada.',
      data: {
        provider: 'claude',
        profile: 'primary',
        display_name: 'Claude Primary',
        recommended_provider: 'codex',
        recommended_profile: 'work',
        recommended_display_name: 'Codex Work',
        affected_runtime_id: 'rt-456',
      },
      timestamp: '2026-09-06T04:15:00Z',
    });

    expect(notif).not.toBeNull();
    expect(notif?.runtimeId).toBe('rt-456');
    expect(notif?.action).toEqual({
      type: 'context_continue',
      label: 'Continuar com Codex Work',
      sourceRuntimeId: 'rt-456',
      targetProvider: 'codex',
      targetProfile: 'work',
      targetDisplayName: 'Codex Work',
      isCrossProvider: true,
    });
  });
});
