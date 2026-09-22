import type {
  ComposerSessionView,
  ContextFileRef,
  ContextReadinessState,
  PromptArtifact,
} from '../../types';
import { asArray, asStringArray } from '../../lib/safeArray';

export type ComposerBriefUnknown = NonNullable<ComposerSessionView['brief']['unknowns']>[number];

export type ComposerGateAction = 'NONE' | 'PREPARE' | 'WAIT' | 'REFRESH' | 'RETRY';
export interface ComposerGate {
  canMaterialize: boolean;
  canExecute: boolean;
  action: ComposerGateAction;
  /** i18n key under work.gate.* */
  reasonKey: string;
  /** @deprecated prefer reasonKey + t() */
  reason: string;
}

const REASONS: Record<string, string> = {
  ready: 'Durable project context matches the current source fingerprint.',
  missing: 'No durable context readiness checkpoint exists yet.',
  hydrating: 'Context readiness is being evaluated.',
  stale: 'Branch, HEAD, dirty state or Maestro version changed.',
  failed: 'Context readiness could not be established.',
};

export function composerNeedsGapConfirmation(brief: {
  open_questions?: string[];
  readiness?: { state?: string };
  unknowns?: Array<{ severity?: string; status?: string }>;
}): boolean {
  if (brief.readiness?.state === 'BLOCKED') return true;

  const unknowns = brief.unknowns;
  if (Array.isArray(unknowns) && unknowns.length > 0) {
    return unknowns.some(
      (unknown) =>
        String(unknown.severity || '').toUpperCase() === 'BLOCKING' &&
        !['ANSWERED', 'CONFIRMED', 'DISMISSED'].includes(
          String(unknown.status || '').toUpperCase(),
        ),
    );
  }

  // Compatibility fallback for older payloads that did not expose structured
  // unknown severity. Current Composer views always include structured unknowns.
  return (brief.open_questions?.length ?? 0) > 0;
}

/** Prefer a readable session label over keyboard-spam or raw IDs. */
export function formatComposerSessionTitle(
  title: string | undefined,
  id: string,
  createdAt?: string,
): string {
  const raw = String(title || '').trim();
  const repeatedKeySpam = raw.length <= 24 && /^(.)\1{4,}$/u.test(raw);
  const spam = raw.length < 2 || repeatedKeySpam || /^[çc]l?k+$/iu.test(raw);
  if (!raw || spam) {
    if (createdAt) {
      const ts = Date.parse(createdAt);
      if (Number.isFinite(ts)) {
        return new Intl.DateTimeFormat(undefined, {
          day: '2-digit',
          month: 'short',
          hour: '2-digit',
          minute: '2-digit',
        }).format(ts);
      }
    }
    return id.slice(-8).toUpperCase();
  }
  return raw.length > 52 ? `${raw.slice(0, 49)}…` : raw;
}

export function composerSessionStateKey(state: string | undefined): string {
  const normalized = String(state || 'UNKNOWN').toUpperCase();
  return `work.composer.sessionState.${normalized}`;
}

export function isComposerUnknownResolved(status: string): boolean {
  return status === 'ANSWERED' || status === 'CONFIRMED' || status === 'DISMISSED';
}

/** FINALIZED sessions restore their destinations from the latest persisted artifact. */
export function latestComposerArtifact(view: ComposerSessionView | null): PromptArtifact | null {
  const artifacts = asArray<PromptArtifact>(view?.artifacts);
  return artifacts.length > 0 ? artifacts[artifacts.length - 1] : null;
}

/** Referenced files live only on the object form of brief.context. */
export function composerContextItems(
  view: ComposerSessionView | null,
  decision?: 'KEEP' | 'REFERENCE_ONLY' | 'DROP',
) {
  const items = asArray<NonNullable<ComposerSessionView['context_manifest']>['items'][number]>(
    view?.context_manifest?.items,
  );
  return decision ? items.filter((item) => item.decision === decision) : items;
}

