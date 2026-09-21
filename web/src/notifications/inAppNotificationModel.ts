import type { EventRecord, RuntimeSession } from '../types';

export type InAppNotificationAction = {
  type: 'account_handoff' | 'context_continue';
  label: string;
  sourceRuntimeId?: string;
  targetProvider: string;
  targetProfile: string;
  targetDisplayName?: string;
  isCrossProvider: boolean;
};

export type InAppNotification = {
  id: string;
  runtimeId?: string;
  projectName: string;
  title: string;
  message: string;
  tone: 'success' | 'danger' | 'warning';
  action?: InAppNotificationAction;
};

export type QuotaNotificationCopy = {
  degradedTitle: string;
  degradedMessage: (reason: string) => string;
};

export function notificationFromQuotaEvent(
  event: EventRecord,
  copy?: QuotaNotificationCopy,
): InAppNotification | null {
  if (event.type === 'QUOTA_MONITOR_DEGRADED') {
    const reason =
      (event.data?.reason as string) || event.summary || 'Leitura de quota indisponível.';
    return {
      id: `${event.id || 'quota-monitor'}:${event.type}`,
      projectName: 'Nexus Quota',
      title: copy?.degradedTitle || 'Monitoramento de quota degradado',
      message:
        copy?.degradedMessage(reason) ||
        `${reason} Nenhum alerta de consumo foi emitido com dados não confiáveis.`,
      tone: 'warning',
    };
  }
  if (event.type !== 'QUOTA_LOW' && event.type !== 'QUOTA_EXHAUSTED') {
    return null;
  }

  const provider =
    (event.data?.display_name as string) || (event.data?.provider as string) || 'Provedor';
  const remaining =
    typeof event.data?.remaining_percent === 'number' ? `${event.data.remaining_percent}%` : '';
  const affectedRuntimeId =
    (event.data?.affected_runtime_id as string) || event.runtime_id || undefined;
  const recProvider = event.data?.recommended_provider as string | undefined;
  const recProfile = event.data?.recommended_profile as string | undefined;
  const recDisplayName =
    (event.data?.recommended_display_name as string) ||
    (recProvider && recProfile ? `${recProvider} (${recProfile})` : undefined);
  const sourceProvider = (event.data?.provider as string) || '';
  const group = (event.data?.group as string) || '';
  const window = (event.data?.window as string) || '';
  const age = (event.data?.data_age as string) || '';
  const context = [group, window].filter(Boolean).join(' · ');
  const freshness = age ? ` Última leitura: ${age}.` : '';

  if (event.type === 'QUOTA_EXHAUSTED') {
    let action: InAppNotificationAction | undefined;
    if (recProvider && recProfile) {
      const isCrossProvider = sourceProvider.toLowerCase() !== recProvider.toLowerCase();
      action = {
        type: isCrossProvider ? 'context_continue' : 'account_handoff',
        label: isCrossProvider
          ? `Continuar com ${recDisplayName}`
          : `Alternar para ${recDisplayName}`,
        sourceRuntimeId: affectedRuntimeId,
        targetProvider: recProvider,
        targetProfile: recProfile,
        targetDisplayName: recDisplayName,
        isCrossProvider,
      };
    }

    return {
      id: `${event.id || 'quota-exhausted'}:${event.type}:${provider}`,
      runtimeId: affectedRuntimeId,
      projectName: 'Nexus Quota',
      title: `Quota esgotada: ${provider}`,
      message:
        event.summary ||
        `Quota do provedor ${provider}${context ? ` (${context})` : ''} esgotada (0%). Notificações pausadas até renovação.${freshness}`,
      tone: 'danger',
      action,
    };
  }

  return {
    id: `${event.id || 'quota-low'}:${event.type}:${provider}:${remaining}`,
    projectName: 'Nexus Quota',
    title: `Consumo de quota: ${provider} (${remaining})`,
    message:
      event.summary ||
      `Quota de tokens${context ? ` (${context})` : ''} reduzida para ${remaining}. Considere alternar de provedor ou aguardar reset.${freshness}`,
    tone: 'warning',
  };
}

