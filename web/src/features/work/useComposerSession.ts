import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { nexus } from '../../nexus/api';
import type {
  Agent,
  ComposerSession,
  ComposerSessionView,
  Project,
  PromptArtifact,
} from '../../types';
import { asArray, asStringArray } from '../../lib/safeArray';
import { selectResumableComposerSession } from './composerSessionModel';
import {
  composerNeedsGapConfirmation,
  composerSessionStateKey,
  formatComposerSessionTitle,
} from './composerModel';
import type { ComposerGate } from './composerModel';
import type { ComposerStartCreateArgs } from './ComposerStart';

export type BriefUnknown = NonNullable<ComposerSessionView['brief']['unknowns']>[number];

function isUnknownResolved(status: string): boolean {
  return status === 'ANSWERED' || status === 'CONFIRMED' || status === 'DISMISSED';
}

/** FINALIZED sessions restore their destinations from the latest persisted artifact. */
function latestArtifactOf(view: ComposerSessionView | null): PromptArtifact | null {
  const artifacts = asArray<PromptArtifact>(view?.artifacts);
  return artifacts.length > 0 ? artifacts[artifacts.length - 1] : null;
}

/**
 * Owns all Composer elaboration state, API orchestration and derived view
 * models. Keeping this in a colocated hook lets ComposerSurface stay a thin
 * presentational orchestrator that only wires the decomposed subcomponents.
 */
