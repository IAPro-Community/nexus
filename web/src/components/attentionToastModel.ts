import type { RuntimeSession } from '../types';
import { attentionFingerprintOf } from '../app/documentTitle';

/**
 * A notification belongs to one runtime and one provider interaction.
 * Context text is not an identity: two providers can ask the same question.
 */
export function attentionToastKey(runtime: RuntimeSession): string {
  return `${runtime.runtime_id}\0${attentionFingerprintOf(runtime)}`;
}
