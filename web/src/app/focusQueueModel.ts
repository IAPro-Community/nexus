import type { RuntimeSession } from '../types';
import { sanitizeAttentionText } from '../components/attentionText';
import {
  attentionFingerprintOf,
  isActiveAttentionRuntime,
  isHonestNeedsUser,
} from './documentTitle';

/**
 * Prioritized pending queue of everything that needs an operator decision.
 *
 * Severity first (`needs_user` before `error`), then the oldest pending
 * episode first. Repeated list entries of the same attention episode collapse
 * into one row, and an episode already answered for its current durable
 * revision never re-enters the queue — matching the server's one-shot
 * attention claim, so a refresh cannot reopen resolved actions.
 */
export type FocusQueueSeverity = 'needs_user' | 'error';

export type FocusQueueItem = {
  /** Stable identity for list rendering and deduplication. */
  key: string;
  runtimeId: string;
  agentId?: string;
  projectId: string;
  projectName: string;
  title: string;
  severity: FocusQueueSeverity;
  context?: string;
  fingerprint?: string;
  promptKind?: string;
  /** Milliseconds the current episode has been pending. */
  waitedMs: number;
};

const SEVERITY_RANK: Record<FocusQueueSeverity, number> = { needs_user: 0, error: 1 };

export function focusQueueKey(
  runtime: RuntimeSession,
  severity: FocusQueueSeverity,
  fingerprint: string,
): string {
  // An attention fingerprint identifies one durable episode across refreshes;
  // errors fall back to their runtime identity.
  return fingerprint ? `fp:${fingerprint}` : `rt:${runtime.runtime_id}|${severity}`;
}

function pendingSinceMs(runtime: RuntimeSession, now: number): number {
  const basis = runtime.updated_at || runtime.started_at;
  if (!basis) return 0;
  const at = Date.parse(basis);
  if (!Number.isFinite(at)) return 0;
  return Math.max(0, now - at);
}

function isObservedError(runtime: RuntimeSession): boolean {
  return runtime.attention_kind === 'error' || runtime.attention_reason === 'ERROR';
}

export function buildFocusQueue(
  runtimes: RuntimeSession[],
  now: number = Date.now(),
): FocusQueueItem[] {
  const list = Array.isArray(runtimes) ? runtimes : [];
  const seen = new Set<string>();
  const items: FocusQueueItem[] = [];

  for (const runtime of list) {
    if (!runtime || !isActiveAttentionRuntime(runtime)) continue;

    // Answered episodes never reopen: the server persists the submitted
    // revision with a one-shot claim, so a submitted revision at or past the
    // displayed revision means this exact episode was already handled.
    const revision = runtime.attention_revision ?? 0;
    const submitted = runtime.attention_submitted_revision ?? 0;
    if (revision > 0 && submitted >= revision) continue;

    const honest = isHonestNeedsUser(runtime);
    if (!honest && !isObservedError(runtime)) continue;
    const severity: FocusQueueSeverity = honest ? 'needs_user' : 'error';

    const fingerprint = honest ? attentionFingerprintOf(runtime) : '';
    const key = focusQueueKey(runtime, severity, fingerprint);
    if (seen.has(key)) continue;
    seen.add(key);

    const projectId = runtime.project_id || runtime.workspace || 'unknown';
    const title = runtime.dynamic_title || runtime.title || runtime.runtime_id;
    const context = honest ? sanitizeAttentionText(runtime.attention_context, '') : '';
    items.push({
      key,
      runtimeId: runtime.runtime_id,
      agentId: runtime.agent_id || undefined,
      projectId,
      projectName: runtime.project_name || projectId,
      title,
      severity,
      context: context || undefined,
      fingerprint: honest ? fingerprint : undefined,
      promptKind: runtime.prompt_kind,
      waitedMs: pendingSinceMs(runtime, now),
    });
  }

  return items.sort((a, b) => {
    const rank = SEVERITY_RANK[a.severity] - SEVERITY_RANK[b.severity];
    if (rank !== 0) return rank;
    // Oldest pending episode first; the key tiebreak keeps rebuilds stable.
    if (b.waitedMs !== a.waitedMs) return b.waitedMs - a.waitedMs;
    return a.key.localeCompare(b.key);
  });
}
