import React, { useCallback, useEffect, useState } from 'react';
import { FileWarning, RefreshCw } from 'lucide-react';
import { Badge, Button, Card, IconButton } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ChangeCollisionsReport } from '../../types';
import { useTranslation } from 'react-i18next';
import {
  confidenceLabelKey,
  hasPartialCoverage,
  sortCollisions,
  summarizeCollisions,
  worktreeLabel,
} from './changeCollisionsModel';
import styles from './ChangeCollisionsCard.module.scss';

/**
 * Read-only surface for the change-collision detector. It shows which files
 * are being changed by more than one worktree and keeps the detector's
 * uncertainty explicit: same-file candidates are never presented as
 * confirmed conflicts, and the card never blocks merges by itself.
 */
export const ChangeCollisionsCard: React.FC<{ projectId: string }> = ({ projectId }) => {
  const { t } = useTranslation();
  const [report, setReport] = useState<ChangeCollisionsReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const scan = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      setReport(await nexus.getProjectChangeCollisions(projectId));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    void scan();
  }, [scan]);

  const summary = report ? summarizeCollisions(report) : null;
  const collisions = report ? sortCollisions(report.collisions || []) : [];

  return (
    <Card className={styles.card} data-testid="change-collisions-card">
      <div className={styles.header}>
        <span className={styles.titleGroup}>
          <FileWarning size={16} aria-hidden="true" />
          <span className={styles.title}>{t('agents.collisions.title')}</span>
          {summary && summary.total > 0 ? (
            <Badge tone={summary.hunkOverlaps > 0 ? 'danger' : 'warning'}>{summary.total}</Badge>
          ) : null}
        </span>
        <IconButton
          label={t('agents.collisions.refresh')}
          onClick={() => void scan()}
          disabled={loading}
        >
          <RefreshCw size={14} aria-hidden="true" />
        </IconButton>
      </div>

      <p className={styles.status} aria-live="polite">
        {loading
          ? t('agents.collisions.scanning')
          : error
            ? t('agents.collisions.error', { message: error })
            : summary
              ? t('agents.collisions.scanned', { count: summary.scannedWorktrees })
              : ''}
      </p>

      {error ? (
        <Button tone="default" onClick={() => void scan()}>
          {t('agents.collisions.retry')}
        </Button>
      ) : null}

      {!error && summary && summary.total === 0 ? (
        <p className={styles.status}>{t('agents.collisions.none')}</p>
      ) : null}

      {collisions.length > 0 ? (
        <>
          <ul className={styles.list}>
            {collisions.map((collision) => (
              <li
                key={`${collision.file}:${collision.worktrees.join('|')}`}
                className={styles.item}
              >
                <span className={styles.itemHeader}>
                  <span className={styles.file}>{collision.file}</span>
                  <Badge tone={collision.confidence === 'hunk-overlap' ? 'danger' : 'warning'}>
                    {t(confidenceLabelKey(collision))}
                  </Badge>
                </span>
                <p className={styles.reason}>{collision.reason}</p>
                <span className={styles.worktrees}>
                  {collision.worktrees.map((path) => (
                    <Badge key={path} tone="default">
                      {worktreeLabel(path, report?.worktrees.find((w) => w.path === path)?.branch)}
                    </Badge>
                  ))}
                </span>
              </li>
            ))}
          </ul>
          <p className={styles.reviewHint}>{t('agents.collisions.reviewHint')}</p>
        </>
      ) : null}

      {report && hasPartialCoverage(report) ? (
        <p className={styles.partial}>
          {t('agents.collisions.partial', { count: (report.scan_errors || []).length })}
        </p>
      ) : null}
    </Card>
  );
};
