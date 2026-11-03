import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, DirectorFleetProjection } from '../../types';
import styles from './DirectorAgentPanel.module.scss';

export function DirectorFleetPanel({
  projectId, agents, onInspect,
}: {
  projectId: string;
  agents: Agent[];
  onInspect: (agent: Agent) => void;
}) {
  const { t } = useTranslation();
  const [refresh, setRefresh] = useState(0);
  const [view, setView] = useState<DirectorFleetProjection | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  useEffect(() => {
    let active = true;
    setLoading(true);
    setError('');
    setView(null);
    nexus.getDirectorFleet(projectId)
      .then((value) => { if (active) setView(value); })
      .catch((reason: unknown) => {
        if (active) setError(reason instanceof Error ? reason.message : t('director.error'));
      })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, [projectId, refresh, t]);

  return (
    <section className={styles.panel} aria-label={t('director.fleetTitle')}>
      <p className={styles.subtitle}>{t('director.subtitle')}</p>
      <div className={styles.controls}>
        <Button size="sm" tone="ghost" disabled={loading} onClick={() => setRefresh((v) => v + 1)}>
          {t('director.refresh')}
        </Button>
      </div>
      {loading && <p role="status">{t('director.loading')}</p>}
      {error && !loading && <p role="alert" className={styles.error}>{error}</p>}
      {view && !loading && (
        <>
          <p className={styles.notice}>
            {t('director.fleetCount', { shown: view.agents.length, total: view.total })}
          </p>
          <ul className={styles.actions}>
            {view.agents.map((item) => {
              const target = agents.find((agent) => agent.id === item.agent_id);
              return (
                <li className={styles.action} key={item.agent_id}>
                  <span>
                    <strong>{target?.name || item.agent_id}</strong>
                    {' · '}{item.process_state}
                    {' · '}{item.provider || t('director.notAvailable')}
                  </span>
                  <Button size="sm" tone="ghost" disabled={!target} onClick={() => {
                    if (target) onInspect(target);
                  }}>
                    {t('director.inspect')}
                  </Button>
                </li>
              );
            })}
          </ul>
          {view.agents.length === 0 && <p>{t('director.fleetEmpty')}</p>}
          {view.truncated && <p className={styles.notice}>{t('director.fleetTruncated')}</p>}
        </>
      )}
    </section>
  );
}
