import type { RuntimeSession } from '../types';
import { isHonestNeedsUser } from '../app/documentTitle';

/** Server preconditions are repeated here for safe UI affordances, not trust. */
export interface AttentionResponseGuard {
  attention_fingerprint: string;
  attention_revision: number;
  provider_generation: number;
  host_generation: number;
  interaction_id: string;
}

export function attentionResponseGuardForRuntime(
  runtime: RuntimeSession,
): AttentionResponseGuard | null {
  const revision = runtime.attention_revision || 0;
  if (!isHonestNeedsUser(runtime) ||
      runtime.attention_source !== 'INFERRED' ||
      runtime.attention_reason !== 'QUESTION' ||
      !runtime.attention_fingerprint ||
      !Number.isSafeInteger(revision) ||
      revision <= 0 ||
      runtime.attention_submitted_revision === revision) {
    return null;
  }
  if (!['yn', 'choice', 'free_text'].includes(runtime.prompt_kind || '')) return null;
  const providerGeneration = runtime.provider_generation || 0;
  const hostGeneration = runtime.host_generation || 0;
  if (providerGeneration <= 0 && hostGeneration <= 0) return null;
  return {
    attention_fingerprint: runtime.attention_fingerprint,
    attention_revision: revision,
    provider_generation: providerGeneration,
    host_generation: hostGeneration,
    interaction_id: runtime.attention_interaction_id || '',
  };
}
