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
