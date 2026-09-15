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
import { selectResumableComposerSession } from './composerSessionModel';
import {
  buildComposerBriefItems,
  composerNeedsGapConfirmation,
  composerSessionStateKey,
  formatComposerSessionTitle,
  latestComposerArtifact,
  openComposerUnknowns,
  selectComposerNextUnknown,
} from './composerModel';
import type { ComposerStartCreateArgs } from './ComposerStart';

/**
 * Owns all Composer elaboration state, API orchestration and derived view
 * models. Keeping this in a colocated hook lets ComposerSurface stay a thin
 * presentational orchestrator that only wires the decomposed subcomponents.
 */
export function useComposerSession(project: Project) {
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

  /** Shared busy/error envelope so handlers stay declarative. */
  const runBusy = useCallback(async (fn: () => Promise<void>) => {
    setBusy(true);
    setError('');
    try {
      await fn();
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
          setArtifact(latestComposerArtifact(loaded));
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

  const briefItems = useMemo(() => (view ? buildComposerBriefItems(view, t) : []), [t, view]);
  const nextUnknown = useMemo(() => selectComposerNextUnknown(view), [view]);
  const actionableGaps = useMemo(() => openComposerUnknowns(view), [view]);
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

  const loadSession = (id: string) =>
    runBusy(async () => {
      const loaded = await nexus.getComposerSession(id);
      setView(loaded);
      setArtifact(latestComposerArtifact(loaded));
    });

  const create = ({ mode, draft, sourcePrompt }: ComposerStartCreateArgs) =>
    runBusy(async () => {
      if (!draft.trim()) return;
      const next = await nexus.createComposerSessionWithMode(
        project.id,
        draft.trim(),
        mode,
        mode === 'EXISTING_PROMPT' ? sourcePrompt.trim() : undefined,
      );
      setView(next);
      setArtifact(null);
      await refreshSessions();
    });

  const send = () =>
    runBusy(async () => {
      if (!view || !message.trim()) return;
      setView(await nexus.addComposerTurn(view.session.id, message.trim(), view.session.revision));
      setMessage('');
      await refreshSessions();
    });

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

  const refine = () =>
    runBusy(async () => {
      if (!view) return;
      setArtifact(await nexus.refineComposerArtifact(view.session.id, refineText.trim()));
      setShowRefineInput(false);
      setRefineText('');
      setView(await nexus.getComposerSession(view.session.id));
      await refreshSessions();
    });

  const resolveUnknown = (unknownId: string, status: string, answerOverride?: string) =>
    runBusy(async () => {
      if (!view) return;
      setView(
        await nexus.resolveComposerUnknown(
          view.session.id,
          unknownId,
          answerOverride ?? '',
          status,
          view.session.revision,
        ),
      );
      await refreshSessions();
    });

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

  const updateSkill = (skillId: string, state: 'ACCEPTED' | 'REJECTED') =>
    runBusy(async () => {
      if (!view) return;
      setView(await nexus.updateComposerSkillState(view.session.id, skillId, state));
    });

  const attachContext = (path: string) =>
    runBusy(async () => {
      if (!view) return;
      setView(await nexus.attachComposerContext(view.session.id, path, view.session.revision));
      await refreshSessions();
    });

  const detachContext = (path: string) =>
    runBusy(async () => {
      if (!view) return;
      setView(await nexus.detachComposerContext(view.session.id, path, view.session.revision));
      await refreshSessions();
    });

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

  const resetSession = () => {
    setView(null);
    setArtifact(null);
  };

  return {
    sessions,
    view,
    artifact,
    error,
    busy,
    confirmGaps,
    message,
    refineText,
    showRefineInput,
    agents,
    selectedAgentId,
    briefItems,
    nextUnknown,
    actionableGaps,
    sessionOptions,
    setMessage,
    setRefineText,
    setShowRefineInput,
    setSelectedAgentId,
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
    resetSession,
  };
}
