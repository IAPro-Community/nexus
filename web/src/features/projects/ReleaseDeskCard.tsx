import React, { useCallback, useEffect, useState } from 'react';
import { GitCompareArrows, RefreshCw, ShieldCheck } from 'lucide-react';
import { Badge, Button, Card, IconButton, Input } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ReleaseDeskReport } from '../../types';
import { useTranslation } from 'react-i18next';
import {
  deskTone,
  formatApprovalDate,
  gateTone,
  shortSha,
  summarizeRelease,
} from './releaseDeskModel';
import styles from './ReleaseDeskCard.module.scss';

/**
 * Read-only surface for the Release Desk. It shows the commit range linked to
 * a base ref, the canonical release gates with their honest verification
 * status, and the signed approval when one is bound to the exact head.
 * The card never suggests merge/publish by itself: the verdict badge mirrors
 * the backend's merge_publish_allowed decision, which stays blocked while any
 * gate is unverified — approval alone never unlocks integration.
 */
export const ReleaseDeskCard: React.FC<{ projectId: string }> = ({ projectId }) => {
  const { t, i18n } = useTranslation();
  const [report, setReport] = useState<ReleaseDeskReport | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const [baseInput, setBaseInput] = useState('');
  const [activeBase, setActiveBase] = useState('');

  const load = useCallback(
    async (base: string) => {
      setLoading(true);
      setError('');
      try {
        setReport(await nexus.getProjectReleaseDesk(projectId, base || undefined));
        setActiveBase(base);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setLoading(false);
      }
    },
    [projectId],
  );

  useEffect(() => {
    void load('');
  }, [load]);

  const summary = report ? summarizeRelease(report) : null;
  const locale = i18n.language || 'en';

  return (
    <Card className={styles.card} data-testid="release-desk-card">
      <div className={styles.header}>
        <span className={styles.titleGroup}>
          <GitCompareArrows size={16} aria-hidden="true" />
          <span className={styles.title}>{t('projects.releaseDesk.title')}</span>
          {summary ? (
            <Badge tone={deskTone(summary.mergePublishAllowed)}>
              {summary.mergePublishAllowed
                ? t('projects.releaseDesk.allowed')
                : t('projects.releaseDesk.blocked')}
            </Badge>
          ) : null}
        </span>
        <IconButton
          label={t('projects.releaseDesk.refresh')}
          onClick={() => void load(activeBase)}
          disabled={loading}
        >
          <RefreshCw size={14} aria-hidden="true" />
        </IconButton>
      </div>

      <div className={styles.baseRow}>
        <label className={styles.baseLabel} htmlFor="release-desk-base">
          {t('projects.releaseDesk.baseLabel')}
        </label>
        <Input
          id="release-desk-base"
          value={baseInput}
          onChange={setBaseInput}
          onEnter={() => void load(baseInput.trim())}
          placeholder={t('projects.releaseDesk.basePlaceholder')}
          mono
          className={styles.baseInput}
        />
        <Button tone="default" onClick={() => void load(baseInput.trim())} disabled={loading}>
          {t('projects.releaseDesk.analyze')}
        </Button>
      </div>

      <p className={styles.status} aria-live="polite">
        {loading
          ? t('projects.releaseDesk.loading')
          : error
            ? t('projects.releaseDesk.error', { message: error })
            : report
              ? t('projects.releaseDesk.branch', { branch: report.branch })
              : ''}
      </p>

      {error ? (
        <Button tone="default" onClick={() => void load(activeBase)}>
          {t('projects.releaseDesk.retry')}
        </Button>
      ) : null}

      {!error && report ? (
        <>
          <p className={styles.range}>
            <span className={styles.sha}>{shortSha(report.base_sha)}</span>
            {' → '}
            <span className={styles.sha}>{shortSha(report.head_sha)}</span>
          </p>
          {!activeBase ? (
            <p className={styles.hint}>{t('projects.releaseDesk.rangeMissing')}</p>
          ) : null}

          <div className={styles.refCounts}>
            <Badge tone="default">
              {t('projects.releaseDesk.commits', { count: summary?.commitCount ?? 0 })}
            </Badge>
            <Badge tone="default">
              {t('projects.releaseDesk.prs', { count: summary?.prCount ?? 0 })}
            </Badge>
            <Badge tone="default">
              {t('projects.releaseDesk.issues', { count: summary?.issueCount ?? 0 })}
            </Badge>
          </div>

          {summary && summary.commitCount === 0 && activeBase ? (
            <p className={styles.hint}>{t('projects.releaseDesk.noCommits')}</p>
          ) : null}

          <ul className={styles.gates} aria-label={t('projects.releaseDesk.gatesTitle')}>
            {(report.gates || []).map((gate) => (
              <li key={gate.name} className={styles.gate}>
                <span className={styles.gateName}>{gate.name}</span>
                <code className={styles.gateCommand}>{gate.verify_command}</code>
                <Badge tone={gateTone(gate.status)}>
                  {gate.status === 'verified'
                    ? t('projects.releaseDesk.gateVerified')
                    : t('projects.releaseDesk.gateUnverified')}
                </Badge>
              </li>
            ))}
          </ul>

          {report.approval ? (
            <p className={styles.approval}>
              <ShieldCheck size={14} aria-hidden="true" />
              {t('projects.releaseDesk.approval', {
                approver: report.approval.approver,
                id: report.approval.approval_id,
                date: formatApprovalDate(report.approval.approved_at, locale),
              })}
            </p>
          ) : (
            <p className={styles.noApproval}>{t('projects.releaseDesk.noApproval')}</p>
          )}

          {!summary?.mergePublishAllowed && (report.block_reasons || []).length > 0 ? (
            <div className={styles.reasons}>
              <span className={styles.reasonsTitle}>{t('projects.releaseDesk.reasonsTitle')}</span>
              <ul className={styles.reasonList}>
                {(report.block_reasons || []).map((reason) => (
                  <li key={reason}>{reason}</li>
                ))}
              </ul>
            </div>
          ) : null}
        </>
      ) : null}
    </Card>
  );
};
