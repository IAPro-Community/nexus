import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Check,
  ChevronDown,
  ChevronRight,
  CircleAlert,
  LoaderCircle,
  SkipForward,
} from 'lucide-react';
import { Badge, IconButton } from '../../design-system';
import type { NexusActivityItem, NexusActivityPayload } from './types';
import styles from './AgentActivity.module.scss';

function formatDuration(startedAt?: string, finishedAt?: string): string | null {
  if (!startedAt) return null;
  const start = Date.parse(startedAt);
  if (Number.isNaN(start)) return null;
  const end = finishedAt ? Date.parse(finishedAt) : Date.now();
  if (Number.isNaN(end)) return null;
  const secs = Math.max(0, Math.round((end - start) / 1000));
  return `${secs}s`;
}

const StatusIcon: React.FC<{ status: NexusActivityItem['status'] }> = ({ status }) => {
  if (status === 'running') return <LoaderCircle size={14} className={styles.spin} aria-hidden />;
  if (status === 'done') return <Check size={14} aria-hidden />;
  if (status === 'skipped') return <SkipForward size={14} aria-hidden />;
  if (status === 'error') return <CircleAlert size={14} aria-hidden />;
  return <span className={styles.dot} aria-hidden />;
};

/**
 * Operational agent activity — never chain-of-thought.
 * Shows current state, completed steps, optional expandable details.
 */
export const AgentActivity: React.FC<{
  activity: NexusActivityPayload;
  defaultExpanded?: boolean;
}> = ({ activity, defaultExpanded = true }) => {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(defaultExpanded || Boolean(activity.expandable));
  const current = useMemo(
    () =>
      activity.items.find((item) => item.id === activity.currentId) ||
      activity.items.find((i) => i.status === 'running'),
    [activity],
  );
  const duration = formatDuration(current?.startedAt, current?.finishedAt);

  return (
    <section className={styles.root} aria-live="polite" role="status">
      <header className={styles.header}>
        <div className={styles.titleRow}>
          <strong>{activity.title}</strong>
          {current ? (
            <Badge tone={current.status === 'error' ? 'danger' : 'info'}>
              {t(`conversation.activity.${current.status}`, current.status)}
            </Badge>
          ) : null}
          {duration ? <span className={styles.meta}>{duration}</span> : null}
        </div>
        <IconButton
          label={
            expanded ? t('conversation.collapse', 'Collapse') : t('conversation.expand', 'Expand')
          }
          onClick={() => setExpanded((v) => !v)}
        >
          {expanded ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
        </IconButton>
      </header>
      {current ? <p className={styles.current}>{current.label}</p> : null}
      {activity.error ? <p className={styles.error}>{activity.error}</p> : null}
      {expanded ? (
        <ol className={styles.list}>
          {activity.items.map((item) => (
            <li key={item.id} className={styles.item} data-status={item.status}>
              <StatusIcon status={item.status} />
              <div>
                <span>{item.label}</span>
                {item.detail ? <small className={styles.detail}>{item.detail}</small> : null}
              </div>
            </li>
          ))}
        </ol>
      ) : null}
    </section>
  );
};
