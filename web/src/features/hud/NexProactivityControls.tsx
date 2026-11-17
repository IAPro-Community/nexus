import React from 'react';
import { nexus } from '../../nexus/api';
import type { NexProactivityCandidate, NexProactivitySettings } from './nexProactivityTypes';
import styles from './NexAssistantPanel.module.scss';

export interface NexProactivityControlsProps {
  projectId?: string;
  revision?: string;
  active: boolean;
  privacyMode: boolean;
  onOpenMissions?: () => void;
  onOpenUsage?: () => void;
}

const LOCAL_ZONE = (() => {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC';
  } catch {
    return 'UTC';
  }
})();

/**
 * Only derives proposals from the already-polled HUD snapshot revision.
 * No background agent, polling timer, model call or duplicate scheduler.
 * A claim is atomically deduplicated in the Nexus Core store, across clients.
 */
export const NexProactivityControls: React.FC<NexProactivityControlsProps> = ({
  projectId,
  revision,
  active,
  privacyMode,
  onOpenMissions,
  onOpenUsage,
}) => {
  const scope = projectId || '';
  const [settings, setSettings] = React.useState<NexProactivitySettings | null>(null);
  const [draft, setDraft] = React.useState<NexProactivitySettings | null>(null);
  const [candidates, setCandidates] = React.useState<NexProactivityCandidate[]>([]);
  const [proposed, setProposed] = React.useState<NexProactivityCandidate | null>(null);
  const [busy, setBusy] = React.useState(false);
  const [error, setError] = React.useState('');
  const epoch = React.useRef(0);

  React.useEffect(() => {
    epoch.current += 1;
    setSettings(null);
    setDraft(null);
    setCandidates([]);
    setProposed(null);
    setError('');
  }, [scope, privacyMode]);

  React.useEffect(() => {
    if (!active || privacyMode || !revision) return;
    const generation = ++epoch.current;
    const canceled = () => epoch.current !== generation;
    const refresh = async () => {
      try {
        const result = await nexus.getNexProactivity(scope);
        if (canceled()) return;
        setSettings(result.settings);
        setDraft((current) => current || result.settings);
        setCandidates(result.settings.mode === 'contextual' ? result.candidates || [] : []);
        if (result.settings.mode !== 'proactive') {
          setProposed(null);
          return;
        }
        if (proposed) return;
        // Surface only fresh evidence; old alerts remain in Center of Attention.
        const recent = (result.candidates || []).filter((candidate) => {
          const time = Date.parse(candidate.observed_at);
          return (
            Number.isFinite(time) && Date.now() - time >= 0 && Date.now() - time <= 15 * 60 * 1000
          );
        });
        for (const item of recent) {
          const claim = await nexus.claimNexSuggestion(scope, item.fingerprint);
          if (canceled()) return;
          if (claim.claimed && claim.suggestion) {
            setProposed(claim.suggestion);
            break;
          }
        }
      } catch {
        if (!canceled()) setError('Não foi possível carregar as sugestões do Nexus.');
      }
    };
    void refresh();
    return () => {
      epoch.current += 1;
    };
    // Proposal is held while visible; new snapshots cannot overwrite it.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [active, privacyMode, revision, scope]);

  const changeSettings = <K extends keyof NexProactivitySettings>(
    field: K,
    value: NexProactivitySettings[K],
  ) => setDraft((current) => (current ? { ...current, [field]: value } : current));

  const save = async () => {
    if (!draft || busy || privacyMode) return;
    setBusy(true);
    setError('');
    try {
      await nexus.configureNexProactivity(scope, draft);
      setSettings(draft);
      setProposed(null);
      setCandidates([]);
    } catch {
      setError('As preferências não foram salvas. Verifique fuso horário e intervalo.');
    } finally {
      setBusy(false);
    }
  };

  const decide = async (choice: 'dismiss' | 'snooze') => {
    if (!proposed || busy || privacyMode) return;
    setBusy(true);
    setError('');
    try {
      await nexus.decideNexSuggestion(
        scope,
        proposed.fingerprint,
        choice,
        choice === 'snooze' ? 60 : 0,
      );
      setProposed(null);
    } catch {
      setError('Não foi possível registrar a decisão sobre a sugestão.');
    } finally {
      setBusy(false);
    }
  };

  const navigate = (candidate: NexProactivityCandidate) => {
    if (candidate.target === 'missions') onOpenMissions?.();
    if (candidate.target === 'usage') onOpenUsage?.();
  };

  if (!active || privacyMode) return null;

  return (
    <section className={styles.scheduleBox} aria-label="Proatividade do Nex">
      <strong>Iniciativa do Nex</strong>
      <span className={styles.caption}>
        Sugestões baseadas nos eventos reais do Nexus. Nenhuma execução é iniciada automaticamente.
      </span>
      {draft && (
        <>
          <label className={styles.setting}>
            Comportamento
            <select
              className={styles.input}
              value={draft.mode}
              disabled={busy}
              onChange={(event) => {
                const next = event.target.value as NexProactivitySettings['mode'];
                setDraft((current) =>
                  current
                    ? {
                        ...current,
                        mode: next,
                        timezone:
                          current.mode === 'manual' &&
                          current.timezone === 'UTC' &&
                          next !== 'manual'
                            ? LOCAL_ZONE
                            : current.timezone,
                      }
                    : current,
                );
              }}
            >
              <option value="manual">Somente quando eu pedir</option>
              <option value="contextual">Mostrar sugestões no painel</option>
              <option value="proactive">Sugerir novidades enquanto o HUD estiver ativo</option>
            </select>
          </label>
          <label className={styles.setting}>
            Silêncio a partir de
            <input
              className={styles.input}
              type="time"
              value={draft.quiet_start}
              disabled={busy}
              onChange={(event) => changeSettings('quiet_start', event.target.value)}
            />
          </label>
          <label className={styles.setting}>
            Até
            <input
              className={styles.input}
              type="time"
              value={draft.quiet_end}
              disabled={busy}
              onChange={(event) => changeSettings('quiet_end', event.target.value)}
            />
          </label>
          <label className={styles.setting}>
            Fuso IANA
            <input
              className={styles.input}
              value={draft.timezone}
              disabled={busy}
              onChange={(event) => changeSettings('timezone', event.target.value)}
              placeholder="America/Sao_Paulo"
            />
            <button
              type="button"
              className={styles.send}
              disabled={busy}
              onClick={() => changeSettings('timezone', LOCAL_ZONE)}
            >
              Usar local
            </button>
          </label>
          <label className={styles.setting}>
            Intervalo mínimo (minutos)
            <input
              className={styles.input}
              type="number"
              min={5}
              max={1440}
              value={draft.min_interval_minutes}
              disabled={busy}
              onChange={(event) =>
                changeSettings('min_interval_minutes', Number(event.target.value))
              }
            />
          </label>
          <button
            type="button"
            className={styles.send}
            disabled={busy || !settings || JSON.stringify(settings) === JSON.stringify(draft)}
            onClick={() => void save()}
          >
            Salvar preferências
          </button>
        </>
      )}
      {settings?.mode === 'contextual' &&
        candidates.map((candidate) => (
          <button
            type="button"
            className={styles.action}
            key={candidate.fingerprint}
            disabled={busy || (candidate.target === 'missions' ? !onOpenMissions : !onOpenUsage)}
            onClick={() => navigate(candidate)}
          >
            <strong>{candidate.title}</strong>
            <small>{candidate.description}</small>
          </button>
        ))}
      {settings?.mode === 'proactive' && proposed && (
        <div className={styles.scheduleBox} aria-live="polite">
          <strong>{proposed.title}</strong>
          <span>{proposed.description}</span>
          <div className={styles.historyActions}>
            <button
              type="button"
              className={styles.send}
              disabled={busy || (proposed.target === 'missions' ? !onOpenMissions : !onOpenUsage)}
              onClick={() => navigate(proposed)}
            >
              Revisar
            </button>
            <button
              type="button"
              className={styles.send}
              disabled={busy}
              onClick={() => void decide('snooze')}
            >
              Adiar 1h
            </button>
            <button
              type="button"
              className={styles.send}
              disabled={busy}
              onClick={() => void decide('dismiss')}
            >
              Dispensar
            </button>
          </div>
        </div>
      )}
      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}
    </section>
  );
};