export function composerContextBudgetPercent(view: ComposerSessionView | null): number {
  const budget = view?.context_manifest?.budget;
  const max = Number(budget?.max_estimated_tokens || 0);
  const used = Number(budget?.used_estimated_tokens || 0);
  if (!Number.isFinite(max) || max <= 0 || !Number.isFinite(used) || used <= 0) return 0;
  return Math.max(0, Math.min(100, Math.round((used / max) * 100)));
}

export function composerReferencedFiles(view: ComposerSessionView | null): ContextFileRef[] {
  const ctx = view?.brief.context;
  if (ctx && typeof ctx === 'object' && !Array.isArray(ctx)) {
    return asArray<ContextFileRef>(
      (ctx as { referenced_files?: ContextFileRef[] }).referenced_files,
    );
  }
  return [];
}

/** Open (still unresolved) unknowns, in their natural order. */
export function openComposerUnknowns(view: ComposerSessionView | null): ComposerBriefUnknown[] {
  if (!view) return [];
  return asArray<ComposerBriefUnknown>(view.brief.unknowns).filter(
    (u) => !isComposerUnknownResolved(u.status),
  );
}

/** The single unknown the conversation should surface next, if any. */
export function selectComposerNextUnknown(
  view: ComposerSessionView | null,
): ComposerBriefUnknown | null {
  const unresolved = openComposerUnknowns(view);
  const intelligenceBlocking = unresolved.find(
    (u) => u.severity === 'BLOCKING' && String(u.source || '').toUpperCase() === 'INTELLIGENCE',
  );
  if (intelligenceBlocking) return intelligenceBlocking;
  const blocking = unresolved.find((u) => u.severity === 'BLOCKING');
  const withOptions = unresolved.find((u) => asArray(u.suggested_options).length > 0);
  return blocking || withOptions || unresolved[0] || null;
}

/** Live-briefing label/value pairs, translated via the provided function. */
export function buildComposerBriefItems(
  view: ComposerSessionView,
  t: (key: string, fallback: string) => string,
): Array<[string, string]> {
  const ctx = view.brief.context;
  const existingState =
    ctx && typeof ctx === 'object' && !Array.isArray(ctx)
      ? asStringArray((ctx as { existing_state?: unknown }).existing_state)
      : [];
  const items: Array<[string, string]> = [
    [t('work.composer.understood', 'Understood'), view.brief.goal],
    [t('work.composer.context', 'Context'), [...asStringArray(ctx), ...existingState].join(' · ')],
    [t('work.composer.decisions', 'Decisions'), asStringArray(view.brief.decisions).join(' · ')],
    [
      t('work.composer.criteria', 'Criteria'),
      asStringArray(view.brief.quality?.acceptance_criteria ?? view.brief.success_criteria).join(
        ' · ',
      ),
    ],
    [
      t('work.composer.openDoubts', 'Open doubts'),
      asStringArray(view.brief.open_questions).join(' · '),
    ],
  ];
  return items.filter(([, value]) => value);
}

export type ComposerReadinessDisplay = {
  mode: 'score' | 'semantic';
  state: string;
  score?: number;
};

/** Show a percentage only when the backend supplied a real positive score. */
export function composerReadinessDisplay(
  readiness?: { score?: number; state?: string } | null,
): ComposerReadinessDisplay {
  const state = String(readiness?.state || 'UNKNOWN');
  const score = typeof readiness?.score === 'number' ? readiness.score : null;
  if (score != null && Number.isFinite(score) && score > 0) {
    return { mode: 'score', state, score: Math.max(0, Math.min(100, Math.round(score))) };
  }
  return { mode: 'semantic', state };
}

export type ComposerBriefStats = {
  decisionsDefined: number;
  decisionsPending: number;
  premisesInferred: number;
  risksIdentified: number;
  openDoubts: number;
};

export function composerBriefStats(view: ComposerSessionView | null): ComposerBriefStats {
  const unknowns = asArray<ComposerBriefUnknown>(view?.brief.unknowns);
  const open = openComposerUnknowns(view);
  const pendingDecisions = open.filter(
    (item) => String(item.status || '').toUpperCase() !== 'INFERRED',
  );
  const risks = asStringArray(view?.brief.risks);
  const questions = asStringArray(view?.brief.open_questions);
  return {
    decisionsDefined: asStringArray(view?.brief.decisions).length,
    decisionsPending: pendingDecisions.length,
    premisesInferred: unknowns.filter((item) => item.status === 'INFERRED').length,
    risksIdentified: risks.length,
    openDoubts: questions.length || pendingDecisions.length,
  };
}

