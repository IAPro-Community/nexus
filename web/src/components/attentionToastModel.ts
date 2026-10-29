import type { RuntimeSession } from '../types';
import { attentionFingerprintOf } from '../app/documentTitle';

/**
 * A notification belongs to one runtime and one provider interaction.
 * Context text is not an identity: two providers can ask the same question.
 */
export function attentionToastKey(runtime: RuntimeSession): string {
  // Go int64 generation tokens must be compared as exact decimal text: JavaScript
  // numbers cannot distinguish neighboring generations above 2^53.
  const generation =
    runtime.attention_generation ||
    `${runtime.host_generation || 0}:${runtime.provider_generation || 0}`;
  return `${runtime.runtime_id}\0${generation}:${runtime.attention_revision || 0}\0${attentionFingerprintOf(runtime)}`;
}
