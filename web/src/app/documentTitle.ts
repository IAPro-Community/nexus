import type { RuntimeSession } from '../types';
import { sanitizeAttentionText } from '../components/attentionText';

export type AttentionPromptKind = 'yn' | 'choice' | 'free_text' | 'native_status' | 'none' | '';

function honestContext(runtime: RuntimeSession): string {
  return sanitizeAttentionText(runtime.attention_context, '');
}

export function isHonestNeedsUser(runtime: RuntimeSession): boolean {
  const context = honestContext(runtime);
  if (!context) return false;
  if (runtime.attention_kind === 'needs_user') {
    return runtime.prompt_kind !== 'none' && runtime.prompt_kind !== '';
  }
  // Legacy payloads without attention_kind: require QUESTION/APPROVAL + real context.
  if (runtime.attention_reason !== 'QUESTION' && runtime.attention_reason !== 'APPROVAL') {
    return false;
  }
  return true;
}

/** Per-runtime key used to suppress re-notify of the same wait on one terminal. */
export function attentionFingerprintOf(runtime: RuntimeSession): string {
  if (runtime.attention_fingerprint) return runtime.attention_fingerprint;
  const ctx = honestContext(runtime).toLowerCase().replace(/\s+/g, ' ');
  const kind = runtime.prompt_kind || 'none';
  return `${runtime.runtime_id}|${kind}|${ctx}`;
}

/**
 * Cross-runtime message key: identical question text collapses to one title/radar count
 * even when several Agents share the same prompt.
 */
export function attentionMessageKey(runtime: RuntimeSession): string {
  const ctx = honestContext(runtime).toLowerCase().replace(/\s+/g, ' ');
  const kind = runtime.prompt_kind || 'none';
  return `${kind}|${ctx}`;
}

/** Distinct honest waits by message content (not by runtime_id). */
export function distinctAttentionFingerprints(runtimes: RuntimeSession[]): string[] {
  const seen = new Set<string>();
  for (const runtime of runtimes) {
    if (!isHonestNeedsUser(runtime)) continue;
    seen.add(attentionMessageKey(runtime));
  }
  return [...seen];
}

export function buildDocumentTitle(projectName: string, runtimes: RuntimeSession[]): string {
  const name = (projectName || 'Project').trim() || 'Project';
  const fingerprints = distinctAttentionFingerprints(Array.isArray(runtimes) ? runtimes : []);
  if (fingerprints.length === 0) {
    return `Nexus · ${name}`;
  }
  if (fingerprints.length === 1) {
    const sample = runtimes.find(
      (runtime) => isHonestNeedsUser(runtime) && attentionMessageKey(runtime) === fingerprints[0],
    );
    const question = honestContext(sample || ({} as RuntimeSession)).split(/\n/)[0] || '';
    const short = question.length > 64 ? `${question.slice(0, 61)}...` : question;
    if (short) return `Nexus · ${name} · ${short}`;
  }
  return `Nexus · ${name} · ${fingerprints.length} esperando input`;
}
