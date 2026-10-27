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
  const conversationAbort = useRef<AbortController | null>(null);
  const skillCatalogRequest = useRef(0);

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

  useEffect(
    () => () => {
      conversationAbort.current?.abort();
    },
    [],
  );

  useEffect(() => {
    void (async () => {
      try {
        const next = await refreshSessions();
        const focusKey = `nexus:composer:focus:${project.id}`;
        let requestedId = '';
        try {
          requestedId = window.sessionStorage.getItem(focusKey) || '';
          if (requestedId) window.sessionStorage.removeItem(focusKey);
        } catch {
          // Browser storage is optional; regular resume remains available.
        }
        const id = requestedId || selectResumableComposerSession(next);
        if (id) {
          const loaded = await nexus.getComposerSession(id);
          if (loaded.session.project_id !== project.id) {
            throw new Error('A elaboração solicitada pertence a outro projeto.');
          }
          setView(loaded);
          setArtifact(latestComposerArtifact(loaded));
        }
      } catch (err) {
        setError(mapNetworkError(err, t));
      }
    })();
  }, [project.id, refreshSessions, t]);

  // Work surfaces may remain mounted in a hidden workspace tab. The event
  // complements sessionStorage so a review created by Flow opens immediately
  // without depending on an unmount/remount cycle.
  useEffect(() => {
    const focus = (event: Event) => {
      const detail = (event as CustomEvent<{ projectId: string; sessionId: string }>).detail;
      if (!detail || detail.projectId !== project.id || !detail.sessionId) return;
      void nexus
        .getComposerSession(detail.sessionId)
        .then((loaded) => {
          if (loaded.session.project_id !== project.id) {
            setError('A revisão pertence a outro projeto.');
            return;
          }
          setView(loaded);
          setArtifact(latestComposerArtifact(loaded));
          setError('');
          try {
            window.sessionStorage.removeItem(`nexus:composer:focus:${project.id}`);
          } catch {
            // Optional browser storage.
          }
        })
        .catch((reason: unknown) => {
          setError(mapNetworkError(reason, t));
        });
    };
    window.addEventListener('nexus:composer:focus', focus);
    return () => window.removeEventListener('nexus:composer:focus', focus);
  }, [project.id, t]);

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
    const request = ++skillCatalogRequest.current;
    void nexus
      .listComposerSkillCatalog(project.id)
      .then((items) => {
        if (request === skillCatalogRequest.current) setSkillCatalog(items || []);
      })
      .catch(() => {
        if (request === skillCatalogRequest.current) setSkillCatalog([]);
      });
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

  const deleteSession = (sessionId: string, expectedRevision: number) =>
    runBusy(async () => {
      await nexus.deleteComposerSession(sessionId, expectedRevision);
      if (view?.session.id === sessionId) {
        conversationAbort.current?.abort();
        setView(null);
        setArtifact(null);
        setBootstrapActivity(null);
      }
      await refreshSessions();
    });

  const create = ({ mode, interviewMode, draft, sourcePrompt }: ComposerStartCreateArgs) =>
    runBusy(
      async () => {
        if (!draft.trim()) return;
        conversationAbort.current?.abort();
        const controller = new AbortController();
        conversationAbort.current = controller;
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
              interview_mode: interviewMode,
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
              interviewMode,
              controller.signal,
            );
            setView(next);
            setArtifact(null);
            setBootstrapActivity(null);
            await refreshSessions();
            return;
          }
          throw err;
        } finally {
          // A failed provider probe or an interrupted stream must never leave
          // the startup activity projected as "pending". This block also
          // runs when fallback JSON creation fails.
          setBootstrapActivity(null);
          if (conversationAbort.current === controller) conversationAbort.current = null;
        }
      },
      { conversation: true },
    );

  const cancelConversation = () => {
    conversationAbort.current?.abort();
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
        conversationAbort.current?.abort();
        const controller = new AbortController();
        conversationAbort.current = controller;
        try {
          setView(
            await nexus.addComposerTurn(
              view.session.id,
              text.trim(),
              view.session.revision,
              attachments,
              controller.signal,
            ),
          );
          setMessage('');
          await refreshSessions();
        } finally {
          if (conversationAbort.current === controller) conversationAbort.current = null;
        }
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

  const searchSkillCatalog = useCallback(
    async (query: string) => {
      const request = ++skillCatalogRequest.current;
      try {
        const items = await nexus.listComposerSkillCatalog(project.id, query);
        if (request === skillCatalogRequest.current) setSkillCatalog(items || []);
      } catch {
        if (request === skillCatalogRequest.current) setSkillCatalog([]);
      }
    },
    [project.id],
  );

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
    cancelConversation();
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
    cancelConversation,
    send,
    sendText,
    finalize,
    refine,
    resolveUnknown,
    rememberUnknown,
    updateSkill,
    searchSkillCatalog,
    selectCatalogSkill,
    attachContext,
    detachContext,
    copy,
    sendToAgent,
    loadSession,
    deleteSession,
    resetSession,
  };
}