export function formatComposerRelativeTime(
  iso: string | undefined,
  now = Date.now(),
  locale?: string,
): string {
  const ts = Date.parse(String(iso || ''));
  if (!Number.isFinite(ts)) return '';
  const deltaSec = Math.round((ts - now) / 1000);
  const abs = Math.abs(deltaSec);
  const rtf = new Intl.RelativeTimeFormat(locale, { numeric: 'auto' });
  if (abs < 60) return rtf.format(Math.round(deltaSec), 'second');
  if (abs < 3600) return rtf.format(Math.round(deltaSec / 60), 'minute');
  if (abs < 86400) return rtf.format(Math.round(deltaSec / 3600), 'hour');
  return rtf.format(Math.round(deltaSec / 86400), 'day');
}

export function composerExpectedOutcome(view: ComposerSessionView | null): string {
  const dod = asStringArray(view?.brief.quality?.definition_of_done);
  if (dod.length) return dod.join(' · ');
  return asStringArray(
    view?.brief.quality?.acceptance_criteria ?? view?.brief.success_criteria,
  ).join(' · ');
}

export function composerScopeLines(view: ComposerSessionView | null): string[] {
  const ctx = view?.brief.context;
  if (Array.isArray(ctx)) return asStringArray(ctx);
  if (ctx && typeof ctx === 'object') {
    return [
      ...asStringArray((ctx as { existing_state?: unknown }).existing_state),
      ...asStringArray((ctx as { evidence?: unknown }).evidence),
    ];
  }
  return [];
}

export type ComposerActionPhase = 'elaborating' | 'ready_review' | 'finalized';

export function composerActionPhase(
  view: ComposerSessionView | null,
  hasArtifact: boolean,
): ComposerActionPhase {
  if (!view) return 'elaborating';
  if (view.session.state === 'FINALIZED' || hasArtifact) return 'finalized';
  const state = String(view.brief.readiness?.state || view.session.state || '').toUpperCase();
  if (state === 'READY' || state === 'READY_WITH_ASSUMPTIONS' || state === 'READY_WITH_GAPS') {
    return 'ready_review';
  }
  return 'elaborating';
}

export function composerConstraintLines(view: ComposerSessionView | null): string[] {
  const raw = view?.brief.constraints;
  if (Array.isArray(raw)) return asStringArray(raw);
  if (raw && typeof raw === 'object') {
    return Object.values(raw).flatMap((value) =>
      Array.isArray(value) ? asStringArray(value) : typeof value === 'string' ? [value] : [],
    );
  }
  return [];
}

export function composerGateForReadiness(state: ContextReadinessState): ComposerGate {
  switch (state) {
    case 'READY':
      return {
        canMaterialize: true,
        canExecute: true,
        action: 'NONE',
        reasonKey: 'work.gate.ready',
        reason: REASONS.ready,
      };
    case 'MISSING':
      return {
        canMaterialize: false,
        canExecute: false,
        action: 'PREPARE',
        reasonKey: 'work.gate.missing',
        reason: REASONS.missing,
      };
    case 'HYDRATING':
      return {
        canMaterialize: false,
        canExecute: false,
        action: 'WAIT',
        reasonKey: 'work.gate.hydrating',
        reason: REASONS.hydrating,
      };
    case 'STALE':
      // Fingerprint drifted (dirty git, branch, Maestro). Context still exists —
      // locking Send/Flow here leaves destinations dead for the whole coding session.
      return {
        canMaterialize: true,
        canExecute: true,
        action: 'REFRESH',
        reasonKey: 'work.gate.stale',
        reason: REASONS.stale,
      };
    case 'FAILED':
    default:
      return {
        canMaterialize: false,
        canExecute: false,
        action: 'RETRY',
        reasonKey: 'work.gate.failed',
        reason: REASONS.failed,
      };
  }
}
