import type { RuntimeSession } from '../types';
import { attentionFingerprintOf } from '../app/documentTitle';

/**
 * Attention delivery contract (agents vs terminals).
 *
 * - Source of truth for agent attention: registry + event bus (poll / control path).
 * - Attach WebSocket (AgentTerminal / TerminalPane) is display-only: may update
 *   local title from attention frames, but must never call sendPush.
 * - Browser push authority: NexusWorkspaceApp universal-registry-poll (cross-project).
 * - PTY regex on the host is TERMINAL-mode fallback only; shell never emits
 *   agent attention. Structured provider adapters (Codex app-server) are deferred
 *   — see DEV/AI_CONTROL_DEFERRED.md.
 */
export const ATTENTION_PUSH_AUTHORITY = 'universal-registry-poll' as const;

export const ATTENTION_ATTACH_WS_ROLE = 'display-only' as const;

/** True when the product Terminais tab is active and this PTY view is the focused one. */
export function isPtyAttentionFocused(input: {
  terminalsProductSurfaceId: string;
  agentViewId: string;
  stackActiveIds: Set<string>;
  activePtyViewId: string;
}): boolean {
  if (!input.stackActiveIds.has(input.terminalsProductSurfaceId)) return false;
  if (!input.agentViewId || !input.activePtyViewId) return false;
  return input.agentViewId === input.activePtyViewId;
}

/**
 * Per-intervention push identity, never the provider text fingerprint alone.
 * The same question from two agents, a new episode or a restarted process
 * must produce independent notifications; repeated polls stay idempotent.
 */
export function attentionPushKey(runtime: RuntimeSession): string {
  // JSON preserves field boundaries and escapes NUL separators for Notification.tag.
  return JSON.stringify([
    attentionResponseEpisodeKey(runtime),
    runtime.attention_reason || '',
    runtime.attention_kind || '',
  ]);
}

/**
 * Stable identity of one answerable episode across reconnects and clients.
 * Runtime, process generation, durable revision, fingerprint and provider
 * interaction id together identify exactly one permitted response.
 */
export function attentionResponseEpisodeKey(runtime: RuntimeSession): string {
  const generation =
    runtime.attention_generation ||
    `${runtime.host_generation || 0}:${runtime.provider_generation || 0}`;
  return JSON.stringify([
    runtime.runtime_id,
    generation,
    runtime.attention_revision || 0,
    attentionFingerprintOf(runtime),
    runtime.attention_interaction_id || '',
  ]);
}
