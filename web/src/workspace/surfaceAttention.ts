import type { RuntimeSession } from '../types';
import { isHonestNeedsUser } from '../app/documentTitle';
import { sanitizeAttentionText } from '../components/attentionText';

export type SurfaceAttentionKind = 'needs_user' | 'completed' | 'error' | 'working' | '';

export function attentionKindFromRuntime(runtime?: RuntimeSession | null): SurfaceAttentionKind {
  if (!runtime) return '';
  // Match Radar, document title and toast rules. A stale QUESTION reason
  // without a live, actionable prompt must not mark this surface unread.
  if (isHonestNeedsUser(runtime)) {
    return 'needs_user';
  }
  if (
    runtime.attention_reason === 'ERROR' ||
    runtime.attention_kind === 'error' ||
    runtime.state === 'FAILED'
  ) {
    return 'error';
  }
  if (runtime.attention_reason === 'TASK_COMPLETED' || runtime.attention_kind === 'completed') {
    return 'completed';
  }
  if (runtime.attention_reason === 'WORKING' || runtime.attention_kind === 'working') {
    return 'working';
  }
  return '';
}

export function statusSuffixFromRuntime(runtime?: RuntimeSession | null): string {
  const kind = attentionKindFromRuntime(runtime);
  switch (kind) {
    case 'needs_user':
      return 'pergunta';
    case 'completed':
      return 'concluído';
    case 'error':
      return 'erro';
    case 'working':
      return 'trabalhando';
    default:
      return '';
  }
}

/**
 * Keep the agent name stable; status is a short suffix, never a full replacement.
 * dynamic_title may inform document title / secondary labels, not the surface title root.
 */
export function surfaceTitleFromAgent(
  agentName: string,
  runtime?: RuntimeSession | null,
  customTitle?: string,
): {
  title: string;
  statusSuffix: string;
  dynamicTitle: string;
  hasAttention: boolean;
  attentionKind: SurfaceAttentionKind;
  fingerprint: string;
} {
  const name = (customTitle || agentName || '').trim() || 'Agent';
  const statusSuffix = statusSuffixFromRuntime(runtime);
  const dynamicTitle = sanitizeAttentionText(runtime?.dynamic_title, '').trim();
  const kind = attentionKindFromRuntime(runtime);
  const hasAttention = kind === 'needs_user' || kind === 'completed' || kind === 'error';
  const fingerprint =
    runtime?.attention_fingerprint ||
    `${runtime?.runtime_id || ''}|${runtime?.prompt_kind || ''}|${sanitizeAttentionText(runtime?.attention_context, '').toLowerCase()}`;
  return {
    title: statusSuffix ? `${name} · ${statusSuffix}` : name,
    statusSuffix,
    dynamicTitle,
    hasAttention,
    attentionKind: kind,
    fingerprint: hasAttention ? fingerprint : '',
  };
}

export function shouldMarkUnread(opts: {
  previousFingerprint?: string;
  nextFingerprint: string;
  hasAttention: boolean;
  surfaceFocused: boolean;
  attentionKind: SurfaceAttentionKind;
}): boolean {
  if (!opts.hasAttention || !opts.nextFingerprint) return false;
  if (opts.attentionKind === 'working') return false;
  if (opts.surfaceFocused) return false;
  return opts.previousFingerprint !== opts.nextFingerprint;
}