export function useComposerSession({
  project,
  destinationGate,
}: {
  project: Project;
  destinationGate?: ComposerGate;
}) {
  const { t } = useTranslation();
  const [sessions, setSessions] = useState<ComposerSession[]>([]);
  const [view, setView] = useState<ComposerSessionView | null>(null);
  const [message, setMessage] = useState('');
  const [artifact, setArtifact] = useState<PromptArtifact | null>(null);
  const [error, setError] = useState('');
  const [confirmGaps, setConfirmGaps] = useState(false);
  const [busy, setBusy] = useState(false);
  const [refineText, setRefineText] = useState('');
  const [showRefineInput, setShowRefineInput] = useState(false);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [selectedAgentId, setSelectedAgentId] = useState('');
  const canMaterialize = destinationGate?.canMaterialize ?? true;
  const canExecute = destinationGate?.canExecute ?? true;

  const selectedSkillIds = useMemo(
    () =>
      (view?.skills || [])
        .filter((skill) => skill.state === 'ACCEPTED' || skill.state === 'APPLIED')
        .map((skill) => skill.skill_id),
    [view?.skills],
  );

  const refreshSessions = useCallback(async () => {
    const next = await nexus.listComposerSessions(project.id);
    setSessions(next || []);
    return next || [];
  }, [project.id]);

  const loadSession = useCallback(async (id: string) => {
    setBusy(true);
    setError('');
    try {
      const loaded = await nexus.getComposerSession(id);
      setView(loaded);
      setArtifact(latestArtifactOf(loaded));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }, []);

  useEffect(() => {
    void (async () => {
      try {
        const next = await refreshSessions();
        const id = selectResumableComposerSession(next);
        if (id) {
          const loaded = await nexus.getComposerSession(id);
          setView(loaded);
          setArtifact(latestArtifactOf(loaded));
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      }
    })();
  }, [project.id, refreshSessions]);

  useEffect(() => {
    void nexus
      .listAgents(project.id)
      .then((items) => {
        const next = items || [];
        setAgents(next);
        setSelectedAgentId((current) => current || next[0]?.id || '');
      })
      .catch(() => undefined);
  }, [project.id]);

  const briefItems = useMemo<Array<[string, string]>>(() => {
    if (!view) return [];
    const items: Array<[string, string]> = [
      [t('work.composer.understood', 'Understood'), view.brief.goal],
      [
        t('work.composer.context', 'Context'),
        [
          ...asStringArray(view.brief.context),
          ...(view.brief.context &&
          typeof view.brief.context === 'object' &&
          !Array.isArray(view.brief.context)
            ? asStringArray((view.brief.context as { existing_state?: unknown }).existing_state)
            : []),
        ].join(' · '),
      ],
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
    return items.filter(([, value]) => Boolean(value));
  }, [t, view]);

  const nextUnknown = useMemo<BriefUnknown | null>(() => {
    if (!view) return null;
    const unknowns = asArray<BriefUnknown>(view.brief.unknowns);
    const unresolved = unknowns.filter((u) => !isUnknownResolved(u.status));
    const intelligenceBlocking = unresolved.find(
      (u) => u.severity === 'BLOCKING' && String(u.source || '').toUpperCase() === 'INTELLIGENCE',
    );
    if (intelligenceBlocking) return intelligenceBlocking;
    const blocking = unresolved.find((u) => u.severity === 'BLOCKING');
    const withOptions = unresolved.find((u) => asArray(u.suggested_options).length > 0);
    return blocking || withOptions || unresolved[0] || null;
  }, [view]);

  const actionableGaps = useMemo<BriefUnknown[]>(() => {
    if (!view) return [];
    return asArray<BriefUnknown>(view.brief.unknowns).filter((u) => !isUnknownResolved(u.status));
  }, [view]);

  const sessionOptions = useMemo(
    () =>
      sessions.map((s) => ({
        value: s.id,
        label: `${formatComposerSessionTitle(s.title, s.id, s.created_at)} · ${t(
          composerSessionStateKey(s.state),
          s.state,
        )}`,
      })),
    [sessions, t],
  );

  const create = async ({ mode, draft, sourcePrompt }: ComposerStartCreateArgs) => {
    if (!draft) return;
    setBusy(true);
    setError('');
    try {
      const next = await nexus.createComposerSessionWithMode(
        project.id,
        draft,
        mode,
        mode === 'EXISTING_PROMPT' ? sourcePrompt : undefined,
      );
      setView(next);
      setArtifact(null);
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const send = async () => {
    if (!view || !message.trim() || busy) return;
    const textToSend = message.trim();
    setBusy(true);
    setError('');
    try {
      setView(await nexus.addComposerTurn(view.session.id, textToSend, view.session.revision));
      setMessage('');
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const finalize = async (confirmed = false) => {
    if (!view) return;
    if (!confirmed && composerNeedsGapConfirmation(view.brief)) {
      setConfirmGaps(true);
      setError(t('work.confirmGaps'));
      return;
    }
    setBusy(true);
    setError('');
    try {
      const res = await nexus.finalizeComposerSession(view.session.id, selectedSkillIds, confirmed);
      setArtifact(res);
      setView(await nexus.getComposerSession(view.session.id));
      await refreshSessions();
    } catch (err) {
      const detail = err instanceof Error ? err.message : String(err);
      setError(detail);
      setConfirmGaps(detail.includes('open questions') || detail.includes('gaps'));
    } finally {
      setBusy(false);
    }
  };

  const refine = async () => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      const newArtifact = await nexus.refineComposerArtifact(view.session.id, refineText.trim());
      setArtifact(newArtifact);
      setShowRefineInput(false);
      setRefineText('');
      setView(await nexus.getComposerSession(view.session.id));
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const resolveUnknown = async (unknownId: string, status: string, answerOverride?: string) => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      const updated = await nexus.resolveComposerUnknown(
        view.session.id,
        unknownId,
        answerOverride ?? '',
        status,
        view.session.revision,
      );
      setView(updated);
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const rememberUnknown = async (unknown: { id: string; field?: string }, answer: string) => {
    if (!view) return;
    try {
      await nexus.upsertIntelligenceMemory({
        scope: 'PROJECT',
        project_id: project.id,
        key: unknown.field || unknown.id,
        value: answer,
        source: 'explicit',
        force: true,
      });
    } catch (err) {
      // Non-promotable keys can't become memories, but the answer below must
      // still resolve the unknown instead of stranding it.
      setError(err instanceof Error ? err.message : String(err));
    }
    await resolveUnknown(unknown.id, 'ANSWERED', answer);
  };

  const updateSkill = async (skillId: string, state: 'ACCEPTED' | 'REJECTED') => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      setView(await nexus.updateComposerSkillState(view.session.id, skillId, state));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const attachContext = async (path: string) => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      setView(await nexus.attachComposerContext(view.session.id, path, view.session.revision));
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const detachContext = async (path: string) => {
    if (!view) return;
    setBusy(true);
    setError('');
    try {
      setView(await nexus.detachComposerContext(view.session.id, path, view.session.revision));
      await refreshSessions();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const copy = async () => {
    if (!artifact) return;
    try {
      await navigator.clipboard.writeText(artifact.content);
    } catch {
      setError(t('work.composer.copyFailed', 'Could not copy the prompt in this browser.'));
    }
  };

  const sendToAgent = async (
    prompt: string,
    skills: string[],
    startIfNeeded: boolean,
    contextFingerprintId: string,
  ) => {
    await nexus.askAgent(selectedAgentId, prompt, startIfNeeded, skills, {
      projectId: project.id,
      contextFingerprintId,
    });
    setMessage(t('work.composer.sentToAgent', 'Task sent to Agent.'));
  };

  const reset = () => {
    setView(null);
    setArtifact(null);
  };

  return {
    sessions,
    view,
    message,
    setMessage,
    artifact,
    error,
    confirmGaps,
    busy,
    refineText,
    setRefineText,
    showRefineInput,
    setShowRefineInput,
    agents,
    selectedAgentId,
    setSelectedAgentId,
    canMaterialize,
    canExecute,
    briefItems,
    nextUnknown,
    nextQuestion: nextUnknown?.question || '',
    actionableGaps,
    sessionOptions,
    create,
    send,
    finalize,
    refine,
    resolveUnknown,
    rememberUnknown,
    updateSkill,
    attachContext,
    detachContext,
    copy,
    sendToAgent,
    loadSession,
    reset,
  };
}
