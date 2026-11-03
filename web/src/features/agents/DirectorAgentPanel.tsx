import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, DirectorAgentProjection, DirectorActionKind, DirectorAvailability } from '../../types';
import styles from './DirectorAgentPanel.module.scss';

const kinds: readonly DirectorActionKind[] = [
  'INSPECT', 'DIRECTIVE', 'STEER', 'QUEUE_NEXT', 'WAIT', 'CANCEL',
];
const availabilityOptions: readonly DirectorAvailability[] = [
  'SUPPORTED', 'PREPARE_ONLY', 'OBSERVE_ONLY', 'TERMINAL_ONLY', 'UNSUPPORTED',
];

export function DirectorAgentPanel({ agent, onPrepare, onTerminal }: {
  agent: Agent;
  onPrepare: () => void;
  onTerminal: () => void;
}) {
  const { t } = useTranslation();
  const [revision, setRevision] = useState(0);
  const [view, setView] = useState<DirectorAgentProjection | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  useEffect(() => {
    let active = true;
    setView(null);
    setLoading(true);
    setError('');
    nexus.getDirectorAgent(agent.project_id, agent.id)
      .then((result) => { if (active) setView(result); })
      .catch((err: unknown) => {
        if (active) setError(err instanceof Error ? err.message : t('director.error'));
      })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [agent.id, agent.project_id, revision, t]);

  return (
    <section className={styles.panel} aria-label={t('director.title')}>
      <p className={styles.subtitle}>{t('director.subtitle')}</p>
      <div className={styles.controls}>
        <Button size="sm" tone="ghost" disabled={loading} onClick={() => setRevision((v) => v + 1)}>
          {t('director.refresh')}
        </Button>
        <Button size="sm" tone="brand" disabled={loading || !view} onClick={onPrepare}>
          {t('director.prepare')}
        </Button>
        <Button size="sm" tone="ghost" onClick={onTerminal}>
          {t('director.terminal')}
        </Button>
      </div>
      {loading && <p role="status">{t('director.loading')}</p>}
      {error && !loading && <p role="alert" className={styles.error}>{error}</p>}
      {view && !loading && (
        <>
          <dl className={styles.metadata}>
            <div><dt>{t('director.agent')}</dt><dd>{agent.name}</dd></div>
            <div><dt>{t('director.persisted')}</dt><dd>{view.agent_status}</dd></div>
            <div><dt>{t('director.process')}</dt><dd>{t('director.processStates.' + view.process_state)}</dd></div>
            <div><dt>{t('director.evidence')}</dt><dd>{t('director.processEvidence.' + view.process_evidence)}</dd></div>
            <div><dt>{t('director.provider')}</dt><dd>{view.provider || t('director.notAvailable')} / {view.profile || t('director.notAvailable')}</dd></div>
            <div><dt>{t('director.generation')}</dt><dd>{view.generation_id || t('director.notAvailable')}</dd></div>
            <div><dt>{t('director.runtime')}</dt><dd>{view.runtime_id || t('director.notAvailable')}</dd></div>
          </dl>
          <h3 className={styles.heading}>{t('director.controls')}</h3>
          <p className={styles.notice}>{t('director.preparationNotice')}</p>
          <ul className={styles.actions}>
            {kinds.map((kind) => {
              const action = view.actions.find((item) => item.kind === kind);
              const state = availabilityOptions.includes(action?.availability as DirectorAvailability)
                ? action?.availability as DirectorAvailability : 'UNSUPPORTED';
              return (
                <li key={kind} className={styles.action}>
                  <strong>{t('director.actions.' + kind)}</strong>
                  <span>{t('director.states.' + state)}</span>
                </li>
              );
            })}
          </ul>
          <p className={styles.notice}>{t('director.notCertified')}</p>
        </>
      )}
    </section>
  );
}
