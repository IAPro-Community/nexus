import { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, DirectorFleetProjection } from '../../types';
import styles from './DirectorAgentPanel.module.scss';
import { useDirectorPolling } from './useDirectorPolling';
import { filterDirectorFleet, type FleetStateFilter } from './directorFleetModel';

export function DirectorFleetPanel({
  projectId,
  agents,
  onInspect,
}: {
  projectId: string;
  agents: Agent[];
  onInspect: (agent: Agent) => void;
}) {
  const { t, i18n } = useTranslation();
  const [view, setView] = useState<DirectorFleetProjection | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [refreshing, setRefreshing] = useState(false);
  const [lastObservedAt, setLastObservedAt] = useState<string | null>(null);
  const [filter, setFilter] = useState<FleetStateFilter>('all');
  const [search, setSearch] = useState('');
  const { revision, refresh } = useDirectorPolling(loading || refreshing);
  const agentById = useMemo(() => new Map(agents.map((agent) => [agent.id, agent])), [agents]);
  const visible = useMemo(
    () => filterDirectorFleet(view?.agents ?? [], agentById, filter, search),
    [view, agentById, filter, search],
  );

  useEffect(() => {
    let active = true;
    setRefreshing(true);
    setError('');

    nexus
      .getDirectorFleet(projectId)
      .then((value) => {
        if (active) {
          setView(value);
          setLastObservedAt(new Date().toISOString());
        }
      })
      .catch((reason: unknown) => {
        if (active) setError(reason instanceof Error ? reason.message : t('director.error'));
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
  }, [projectId, revision, t]);

  return (
    <section className={styles.panel} aria-label={t('director.fleetTitle')}>
      <p className={styles.subtitle}>{t('director.subtitle')}</p>
      <div className={styles.controls}>
        <Button size="sm" tone="ghost" disabled={loading || refreshing} onClick={refresh}>
          {t('director.refresh')}
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
          <p className={styles.notice}>
            {t('director.fleetCount', { shown: view.agents.length, total: view.total })}
          </p>
          <div className={styles.fleetFilters}>
            <label>
              {t('director.filterStatus')}
              <select
                value={filter}
                onChange={(event) => setFilter(event.target.value as FleetStateFilter)}
              >
                <option value="all">{t('director.filterAll')}</option>
                <option value="blocked">{t('director.filterBlocked')}</option>
                <option value="working">{t('director.filterWorking')}</option>
                <option value="idle">{t('director.filterIdle')}</option>
                <option value="done">{t('director.filterDone')}</option>
                <option value="offline">{t('director.filterOffline')}</option>
              </select>
            </label>
            <label>
              {t('director.filterSearch')}
              <input
                type="search"
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                placeholder={t('director.filterSearch')}
              />
            </label>
          </div>
          <p className={styles.notice} role="status">
            {t('director.filteredCount', { shown: visible.length, total: view.agents.length })}
          </p>
          <ul className={styles.actions}>
            {visible.map((item) => {
              const agent = agentById.get(item.agent_id);
              return (
                <li className={styles.action} key={item.agent_id}>
                  <span>
                    <strong>{agent?.name || item.agent_id}</strong>
                    {' · '}
                    {t('director.processStates.' + item.process_state)}
                    {' · '}
                    <span title={item.observed_reason || undefined}>
                      {t('director.observedState')}: {item.observed_state || 'unknown'}
                    </span>
                    {' · '}
                    {item.provider || t('director.notAvailable')}
                  </span>
                  <Button
                    size="sm"
                    tone="ghost"
                    disabled={!agent}
                    onClick={() => {
                      if (agent) onInspect(agent);
                    }}
                  >
                    {t('director.inspect')}
                  </Button>
                </li>
              );
            })}
          </ul>
          {view.agents.length === 0 && <p>{t('director.fleetEmpty')}</p>}
          {view.agents.length > 0 && visible.length === 0 && (
            <p className={styles.notice}>{t('director.filterNone')}</p>
          )}
          {view.truncated && <p className={styles.notice}>{t('director.fleetTruncated')}</p>}
        </>
      )}
    </section>
  );
}
