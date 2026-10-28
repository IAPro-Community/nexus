import type { RuntimeSession } from '../types';
import { isHonestNeedsUser } from '../app/documentTitle';

/** Server preconditions are repeated here for safe UI affordances, not trust. */
export interface AttentionResponseGuard {
  attention_fingerprint: string;
  attention_revision: number;
  attention_generation: string;
  interaction_id: string;
}

export function attentionResponseGuardForRuntime(
  runtime: RuntimeSession,
): AttentionResponseGuard | null {
  const revision = runtime.attention_revision || 0;
  if (
    !isHonestNeedsUser(runtime) ||
    runtime.attention_source !== 'INFERRED' ||
    runtime.attention_reason !== 'QUESTION' ||
    !runtime.attention_fingerprint ||
    !Number.isSafeInteger(revision) ||
    revision <= 0 ||
    runtime.attention_submitted_revision === revision
  ) {
    return null;
  }
  if (!['yn', 'choice', 'free_text'].includes(runtime.prompt_kind || '')) return null;
  // Exact decimal Go int64 generations must not be round-tripped through
  // JavaScript numbers, which can silently lose precision.
  if (!runtime.attention_generation) return null;
  return {
    attention_fingerprint: runtime.attention_fingerprint,
    attention_revision: revision,
    attention_generation: runtime.attention_generation,
    interaction_id: runtime.attention_interaction_id || '',
  };
}
