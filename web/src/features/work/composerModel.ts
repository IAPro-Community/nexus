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
}): boolean {
  return brief.readiness?.state === 'BLOCKED' || (brief.open_questions?.length ?? 0) > 0;
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
      return {
        canMaterialize: false,
        canExecute: false,
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
