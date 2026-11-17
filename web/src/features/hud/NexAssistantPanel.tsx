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
  type NexAssistantMode,
} from './nexAssistantPreferences';
import { NexAssistantActions } from './NexAssistantActions';
import styles from './NexAssistantPanel.module.scss';

export interface NexAssistantPanelProps {
  projectId?: string;
  privacyMode?: boolean;
  onOpenMissions?: () => void;
  onOpenAgents?: () => void;
  onOpenUsage?: () => void;
  onOpenNexus?: () => void;
}

/**
 * Separate from HUD operational presentation: conversations never mutate
 * missions, capacity, agent state, or the character's canonical mood.
 * History is transient and never sent to Maestro or browser localStorage.
 */
export const NexAssistantPanel: React.FC<NexAssistantPanelProps> = ({
  projectId,
  privacyMode = false,
  onOpenMissions,
  onOpenAgents,
  onOpenUsage,
  onOpenNexus,
}) => {
  const [expanded, setExpanded] = React.useState(false);
  const [overview, setOverview] = React.useState<NexAssistantOverview | null>(null);
  const [mode, setMode] = React.useState<NexAssistantMode>(
    () => readNexAssistantPreferences().mode,
  );
  const [turns, setTurns] = React.useState<NexChatTurn[]>([]);
  const [draft, setDraft] = React.useState('');
  const [includeProjectContext, setIncludeProjectContext] = React.useState(false);
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState('');
  const inFlight = React.useRef(false);
  const allowResultsRef = React.useRef(!privacyMode);
  allowResultsRef.current = !privacyMode;

  React.useEffect(() => {
    setTurns([]);
    setOverview(null);
    setDraft('');
    setError('');
    setIncludeProjectContext(false);
  }, [projectId]);

  React.useEffect(() => {
    if (privacyMode) {
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

  const changeMode = (next: NexAssistantMode) => {
    setMode(next);
    saveNexAssistantPreferences({ version: 1, mode: next });
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
    const history = turns.slice(-8).map((turn) => ({
      role: turn.role,
      content: turn.content.slice(0, 200),
    }));
    setTurns((current) => [...current, { role: 'user' as const, content: message }].slice(-12));
    try {
      const reply = await nexus.askNexAssistant({
        message,
        project_id: projectId || '',
        history,
        include_project_context: includeProjectContext && Boolean(projectId),
      });
      if (!allowResultsRef.current) return;
      setTurns((current) =>
        [...current, { role: 'assistant' as const, content: reply.answer }].slice(-12),
      );
    } catch {
      if (!allowResultsRef.current) return;
      setError(
        'O Nex não conseguiu responder. Confira a conexão e a configuração da inteligência. Os dados operacionais continuam disponíveis no HUD.',
      );
    } finally {
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
          <label className={styles.setting}>
            <input
              type="checkbox"
              checked={mode === 'contextual'}
              onChange={(event) => changeMode(event.target.checked ? 'contextual' : 'manual')}
            />
            Mostrar sugestões contextuais
          </label>
          {mode === 'contextual' && overview?.suggestions.length ? (
            <div className={styles.suggestions} aria-label="Opções com evidência operacional">
              {overview.suggestions.map((suggestion) => (
                <button
                  key={suggestion.id}
                  type="button"
                  className={styles.action}
                  disabled={!availableTarget(suggestion)}
                  onClick={() => runSuggestion(suggestion)}
                >
                  <strong>{suggestion.title}</strong>
                  <small>{suggestion.description}</small>
                </button>
              ))}
            </div>
          ) : null}
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
