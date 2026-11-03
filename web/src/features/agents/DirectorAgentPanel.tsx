import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../design-system';
import { nexus } from '../../nexus/api';
import type {
  Agent,
  DirectorActionKind,
  DirectorAgentProjection,
  DirectorAvailability,
} from '../../types';
import styles from './DirectorAgentPanel.module.scss';
import { useDirectorPolling } from './useDirectorPolling';

const kinds: readonly DirectorActionKind[] = [
  'INSPECT',
  'DIRECTIVE',
  'STEER',
  'QUEUE_NEXT',
  'WAIT',
  'CANCEL',
];

const availabilityOptions: readonly DirectorAvailability[] = [
  'SUPPORTED',
  'PREPARE_ONLY',
  'OBSERVE_ONLY',
  'TERMINAL_ONLY',
  'UNSUPPORTED',
];

export function DirectorAgentPanel({
  agent,
  onPrepare,
  onTerminal,
}: {
  agent: Agent;
  onPrepare: () => void;
  onTerminal: () => void;
}) {
  const { t, i18n } = useTranslation();
  const [view, setView] = useState<DirectorAgentProjection | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refreshing, setRefreshing] = useState(false);
  const [lastObservedAt, setLastObservedAt] = useState<string | null>(null);
  const { revision, refresh } = useDirectorPolling(loading || refreshing);

  useEffect(() => {
    let active = true;
    setRefreshing(true);
    setError('');

    nexus
      .getDirectorAgent(agent.project_id, agent.id)
      .then((result) => {
        if (active) {
          setView(result);
          setLastObservedAt(new Date().toISOString());
        }
      })
      .catch((err: unknown) => {
        if (active) setError(err instanceof Error ? err.message : t('director.error'));
      })
      .finally(() => {
        if (active) {
          setLoading(false);
          setRefreshing(false);
        }
      });

    return () => {
      active = false;
    };
  }, [agent.id, agent.project_id, revision, t]);

  const canPrepare = !error && view?.actions.some(
    (action) => action.kind === 'DIRECTIVE' && action.availability === 'PREPARE_ONLY',
  );
  const canAttach = !error && view?.process_state === 'LIVE' && Boolean(view.runtime_id);
  const metadata = view
    ? [
        { label: t('director.agent'), value: agent.name },
        { label: t('director.persisted'), value: view.agent_status },
        {
          label: t('director.process'),
          value: t('director.processStates.' + view.process_state),
        },
        {
          label: t('director.evidence'),
          value: t('director.processEvidence.' + view.process_evidence),
        },
        {
          label: t('director.provider'),
          value: view.provider + ' / ' + (view.profile || t('director.notAvailable')),
        },
        {
          label: t('director.generation'),
          value: view.generation_id || t('director.notAvailable'),
        },
        { label: t('director.runtime'), value: view.runtime_id || t('director.notAvailable') },
      ]
    : [];

  return (
    <section className={styles.panel} aria-label={t('director.title')}>
      <p className={styles.subtitle}>{t('director.subtitle')}</p>
      <div className={styles.controls}>
        <Button size="sm" tone="ghost" disabled={loading || refreshing} onClick={refresh}>
          {t('director.refresh')}
        </Button>
        <Button size="sm" tone="brand" disabled={loading || !canPrepare} onClick={onPrepare}>
          {t('director.prepare')}
        </Button>
        <Button size="sm" tone="ghost" disabled={loading || !canAttach} onClick={onTerminal}>
          {t('director.terminal')}
        </Button>
      </div>
      {loading && <p role="status">{t('director.loading')}</p>}
      {refreshing && !loading && (
        <p role="status" className={styles.notice}>
          {t('director.refreshing')}
        </p>
      )}
      {lastObservedAt && !loading && (
        <p className={styles.notice}>
          {t('director.lastChecked', {
            time: new Date(lastObservedAt).toLocaleTimeString(i18n.language),
          })}
        </p>
      )}
      {error && !loading && (
        <p role="alert" className={styles.error}>
          {error}
        </p>
      )}
      {view && !loading && (
        <>
          <dl className={styles.metadata}>
            {metadata.map(({ label, value }) => (
              <div key={label}>
                <dt>{label}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
          <h3 className={styles.heading}>{t('director.controls')}</h3>
          <p className={styles.notice}>{t('director.preparationNotice')}</p>
          <ul className={styles.actions}>
            {kinds.map((kind) => {
              const action = view.actions.find((item) => item.kind === kind);
              const availability = action?.availability;
              const state =
                availability && availabilityOptions.includes(availability)
                  ? availability
                  : 'UNSUPPORTED';
              return (
                <li key={kind} className={styles.action}>
                  <strong>{t('director.actions.' + kind)}</strong>
                  <span>{t('director.states.' + state)}</span>
                </li>
              );
            })}
          </ul>
          <p className={styles.notice}>{t('director.notCertified')}</p>
          <h3 className={styles.heading}>{t('director.recentActivity')}</h3>
          {view.recent_events?.length ? (
            <ol className={styles.events}>
              {view.recent_events.map((event) => (
                <li key={event.id} className={styles.event}>
                  <strong>{event.kind}</strong>
                  <time dateTime={event.ts}>
                    {new Date(event.ts).toLocaleString(i18n.language)}
                  </time>
                  {event.correlation_id && (
                    <code className={styles.eventCorrelation}>{event.correlation_id}</code>
                  )}
                </li>
              ))}
            </ol>
          ) : (
            <p className={styles.notice}>{t('director.noRecentActivity')}</p>
          )}
        </>
      )}
    </section>
  );
}
