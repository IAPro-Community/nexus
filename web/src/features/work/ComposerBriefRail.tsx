import React, { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { Badge, Card } from '../../design-system';
import type { ComposerSessionView, PromptReadinessCheck } from '../../types';
import { asArray } from '../../lib/safeArray';
import styles from './ComposerSurface.module.scss';

type BriefUnknown = NonNullable<ComposerSessionView['brief']['unknowns']>[number];

/**
 * Right-hand rail summarizing the live briefing, next question, inferred
 * assumptions, readiness checks and remaining open gaps.
 */
export const ComposerBriefRail: React.FC<{
  view: ComposerSessionView;
  briefItems: Array<[string, string]>;
  nextQuestion: string;
  actionableGaps: BriefUnknown[];
}> = ({ view, briefItems, nextQuestion, actionableGaps }) => {
  const { t } = useTranslation();

  const inferredPremises = useMemo(
    () =>
      asArray<BriefUnknown>(view.brief.unknowns).filter(
        (u) => u.status === 'INFERRED' && u.inferred_value,
      ),
    [view.brief.unknowns],
  );

  const readinessScore = view.brief.readiness?.score ?? 0;
  const readinessState = view.brief.readiness?.state || 'UNKNOWN';
  const readyEnough =
    readinessState !== 'BLOCKED' &&
    (readinessState === 'READY' || readinessState === 'READY_WITH_ASSUMPTIONS');
  const checks = asArray<PromptReadinessCheck>(view.brief.readiness?.checks);

  return (
    <aside className={styles.rail}>
      <Card className={styles.railCard}>
        <strong>{t('work.composer.liveBrief', 'Live briefing')}</strong>
        <div className={styles.flexColGap10}>
          {briefItems.length > 0 ? (
            briefItems.map(([label, value]) => (
              <div key={label}>
                <small className={styles.mutedLabel}>{label}</small>
                <p className={styles.briefValue}>{value}</p>
              </div>
            ))
          ) : (
            <p className={`${styles.mutedCopy} ${styles.briefEmptyHint}`}>
              {t(
                'work.composer.briefEmpty',
                'The live briefing synthesizes goal, context, criteria and decisions as the conversation advances.',
              )}
            </p>
          )}
        </div>
      </Card>

      {nextQuestion ? (
        <Card className={styles.railCard}>
          <strong>{t('work.composer.nextQuestion', 'Next question')}</strong>
          <p className={styles.nextQuestionText}>{nextQuestion}</p>
          <small className={styles.mutedLabel}>
            {t(
              'work.composer.answerInThread',
              'Answer in the conversation thread with the options shown there.',
            )}
          </small>
        </Card>
      ) : null}

      {inferredPremises.length > 0 ? (
        <Card className={styles.railCard}>
          <strong>{t('work.assumedPremise', 'Nexus assumed — edit')}</strong>
          <div className={styles.flexColGap6}>
            {inferredPremises.slice(0, 4).map((item) => (
              <p key={item.id} className={styles.briefValue}>
                {item.inferred_value}
              </p>
            ))}
          </div>
        </Card>
      ) : null}

      <Card className={styles.railCard}>
        <div className={styles.readinessCompact}>
          <strong>{t('work.composer.readiness', 'Readiness')}</strong>
          <Badge tone={readyEnough ? 'success' : 'warning'}>
            {readinessScore}% ·{' '}
            {t(`work.composer.readinessState.${readinessState}`, readinessState)}
          </Badge>
        </div>
        <p className={styles.readinessSummary}>
          {view.brief.readiness?.summary || t('work.composer.readinessEmpty', 'No assessment yet.')}
        </p>
        {checks.length > 0 && (
          <div className={`${styles.flexColGap6} ${styles.readinessSection}`}>
            {checks.map((check) => (
              <div key={check.key} className={styles.readinessCheckItem}>
                <div className={styles.readinessCheckContent}>
                  <span className={styles.readinessCheckLabel}>
                    {t(`work.composer.readinessChecks.${check.key}`, check.label || check.key)}
                  </span>
                </div>
                <span
                  className={
                    check.status === 'COMPLETE'
                      ? styles.readinessCheckStatusComplete
                      : check.status === 'MISSING'
                        ? styles.readinessCheckStatusMissing
                        : styles.readinessCheckStatus
                  }
                >
                  {t(`work.composer.readinessCheckStatus.${check.status}`, check.status)}
                </span>
              </div>
            ))}
          </div>
        )}
      </Card>

      <Card className={styles.railCard}>
        <strong>{t('work.composer.gaps', 'Open refinements')}</strong>
        {actionableGaps.length === 0 ? (
          <p className={styles.mutedCopy}>{t('work.composer.noGaps', 'No open gaps.')}</p>
        ) : (
          <ul className={styles.flexColGap6}>
            {actionableGaps.map((unknown) => (
              <li key={unknown.id} className={styles.unknownQuestion}>
                <Badge tone={unknown.severity === 'BLOCKING' ? 'danger' : 'warning'}>
                  {t(`work.composer.severity.${unknown.severity}`, unknown.severity)}
                </Badge>{' '}
                {unknown.question}
              </li>
            ))}
          </ul>
        )}
      </Card>
    </aside>
  );
};
