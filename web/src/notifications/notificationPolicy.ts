import { sanitizeAttentionText } from '../components/attentionText';
import { notificationTitle } from './notificationModel';

export type NotificationTransport = 'IN_APP' | 'BROWSER' | 'WEB_PUSH' | 'NATIVE_OS';

export interface NotificationPolicyInput {
  runtimeId: string;
  projectId?: string;
  projectName?: string;
  reason: string;
  context?: string;
  summary?: string;
  dynamicTitle?: string;
  attentionEventId?: string;
  view?: string;
  credentials?: unknown;
  transcript?: unknown;
}

export interface NotificationPolicyContext {
  hasActiveClients: boolean;
  visible: boolean;
  hasFocus: boolean;
  backgroundTransport?: Extract<NotificationTransport, 'BROWSER' | 'NATIVE_OS'>;
}

export interface NotificationEnvelope {
  transport: NotificationTransport;
  dedupeKey: string;
  title: string;
  body: string;
  tag: string;
  data: {
    attentionEventId?: string;
    projectId?: string;
    reason: string;
    runtimeId: string;
    view: string;
  };
}

const DEFAULT_BODY = 'Abra o Nexus para revisar a atenção pendente.';
const DEFAULT_VIEW = 'overview';
const MAX_TITLE_LENGTH = 120;
const MAX_BODY_LENGTH = 240;

export function resolveNotificationTransport({
  hasActiveClients,
  visible,
  hasFocus,
  backgroundTransport = 'BROWSER',
}: NotificationPolicyContext): NotificationTransport {
  if (!hasActiveClients) return 'WEB_PUSH';
  if (visible && hasFocus) return 'IN_APP';
  return backgroundTransport;
}

export function prepareNotificationEnvelope(
  payload: NotificationPolicyInput,
  context: NotificationPolicyContext,
): NotificationEnvelope {
  const transport = resolveNotificationTransport(context);
  const body = boundedNotificationBody(payload);
  const dedupeKey = boundedText(payload.attentionEventId, '') || `${payload.runtimeId}:${payload.reason}:${body}`;
  const title = truncate(notificationTitle(payload.projectName, payload.reason, payload.dynamicTitle), MAX_TITLE_LENGTH);

  return {
    transport,
    dedupeKey,
    title,
    body,
    tag: `nexus-${dedupeKey}`,
    data: {
      attentionEventId: boundedText(payload.attentionEventId, '') || undefined,
      projectId: boundedText(payload.projectId, '') || undefined,
      reason: payload.reason,
      runtimeId: payload.runtimeId,
      view: boundedText(payload.view, DEFAULT_VIEW) || DEFAULT_VIEW,
    },
  };
}

export function buildNotificationTarget(
  baseUrl: string,
  data: { projectId?: string; view?: string },
): string {
  const url = new URL(baseUrl);
  url.pathname = '/';
  url.search = '';

  const projectId = boundedText(data.projectId, '');
  const view = boundedText(data.view, DEFAULT_VIEW) || DEFAULT_VIEW;

  if (projectId) url.searchParams.set('project', projectId);
  url.searchParams.set('view', view);
  return url.toString();
}

function boundedNotificationBody(payload: NotificationPolicyInput): string {
  const candidates = [payload.context, payload.summary, payload.dynamicTitle];

  for (const candidate of candidates) {
    const text = boundedText(candidate, '');
    if (!text) continue;

    if (containsSensitiveMarker(text)) {
      continue;
    }

    const redacted = redactInlineSecrets(text);
    if (containsSensitiveMarker(redacted)) {
      continue;
    }

    return truncate(redacted, MAX_BODY_LENGTH);
  }

  return truncate(DEFAULT_BODY, MAX_BODY_LENGTH);
}

function boundedText(value: unknown, fallback: string): string {
  return sanitizeAttentionText(value, fallback);
}

function redactInlineSecrets(text: string): string {
  return text
    .replace(/\bBearer\s+[^\s]+/gi, '[redacted]')
    .replace(/\b(?:sk|pk|rk|ghp)_[A-Za-z0-9_-]+\b/g, '[redacted]')
    .replace(/\b(?:password|token|secret|authorization|api[-_ ]?key)\s*[:=]\s*[^\s]+/gi, '[redacted]');
}

function containsSensitiveMarker(text: string): boolean {
  return /\b(?:transcript|credentials?|password|token|secret|authorization|bearer|api[-_ ]?key)\b/i.test(text);
}

function truncate(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return `${text.slice(0, maxLength - 1).trimEnd()}…`;
}
