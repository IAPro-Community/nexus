import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { getCSRFToken } from '../../api';
import { nexus } from '../../nexus/api';
import type {
  Agent,
  ComposerSession,
  ComposerSessionView,
  ComposerSkillCatalogItem,
  Project,
  PromptArtifact,
} from '../../types';
import {
  composerTurnsToEvents,
  type NexusActivityItem,
  type NexusActivityPayload,
  type NexusConversationEvent,
} from '../conversation';
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
import {
  bootstrapStepsToActivity,
  createComposerSessionWithProgress,
  mapNetworkError,
} from './composerBootstrapStream';

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
  /** Only true for chat-shaped work (create/send). Never for resolve/attach — that freezes assistant-ui. */
  const [conversationRunning, setConversationRunning] = useState(false);
  const [refineText, setRefineText] = useState('');
  const [showRefineInput, setShowRefineInput] = useState(false);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [skillCatalog, setSkillCatalog] = useState<ComposerSkillCatalogItem[]>([]);
  const [selectedAgentId, setSelectedAgentId] = useState('');
  const [bootstrapActivity, setBootstrapActivity] = useState<NexusActivityPayload | null>(null);
  const createAbort = useRef<AbortController | null>(null);

  const selectedSkillIds = useMemo(
    () =>
      (view?.skills || [])
        .filter((skill) => skill.state === 'ACCEPTED' || skill.state === 'APPLIED')
        .map((skill) => skill.skill_id),
    [view?.skills],
  );

  const conversationEvents: NexusConversationEvent[] = useMemo(() => {
    if (!view) return [];
    return composerTurnsToEvents(view.session.id, view.turns || []);
  }, [view]);

  const refreshSessions = useCallback(async () => {
    const next = await nexus.listComposerSessions(project.id);
    setSessions(next || []);
    return next || [];
  }, [project.id]);

  const runBusy = useCallback(
    async (fn: () => Promise<void>, opts?: { conversation?: boolean }) => {
      setBusy(true);
      if (opts?.conversation) setConversationRunning(true);
      setError('');
      try {
        await fn();
      } catch (err) {
        setError(mapNetworkError(err, t));
      } finally {
        if (opts?.conversation) setConversationRunning(false);
        setBusy(false);
      }
    },
    [t],
  );

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
        setError(mapNetworkError(err, t));
      }
    })();
  }, [project.id, refreshSessions, t]);

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

  useEffect(() => {
    void nexus
      .listComposerSkillCatalog(project.id)
      .then((items) => setSkillCatalog(items || []))
      .catch(() => setSkillCatalog([]));
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
    runBusy(
      async () => {
        if (!draft.trim()) return;
        createAbort.current?.abort();
        const controller = new AbortController();
        createAbort.current = controller;
        const steps = new Map<string, NexusActivityItem['status']>();
        const refreshActivity = () =>
          setBootstrapActivity(
            bootstrapStepsToActivity(steps, t('work.composer.starting', 'Starting…')),
          );
        refreshActivity();
        try {
          const next = await createComposerSessionWithProgress(
            project.id,
            {
              goal: draft.trim(),
              input_mode: mode,
              source_prompt: mode === 'EXISTING_PROMPT' ? sourcePrompt.trim() : '',
            },
            {
              signal: controller.signal,
              csrfToken: getCSRFToken(),
              onEvent: (event) => {
                if (event.type === 'step') {
                  const status =
                    event.status === 'started'
                      ? 'running'
                      : event.status === 'done'
                        ? 'done'
                        : event.status === 'error'
                          ? 'error'
                          : 'skipped';
                  steps.set(event.id, status);
                  refreshActivity();
                }
              },
            },
          );
          setView(next);
          setArtifact(null);
          setBootstrapActivity(null);
          await refreshSessions();
        } catch (err) {
          const message = err instanceof Error ? err.message : String(err);
          if (/empty progress stream|unsupported|NDJSON|405|406/i.test(message)) {
            const next = await nexus.createComposerSessionWithMode(
              project.id,
              draft.trim(),
              mode,
              mode === 'EXISTING_PROMPT' ? sourcePrompt.trim() : undefined,
            );
            setView(next);
            setArtifact(null);
            setBootstrapActivity(null);
            await refreshSessions();
            return;
          }
          throw err;
        } finally {
          if (createAbort.current === controller) createAbort.current = null;
        }
      },
      { conversation: true },
    );

  const cancelCreate = () => {
    createAbort.current?.abort();
  };

  const sendText = (
    text: string,
    attachments?: Array<{
      id: string;
      name?: string;
      mimeType?: string;
      url?: string;
      size?: number;
    }>,
  ) =>
    runBusy(
      async () => {
        if (!view) return;
        if (!text.trim() && !(attachments && attachments.length)) return;
        setView(
          await nexus.addComposerTurn(
            view.session.id,
            text.trim(),
            view.session.revision,
            attachments,
          ),
        );
        setMessage('');
        await refreshSessions();
      },
      { conversation: true },
    );

  const send = () => sendText(message);

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
      const detail = mapNetworkError(err, t);
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
      setError(mapNetworkError(err, t));
    }
    await resolveUnknown(unknown.id, 'ANSWERED', answer);
  };

  const updateSkill = (skillId: string, state: 'ACCEPTED' | 'REJECTED') =>
    runBusy(async () => {
      if (!view) return;
      setView(await nexus.updateComposerSkillState(view.session.id, skillId, state));
    });

  const selectCatalogSkill = (skillId: string) =>
    runBusy(async () => {
      if (!view) return;
      setView(await nexus.selectComposerSkill(view.session.id, skillId));
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
    setBootstrapActivity(null);
  };

  return {
    sessions,
    view,
    artifact,
    error,
    busy,
    conversationRunning,
    confirmGaps,
    message,
    refineText,
    showRefineInput,
    agents,
    skillCatalog,
    selectedAgentId,
    briefItems,
    nextUnknown,
    actionableGaps,
    sessionOptions,
    conversationEvents,
    bootstrapActivity,
    setMessage,
    setRefineText,
    setShowRefineInput,
    setSelectedAgentId,
    create,
    cancelCreate,
    send,
    sendText,
    finalize,
    refine,
    resolveUnknown,
    rememberUnknown,
    updateSkill,
    selectCatalogSkill,
    attachContext,
    detachContext,
    copy,
    sendToAgent,
    loadSession,
    resetSession,
  };
}