export function notificationFromRuntime(runtime: RuntimeSession): InAppNotification | null {
  const reason = runtime.attention_reason;
  if (
    reason !== 'TASK_COMPLETED' &&
    reason !== 'ERROR' &&
    reason !== 'QUESTION' &&
    reason !== 'APPROVAL'
  ) {
    return null;
  }

  const message =
    runtime.attention_context ||
    runtime.last_task_summary ||
    runtime.dynamic_title ||
    (reason === 'TASK_COMPLETED'
      ? 'A tarefa foi concluída.'
      : reason === 'ERROR'
        ? 'O terminal reportou um erro.'
        : 'O agente espera sua resposta.');

  const projectName = runtime.project_name || 'Projeto';
  if (reason === 'TASK_COMPLETED') {
    return {
      id: `${runtime.runtime_id}:${reason}:${message}`,
      runtimeId: runtime.runtime_id,
      projectName,
      title: 'Tarefa concluída',
      message,
      tone: 'success',
    };
  }
  if (reason === 'ERROR') {
    return {
      id: `${runtime.runtime_id}:${reason}:${message}`,
      runtimeId: runtime.runtime_id,
      projectName,
      title: 'Erro no terminal',
      message,
      tone: 'danger',
    };
  }
  return {
    id: `${runtime.runtime_id}:${reason}:${message}`,
    runtimeId: runtime.runtime_id,
    projectName,
    title: reason === 'APPROVAL' ? 'Aprovação necessária' : 'Confirmação pendente',
    message,
    tone: 'warning',
  };
}

export type CapacityAlertCopy = {
  lowTitle: (provider: string, remaining: string) => string;
  exhaustedTitle: (provider: string) => string;
  recoveredTitle: (provider: string) => string;
  stateLabel: (state: string) => string;
  resetLabel: (reset: string) => string;
  freshnessLabel: (age: string) => string;
  recommendationLabel: (name: string, reason: string) => string;
};

// capacityAlertFromEvent maps new-monitor bus events (which carry state,
// fingerprint and optional recommendation) to cards. It returns null for
// legacy events without monitor state so the legacy mapper keeps handling
// them unchanged. IDs are fingerprint-based: repeats of the same condition
// collapse instead of stacking, and a new severity re-alerts.
export function capacityAlertFromEvent(
  event: EventRecord,
  copy: CapacityAlertCopy,
): InAppNotification | null {
  if (
    event.type !== 'QUOTA_LOW' &&
    event.type !== 'QUOTA_EXHAUSTED' &&
    event.type !== 'QUOTA_MONITOR_RECOVERED'
  ) {
    return null;
  }
  const state = event.data?.state as string | undefined;
  const fingerprint = event.data?.fingerprint as string | undefined;
  if (!state || !fingerprint) return null;

  const provider =
    (event.data?.display_name as string) || (event.data?.provider as string) || 'Provedor';
  const remaining =
    typeof event.data?.remaining_percent === 'number' ? `${event.data.remaining_percent}%` : '';
  const group = (event.data?.group as string) || '';
  const window = (event.data?.window as string) || '';
  const context = [group, window].filter(Boolean).join(' · ');
  const reset = (event.data?.reset_desc as string) || '';
  const age = (event.data?.data_age as string) || '';
  const recName =
    (event.data?.recommended_display_name as string) ||
    (event.data?.recommended_provider && event.data?.recommended_profile
      ? `${event.data.recommended_provider} (${event.data.recommended_profile})`
      : undefined);
  const recReason = (event.data?.recommendation_reason as string) || '';
  const affectedRuntimeId =
    (event.data?.affected_runtime_id as string) || event.runtime_id || undefined;

  const parts = [copy.stateLabel(state)];
  if (remaining) parts.push(remaining);
  if (context) parts.push(`(${context})`);
  if (reset) parts.push(copy.resetLabel(reset));
  if (age) parts.push(copy.freshnessLabel(age));
  if (recName) parts.push(copy.recommendationLabel(recName, recReason));
  const message = parts.filter(Boolean).join(' · ');

  if (event.type === 'QUOTA_MONITOR_RECOVERED') {
    return {
      id: `capacity:${fingerprint}`,
      projectName: 'Nexus Capacity',
      title: copy.recoveredTitle(provider),
      message,
      tone: 'success',
    };
  }

  let action: InAppNotificationAction | undefined;
  const recProvider = event.data?.recommended_provider as string | undefined;
  const recProfile = event.data?.recommended_profile as string | undefined;
  if (event.type === 'QUOTA_EXHAUSTED' && recProvider && recProfile) {
    const sourceProvider = (event.data?.provider as string) || '';
    const isCrossProvider = sourceProvider.toLowerCase() !== recProvider.toLowerCase();
    action = {
      type: isCrossProvider ? 'context_continue' : 'account_handoff',
      label: recName || `${recProvider} (${recProfile})`,
      sourceRuntimeId: affectedRuntimeId,
      targetProvider: recProvider,
      targetProfile: recProfile,
      targetDisplayName: recName,
      isCrossProvider,
    };
  }

  return {
    id: `capacity:${fingerprint}`,
    runtimeId: affectedRuntimeId,
    projectName: 'Nexus Capacity',
    title:
      event.type === 'QUOTA_EXHAUSTED'
        ? copy.exhaustedTitle(provider)
        : copy.lowTitle(provider, remaining),
    message,
    tone: event.type === 'QUOTA_EXHAUSTED' ? 'danger' : 'warning',
    action,
  };
}
