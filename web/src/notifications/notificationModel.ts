export function notificationTitle(
  projectName: string | undefined,
  reason: string,
  dynamicTitle?: string,
): string {
  const project = projectName?.trim() || 'Projeto';
  const dynamic = dynamicTitle?.trim();
  if (dynamic) return `[Nexus | ${project}] ${dynamic}`;
  if (reason === 'QUESTION') return `[Nexus | ${project}] ❓ Pergunta Requer Atenção`;
  if (reason === 'APPROVAL') return `[Nexus | ${project}] ⚠️ Aprovação Necessária`;
  if (reason === 'TASK_COMPLETED') return `[Nexus | ${project}] ✅ Tarefa Concluída!`;
  return `[Nexus | ${project}] Notificação`;
}

import type { CapacityAlert, CapacityEvent } from '../types';

export function formatCapacityRemaining(percent: number | null | undefined): string {
  if (percent === null || percent === undefined || Number.isNaN(percent)) {
    return 'N/A';
  }
  return `${percent.toFixed(1)}%`;
}

export function capacitySeverityTone(
  severity: string,
): 'brand' | 'warning' | 'danger' | 'success' | 'default' {
  switch (severity?.toUpperCase()) {
    case 'CRITICAL':
    case 'EXHAUSTED':
      return 'danger';
    case 'WARNING':
      return 'warning';
    case 'RECOVERED':
      return 'success';
    case 'HEALTHY':
      return 'brand';
    default:
      return 'default';
  }
}

export function capacityAlertFromEvent(
  event: CapacityEvent | { payload_json?: string; [key: string]: any } | null | undefined,
): CapacityAlert | null {
  if (!event || typeof event !== 'object') {
    return null;
  }

  const raw = event as Record<string, any>;
  let payload: Record<string, any> = {};
  if (typeof raw.payload_json === 'string' && raw.payload_json.trim()) {
    try {
      payload = JSON.parse(raw.payload_json);
    } catch {
      return null;
    }
  } else if (typeof raw.payload === 'object' && raw.payload !== null) {
    payload = raw.payload;
  } else {
    payload = raw;
  }

  const provider = payload.provider || event.provider;
  const severity = payload.severity || event.severity;
  if (!provider || !severity) {
    return null;
  }

  return {
    id: payload.id || event.id || `cap-${Date.now()}`,
    fingerprint: payload.fingerprint || event.fingerprint || '',
    provider,
    profile: payload.profile || event.profile || 'default',
    group: payload.group || payload.quota_group || event.quota_group || 'general',
    window: payload.window || payload.window_kind || event.window_kind || 'sliding_5h',
    severity: (severity.toUpperCase() as CapacityAlert['severity']) || 'WARNING',
    remaining_percent:
      typeof payload.remaining_percent === 'number'
        ? payload.remaining_percent
        : typeof (event as any).remaining_percent === 'number'
          ? (event as any).remaining_percent
          : undefined,
    reset_at: payload.reset_at || payload.reset_description || (event as any).reset_at,
    freshness: payload.freshness || (event as any).freshness,
    confidence: payload.confidence || (event as any).confidence,
    subject: payload.subject || (event as any).subject || `Capacity alert for ${provider}`,
    model_name: payload.model_name || (event as any).model_name,
    recommendation: payload.recommendation || (event as any).recommendation,
    lock_screen_safe_title: payload.lock_screen_safe_title,
    lock_screen_safe_body: payload.lock_screen_safe_body,
    actions: Array.isArray(payload.actions) ? payload.actions : undefined,
  };
}
