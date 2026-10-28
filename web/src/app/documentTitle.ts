import type { RuntimeSession } from '../types';
import { sanitizeAttentionText } from '../components/attentionText';

export type AttentionPromptKind = 'yn' | 'choice' | 'free_text' | 'native_status' | 'none' | '';

function honestContext(runtime: RuntimeSession): string {
  return sanitizeAttentionText(runtime.attention_context, '');
}

/** A stopped, failed or stale runtime cannot still request operator input. */
export function isActiveAttentionRuntime(runtime: RuntimeSession): boolean {
  return ['STARTING', 'RUNNING', 'WAITING', 'APPROVAL', 'DETACHED', 'HANDOFF'].includes(
    runtime.state,
  );
}

export function isHonestNeedsUser(runtime: RuntimeSession): boolean {
  if (!isActiveAttentionRuntime(runtime)) return false;
  const context = honestContext(runtime);
  if (!context) return false;
  if (runtime.attention_kind === 'needs_user') {
    return (
      runtime.prompt_kind === 'yn' ||
      runtime.prompt_kind === 'choice' ||
      runtime.prompt_kind === 'free_text' ||
      runtime.prompt_kind === 'native_status'
    );
  }
  // Explicit working/idle/completed/error clears a previous QUESTION reason.
  // Only legacy payloads without attention_kind may infer a wait from reason.
  if (runtime.attention_kind) return false;
  return runtime.attention_reason === 'QUESTION' || runtime.attention_reason === 'APPROVAL';
}

/** Per-runtime key used to suppress re-notify of the same wait on one terminal. */
export function attentionFingerprintOf(runtime: RuntimeSession): string {
  if (runtime.attention_fingerprint) return runtime.attention_fingerprint;
  const ctx = honestContext(runtime).toLowerCase().replace(/\s+/g, ' ');
  const kind = runtime.prompt_kind || 'none';
  return `${runtime.runtime_id}|${kind}|${ctx}`;
}

/** Semantic message key for presentation grouping only, never authorization identity. */
export function attentionMessageKey(runtime: RuntimeSession): string {
  const ctx = honestContext(runtime).toLowerCase().replace(/\s+/g, ' ');
  const kind = runtime.prompt_kind || 'none';
  return `${kind}|${ctx}`;
}

/** Count separately actionable waits by runtime + correlated provider fingerprint. */
export function distinctAttentionFingerprints(runtimes: RuntimeSession[]): string[] {
  const seen = new Set<string>();
  for (const runtime of runtimes) {
    if (!isHonestNeedsUser(runtime)) continue;
    seen.add(`${runtime.runtime_id}\0${attentionFingerprintOf(runtime)}`);
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
      (runtime) => isHonestNeedsUser(runtime) && `${runtime.runtime_id}\0${attentionFingerprintOf(runtime)}` === fingerprints[0],
    );
    const question = honestContext(sample || ({} as RuntimeSession)).split(/\n/)[0] || '';
    const short = question.length > 64 ? `${question.slice(0, 61)}...` : question;
    if (short) return `Nexus · ${name} · ${short}`;
  }
  return `Nexus · ${name} · ${fingerprints.length} esperando input`;
}
