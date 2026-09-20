import type { ClarificationCheckpoint, WorkPlan } from '../../types';
import type { FlowStepModel } from './flowModel';

export const GENERATE_BUDGET_SEC = 90;

export type StudioView = 'pipeline' | 'graph';
export type FlowStepAction = 'REFINE' | 'EXPAND' | 'COMPARE' | 'SPLIT' | 'REMOVE';

export const composerDraftStorageKey = (projectId: string): string =>
  `nexus:composer:draft:${projectId}`;
export const composerHistoryStorageKey = (projectId: string): string =>
  `nexus:composer:history:${projectId}`;

/** Session key for the unsaved Flow step-draft of one project. Survives surface
 *  unmounts (product-surface isolation) within the tab session; the server plan
 *  remains the source of truth for anything persisted. */
export const flowDraftSessionKey = (projectId: string): string =>
  `nexus:flow:draft-edits:${projectId}`;

export interface StoredFlowDraft {
  planId: string;
  revision: number;
  draft: unknown;
}

/** Reads the unsaved step-draft, tolerating unavailable/broken storage. */
export function loadStoredFlowDraft(key: string): StoredFlowDraft | null {
  try {
    const raw = window.sessionStorage.getItem(key);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as StoredFlowDraft;
    if (!parsed || typeof parsed.planId !== 'string' || !parsed.draft) return null;
    return parsed;
  } catch {
    return null;
  }
}

/** Persists (or clears with null) the unsaved step-draft; best effort. */
export function persistStoredFlowDraft(key: string, value: StoredFlowDraft | null): void {
  try {
    if (!value) window.sessionStorage.removeItem(key);
    else window.sessionStorage.setItem(key, JSON.stringify(value));
  } catch {
    /* best effort persistence */
  }
}

/** Reads the persisted goal draft, tolerating unavailable/broken storage. */
export function loadStoredGoal(key: string): string {
  try {
    return window.localStorage.getItem(key) || '';
  } catch {
    return '';
  }
}

/** Persists (or clears when empty) the goal draft; failures are best effort. */
export function persistStoredGoal(key: string, value: string): void {
  try {
    if (value) window.localStorage.setItem(key, value);
    else window.localStorage.removeItem(key);
  } catch {
    /* best effort persistence */
  }
}

/** Reads the recent-request history, always returning a safe string array. */
export function loadGoalHistory(key: string): string[] {
  try {
    const parsed = JSON.parse(window.localStorage.getItem(key) || '[]');
    return Array.isArray(parsed)
      ? parsed.filter((item): item is string => typeof item === 'string')
      : [];
  } catch {
    return [];
  }
}

/** Prepends a goal to the deduplicated history (max 8) and persists it. */
export function recordGoalHistory(key: string, goal: string): string[] {
  const trimmed = goal.trim();
  const history = [trimmed, ...loadGoalHistory(key)]
    .filter((item, index, all) => item && all.indexOf(item) === index)
    .slice(0, 8);
  try {
    window.localStorage.setItem(key, JSON.stringify(history));
  } catch {
    /* best effort persistence */
  }
  return history;
}

const truncateGoal = (goal: string): string => (goal.length > 72 ? `${goal.slice(0, 69)}…` : goal);

/** Derives up to 8 unique chips (goal, title, phases, packages) from a plan. */
export function chipsFromPlan(plan: WorkPlan, goalText: string): string[] {
  const chips: string[] = [];
  const goal = goalText.trim();
  if (goal) chips.push(truncateGoal(goal));
  if (plan.title && plan.title !== goal) chips.push(plan.title);
  for (const phase of plan.phases || []) {
    if (phase.title) chips.push(phase.title);
    for (const pkg of phase.packages || []) if (pkg.title) chips.push(pkg.title);
  }
  return [...new Set(chips)].slice(0, 8);
}

/** Seeds clarification answers from any pre-filled checkpoint unknowns. */
export function clarificationAnswersFromCheckpoint(
  checkpoint: ClarificationCheckpoint,
): Record<string, string> {
  return Object.fromEntries(
    checkpoint.unknowns.filter((item) => item.answer).map((item) => [item.key, item.answer || '']),
  );
}

/** De-duplicates, trims and drops empty entries from a string list. */
export function uniqueTrimmed(items: string[]): string[] {
  return [...new Set(items.map((item) => item.trim()).filter(Boolean))];
}

/** Builds the local-only "refine" patch that tidies a single step in place. */
export function refineStepPatch(step: FlowStepModel): Partial<FlowStepModel> {
  const skills = uniqueTrimmed(step.skillIds.length ? step.skillIds : step.maestroSkills);
  return {
    title: step.title.trim(),
    goal: step.goal.trim(),
    acceptanceCriteria: uniqueTrimmed(step.acceptanceCriteria),
    relevantPaths: uniqueTrimmed(step.relevantPaths),
    skillIds: skills,
    maestroSkills: skills,
    verificationRequirements: uniqueTrimmed(step.verificationRequirements),
  };
}

/** Deterministic child ids for splitting a step into two local branches. */
export function splitStepIds(stepId: string, stamp: string): [string, string] {
  return [`${stepId}-a-${stamp}`, `${stepId}-b-${stamp}`];
}
