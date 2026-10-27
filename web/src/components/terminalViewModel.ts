import type { RuntimeSession } from '../types';

export type TerminalSplitMode = 'single' | 'split-h' | 'grid';

/** Keep terminal views bound to sessions the user explicitly opened. */
export function normalizeOpenRuntimeIds(
  openIds: string[],
  runtimes: RuntimeSession[],
  activeRuntimeId?: string,
): string[] {
  const available = new Set(runtimes.map((runtime) => runtime.runtime_id));
  const next = openIds.filter((id, index, ids) => available.has(id) && ids.indexOf(id) === index);
  if (activeRuntimeId && available.has(activeRuntimeId) && !next.includes(activeRuntimeId))
    next.push(activeRuntimeId);
  if (next.length === 0 && runtimes[0]) next.push(runtimes[0].runtime_id);
  return next;
}

export function visibleTerminalRuntimes(
  runtimes: RuntimeSession[],
  openIds: string[],
  activeRuntimeId: string | undefined,
  splitMode: TerminalSplitMode,
): RuntimeSession[] {
  const opened = normalizeOpenRuntimeIds(openIds, runtimes, activeRuntimeId)
    .map((id) => runtimes.find((runtime) => runtime.runtime_id === id))
    .filter((runtime): runtime is RuntimeSession => Boolean(runtime));
  if (splitMode === 'single') {
    const active = opened.find((runtime) => runtime.runtime_id === activeRuntimeId);
    return active ? [active] : opened.slice(0, 1);
  }
  const visible = opened.slice(0, splitMode === 'grid' ? 4 : 2);
  // A newly selected session outside the original window must be visible.
  // Replace the last displayed view rather than changing the other pane IDs.
  const active = opened.find((runtime) => runtime.runtime_id === activeRuntimeId);
  if (active && visible.length > 0 && !visible.some((runtime) => runtime.runtime_id === activeRuntimeId)) {
    visible[visible.length - 1] = active;
  }
  return visible;
}
