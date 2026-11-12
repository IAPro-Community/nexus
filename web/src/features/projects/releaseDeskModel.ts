import type { ReleaseDeskReport } from '../../types';

/**
 * Pure presentation helpers for the Release Desk card. The desk never invents
 * authorization: a gate is only "verified" when the backend says so, and the
 * merge/publish verdict is surfaced exactly as reported.
 */

export function shortSha(sha: string): string {
  return sha.slice(0, 7);
}

export interface ReleaseSummary {
  totalGates: number;
  verifiedGates: number;
  commitCount: number;
  prCount: number;
  issueCount: number;
  mergePublishAllowed: boolean;
  hasApproval: boolean;
}

export function summarizeRelease(report: ReleaseDeskReport): ReleaseSummary {
  const gates = report.gates || [];
  return {
    totalGates: gates.length,
    verifiedGates: gates.filter((gate) => gate.status === 'verified').length,
    commitCount: (report.commits || []).length,
    prCount: (report.pull_requests || []).length,
    issueCount: (report.issues || []).length,
    mergePublishAllowed: report.merge_publish_allowed === true,
    hasApproval: Boolean(report.approval),
  };
}

export function gateTone(status: string): 'success' | 'warning' {
  return status === 'verified' ? 'success' : 'warning';
}

export function deskTone(allowed: boolean): 'success' | 'danger' {
  return allowed ? 'success' : 'danger';
}

/**
 * Formats an approval timestamp with Intl (project rule: no hardcoded date
 * rendering). Falls back to the raw value when the input is unparsable so the
 * desk never hides the evidence it was given.
 */
export function formatApprovalDate(isoDate: string, locale: string): string {
  const parsed = new Date(isoDate);
  if (Number.isNaN(parsed.getTime())) {
    return isoDate;
  }
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(
    parsed,
  );
}
