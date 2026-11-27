import { asArray } from '../../lib/safeArray';

/** Row shape returned by `GET /api/v1/projects/prune-candidates`. */
export interface PruneCandidate {
  id: string;
  name: string;
  canonical_path: string;
  last_opened_at?: string | null;
  agent_count: number;
  layout_saved: boolean;
  event_count: number;
  /** Empty when the row keeps evidence and must not be offered for removal. */
  reason: string;
}

/**
 * Keep only rows the CLI would accept as prune candidates: no agents, no saved
 * layout, no events and never opened from the UI. Go may encode the arrays as
 * null, so the input always goes through asArray.
 */
export function pruneCandidates(rows: unknown): PruneCandidate[] {
  return asArray<PruneCandidate>(rows).filter(
    (row) =>
      !!row &&
      typeof row.id === 'string' &&
      typeof row.reason === 'string' &&
      row.reason !== '' &&
      Number(row.agent_count || 0) === 0 &&
      !row.layout_saved &&
      Number(row.event_count || 0) === 0 &&
      !row.last_opened_at,
  );
}

/** Short, translated-safe explanation for a row that stays (or goes). */
export function pruneReasonLabel(row: PruneCandidate): string {
  if (row.reason) return row.reason;
  return [
    `kept: ${Number(row.agent_count || 0)} agent(s)`,
    `layout=${row.layout_saved === true}`,
    `${Number(row.event_count || 0)} event(s)`,
    `opened=${Boolean(row.last_opened_at)}`,
  ].join(', ');
}
