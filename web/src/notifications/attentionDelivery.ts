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
