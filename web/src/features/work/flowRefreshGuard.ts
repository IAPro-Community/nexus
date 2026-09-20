/**
 * Latest-wins guard for overlapping async refreshes (e.g. FlowRunSurface
 * manual refresh racing the polling tick). A slow earlier request must never
 * overwrite fresher state that arrived later.
 */
export interface RefreshGuard {
  /** Starts a new request and returns its sequence id. */
  next(): number;
  /** Returns true only if no newer request started after `id`. */
  isLatest(id: number): boolean;
}

export function createRefreshGuard(): RefreshGuard {
  let seq = 0;
  return {
    next(): number {
      seq += 1;
      return seq;
    },
    isLatest(id: number): boolean {
      return id === seq;
    },
  };
}
