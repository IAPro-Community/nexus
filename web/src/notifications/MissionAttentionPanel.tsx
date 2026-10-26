import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { nexus } from '../nexus/api';
import { asArray } from '../lib/safeArray';
import type { AttentionGroup, AttentionItem, Project } from '../types';
import { sanitizeAttentionText } from '../components/attentionText';
import { missionAttentionSections } from '../nexus/missionAttentionModel';
import styles from './MissionAttentionPanel.module.scss';

export const MissionAttentionPanel: React.FC<{
  onOpenMission: (item: AttentionItem) => void;
}> = ({ onOpenMission }) => {
  const { t } = useTranslation();
  const [projects, setProjects] = useState<Project[]>([]);
  const [attention, setAttention] = useState<AttentionGroup | null>(null);
  const [projectId, setProjectId] = useState('');
  const [missionId, setMissionId] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    void nexus
      .listProjects()
      .then((result) => {
        if (active) setProjects(asArray<Project>(result));
      })
      .catch(() => {
        if (active) setProjects([]);
      });
    return () => {
      active = false;
    };
  }, []);

  const filter = useMemo(
    () => ({ projectId: projectId || undefined, missionId: missionId || undefined }),
    [missionId, projectId],
  );

  useEffect(() => {
    let active = true;
    const refresh = async (initial: boolean) => {
      if (initial) setLoading(true);
      try {
        const result = await nexus.getAttention(filter);
        if (!active) return;
        setAttention(result);
        setError('');
      } catch (cause) {
        if (!active) return;
        setAttention(null);
        setError(cause instanceof Error ? cause.message : String(cause));
      } finally {
        if (active && initial) setLoading(false);
      }
    };

    void refresh(true);
    const interval = window.setInterval(() => void refresh(false), 15_000);
    return () => {
      active = false;
      window.clearInterval(interval);
    };
  }, [filter]);

  const sections = missionAttentionSections(attention);
  const missionOptions = asArray<AttentionItem>(attention?.all_items);
  const projectNames = useMemo(
    () => new Map(projects.map((project) => [project.id, project.name || project.id])),
    [projects],
  );
  const itemCount = sections.reduce((sum, section) => sum + section.items.length, 0);

  return (
    <div className={styles.panel}>
      <div className={styles.filters}>
        <label className="nx-sr-only" htmlFor="mission-attention-project-filter">
          {t('notifications.filterProject')}
        </label>
        <select
          id="mission-attention-project-filter"
          className={styles.filter}
          value={projectId}
          onChange={(event) => {
            setProjectId(event.currentTarget.value);
            setMissionId('');
          }}
        >
          <option value="">{t('notifications.allProjects')}</option>
          {projects.map((project) => (
            <option key={project.id} value={project.id}>
              {project.name || project.id}
            </option>
          ))}
        </select>

        <label className="nx-sr-only" htmlFor="mission-attention-mission-filter">
          {t('notifications.filterMission')}
        </label>
        <select
          id="mission-attention-mission-filter"
          className={styles.filter}
          value={missionId}
          onChange={(event) => setMissionId(event.currentTarget.value)}
          disabled={missionOptions.length === 0}
        >
          <option value="">{t('notifications.allMissions')}</option>
          {missionOptions.map((item) => (
            <option key={item.mission_id} value={item.mission_id}>
              {item.summary || item.mission_id}
            </option>
          ))}
        </select>
      </div>

      {loading && (
        <p className={styles.status} role="status">
          {t('common.loading')}
        </p>
      )}
      {error && (
        <p className={`${styles.status} ${styles.error}`} role="alert">
          {t('notifications.missionAttentionError', { error })}
        </p>
      )}
      {!loading && !error && itemCount === 0 && (
        <p className={styles.empty}>{t('notifications.noMissionAttention')}</p>
      )}

      {!loading && itemCount > 0 && (
        <div className={styles.groups}>
          {sections.map((section) => (
            <section key={section.id} className={styles.group}>
              <h3 className={styles.groupTitle}>
                {t(`notifications.missionAttention.${section.id}`)}
                <span className={styles.count}>{section.items.length}</span>
              </h3>
              {section.items.map((item) => {
                const projectName = projectNames.get(item.project_id) || item.project_id;
                const summary = sanitizeAttentionText(
                  item.summary,
                  t(`notifications.missionAttention.${section.id}`),
                );
                const detail = sanitizeAttentionText(item.question || item.state, item.state);
                return (
                  <button
                    type="button"
                    key={item.mission_id}
                    className={styles.item}
                    data-level={section.id}
                    onClick={() => onOpenMission(item)}
                    title={t('notifications.openMission', { mission: summary })}
                  >
                    <span className={styles.summary}>{summary}</span>
                    <span className={styles.detail}>{detail}</span>
                    <span className={styles.project}>
                      {sanitizeAttentionText(projectName, item.project_id)}
                    </span>
                  </button>
                );
              })}
            </section>
          ))}
        </div>
      )}
    </div>
  );
};
