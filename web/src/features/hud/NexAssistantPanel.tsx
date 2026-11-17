import React from 'react';
import { MessageCircle, ChevronDown, ChevronUp } from 'lucide-react';
import { nexus } from '../../nexus/api';
import type {
  NexAssistantOverview,
  NexAssistantSuggestion,
  NexChatTurn,
} from './nexAssistantTypes';
import {
  readNexAssistantPreferences,
  saveNexAssistantPreferences,
} from './nexAssistantPreferences';
import { NexProactivityControls } from './NexProactivityControls';
import { NexAssistantActions } from './NexAssistantActions';
import styles from './NexAssistantPanel.module.scss';

export interface NexAssistantPanelProps {
  projectId?: string;
  revision?: string;
  privacyMode?: boolean;
  onOpenMissions?: () => void;
  onOpenAgents?: () => void;
  onOpenUsage?: () => void;
  onOpenNexus?: () => void;
}

/**
 * Separate from HUD operational presentation: conversations never mutate
 * missions, capacity, agent state, or the character's canonical mood.
 * Persisted history is opt-in and stored by Nexus Core in SQLite, never in
 * the browser localStorage or Maestro's semantic project memory.
 */
export const NexAssistantPanel: React.FC<NexAssistantPanelProps> = ({
  projectId,
  revision,
  privacyMode = false,
  onOpenMissions,
  onOpenAgents,
  onOpenUsage,
  onOpenNexus,
}) => {
  const [expanded, setExpanded] = React.useState(false);
  const [overview, setOverview] = React.useState<NexAssistantOverview | null>(null);
  const [turns, setTurns] = React.useState<NexChatTurn[]>([]);
  const [persistHistory, setPersistHistory] = React.useState(
    () => readNexAssistantPreferences().persistHistory,
  );
  const [confirmClear, setConfirmClear] = React.useState(false);
  const [draft, setDraft] = React.useState('');
  const [includeProjectContext, setIncludeProjectContext] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState('');
  const inFlight = React.useRef(false);
  const chatAbortController = React.useRef<AbortController | null>(null);
  const allowResultsRef = React.useRef(!privacyMode);
  allowResultsRef.current = !privacyMode;

  React.useEffect(() => {
    chatAbortController.current?.abort();
    setTurns([]);
    setOverview(null);
    setDraft('');
    setError('');
    setIncludeProjectContext(false);
    setConfirmClear(false);
  }, [projectId]);

  React.useEffect(() => () => chatAbortController.current?.abort(), []);

  React.useEffect(() => {
    if (privacyMode) {
      chatAbortController.current?.abort();
      setExpanded(false);
      setTurns([]);
      setDraft('');
      setError('');
      setIncludeProjectContext(false);
    }
  }, [privacyMode]);

  React.useEffect(() => {
    if (!expanded || privacyMode) return;
    let active = true;
    nexus
      .getNexAssistantOverview(projectId)
      .then((next) => {
        if (active) setOverview(next);
      })
      .catch(() => {
        if (active) setError('Não foi possível consultar o estado atual do Nexus.');
      });
    return () => {
      active = false;
    };
  }, [expanded, projectId, privacyMode]);

  const changePersistence = (next: boolean) => {
    setPersistHistory(next);
    saveNexAssistantPreferences({ version: 1, mode: readNexAssistantPreferences().mode, persistHistory: next });
    setTurns([]);
    setConfirmClear(false);
  };

  React.useEffect(() => {
    if (!expanded || privacyMode || !persistHistory) return;
    let active = true;
    nexus
      .getNexConversationHistory(projectId)
      .then((history) => {
        if (!active || inFlight.current) return;
        setTurns(
          (history.turns || []).map((turn) => ({
            role: turn.role,
            content: turn.content,
          })),
        );
      })
      .catch(() => {
        if (active) setError('Não foi possível recuperar a conversa armazenada no Nexus.');
      });
    return () => {
      active = false;
    };
  }, [expanded, privacyMode, persistHistory, projectId]);

  const clearHistory = async () => {
    if (!confirmClear || busy || privacyMode) return;
    try {
      await nexus.deleteNexConversationHistory(projectId);
      setTurns([]);
      setConfirmClear(false);
    } catch {
      setError('Não foi possível apagar o histórico armazenado.');
    }
  };

  const runSuggestion = (suggestion: NexAssistantSuggestion) => {
    switch (suggestion.target) {
      case 'missions':
        onOpenMissions?.();
        break;
      case 'agents':
        onOpenAgents?.();
        break;
      case 'usage':
        onOpenUsage?.();
        break;
      default:
        onOpenNexus?.();
    }
  };

  const availableTarget = (suggestion: NexAssistantSuggestion): boolean => {
    switch (suggestion.target) {
      case 'missions':
        return Boolean(onOpenMissions);
      case 'agents':
        return Boolean(onOpenAgents);
      case 'usage':
        return Boolean(onOpenUsage);
      default:
        return Boolean(onOpenNexus);
    }
  };

  const sendMessage = async (event: React.FormEvent) => {
    event.preventDefault();
    const message = draft.trim();
    if (!message || inFlight.current || privacyMode) return;
    inFlight.current = true;
    setBusy(true);
    setError('');
    setDraft('');
    const controller = new AbortController();
    chatAbortController.current = controller;
    const history = turns.slice(-8).map((turn) => ({
      role: turn.role,
      content: turn.content.slice(0, 200),
    }));
    setTurns((current) => [...current, { role: 'user' as const, content: message }].slice(-12));
    try {
      const reply = await nexus.askNexAssistant(
        {
          message,
          project_id: projectId || '',
          history,
          include_project_context: includeProjectContext && Boolean(projectId),
          persist: persistHistory,
          request_id: persistHistory ? window.crypto.randomUUID() : undefined,
        },
        controller.signal,
      );
      if (controller.signal.aborted || !allowResultsRef.current) return;
      setTurns((current) =>
        [...current, { role: 'assistant' as const, content: reply.answer }].slice(-12),
      );
    } catch {
      if (controller.signal.aborted || !allowResultsRef.current) return;
      setError(
        'O Nex não conseguiu responder. Confira a conexão e a configuração da inteligência. Os dados operacionais continuam disponíveis no HUD.',
      );
    } finally {
      if (chatAbortController.current === controller) chatAbortController.current = null;
      inFlight.current = false;
      setBusy(false);
    }
  };

  return (
    <section className={styles.shell} aria-label="Assistente Nex" data-nex-assistant="v1">
      <button
        type="button"
        className={styles.openButton}
        onClick={() => setExpanded((current) => !current)}
        aria-expanded={expanded}
        disabled={privacyMode}
      >
        <span>
          <MessageCircle size={14} aria-hidden="true" /> Assistente Nex
        </span>
        {expanded ? (
          <ChevronUp size={14} aria-hidden="true" />
        ) : (
          <ChevronDown size={14} aria-hidden="true" />
        )}
      </button>
      {privacyMode && (
        <p className={styles.caption}>Assistente oculto pelo modo de privacidade do HUD.</p>
      )}
      {expanded && !privacyMode && (
        <div className={styles.body}>
          <p className={styles.caption}>
            Posso explicar o estado do Nexus e ajudar a encontrar o próximo passo. Nenhuma ação será
            executada apenas por conversar.
          </p>
          <NexProactivityControls
            active={expanded}
            projectId={projectId}
            revision={revision}
            privacyMode={privacyMode}
            onOpenMissions={onOpenMissions}
            onOpenUsage={onOpenUsage}
          />
          {overview && (
            <p className={styles.caption}>
              Estado consultado: {overview.agent_count} agentes ativos · {overview.needs_user}{' '}
              decisões pendentes.
            </p>
          )}
          {projectId && (
            <label className={styles.setting}>
              <input
                type="checkbox"
                checked={includeProjectContext}
                onChange={(event) => setIncludeProjectContext(event.target.checked)}
              />
              Permitir enviar fatos estáticos do projeto ao modelo configurado
            </label>
          )}
          <div className={styles.chatLog} aria-live="polite" aria-label="Conversa com Nex">
            {turns.map((turn, index) => (
              <p className={styles.turn} data-role={turn.role} key={index}>
                {turn.content}
              </p>
            ))}
          </div>
          <NexAssistantActions
            active={expanded}
            projectId={projectId}
            allowProjectContext={includeProjectContext}
            onResult={(message) => {
              if (!allowResultsRef.current) return;
              setTurns((current) =>
                [...current, { role: 'assistant' as const, content: message }].slice(-12),
              );
            }}
          />
          <form onSubmit={sendMessage} className={styles.inputRow}>
            <input
              className={styles.input}
              value={draft}
              maxLength={2000}
              onChange={(event) => setDraft(event.target.value)}
              placeholder="Pergunte ao Nex..."
              aria-label="Perguntar ao Nex"
              disabled={busy}
            />
            <button className={styles.send} type="submit" disabled={busy || !draft.trim()}>
              {busy ? '...' : 'Enviar'}
            </button>
          </form>
          {error && (
            <p className={styles.error} role="alert">
              {error}
            </p>
          )}
          <p className={styles.caption}>
            Sem voz · sem acesso automático a arquivos · contexto independente do Maestro
          </p>
        </div>
      )}
    </section>
  );
};
