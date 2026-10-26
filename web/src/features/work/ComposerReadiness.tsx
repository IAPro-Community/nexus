import React from 'react';
import { useTranslation } from 'react-i18next';
import type { ComposerSessionView } from '../../types';
import { composerBriefStats } from './composerModel';
import styles from './ComposerInspector.module.scss';

export const ComposerReadiness: React.FC<{
  view: ComposerSessionView;
  onFocusGaps?: () => void;
}> = ({ view, onFocusGaps }) => {
  const { t } = useTranslation();
  // A percentage based on archetype checklist fields is misleading: it
  // rewards answering generic questions rather than validating a useful plan.
  const readinessState = String(view.brief.readiness?.state || 'NEEDS_INFORMATION').toUpperCase();
  const stats = composerBriefStats(view);
  const items = [
    {
      id: 'decisions',
      label: t('work.composer.readinessStats.decisions', 'Decisions'),
      value: t(
        'work.composer.readinessStats.definedPending',
        '{{defined}} defined · {{pending}} pending',
        {
          defined: stats.decisionsDefined,
          pending: stats.decisionsPending,
        },
      ),
      action: stats.decisionsPending > 0 ? onFocusGaps : undefined,
    },
    {
      id: 'premises',
      label: t('work.composer.readinessStats.premises', 'Premises'),
      value: t('work.composer.readinessStats.inferred', '{{count}} inferred', {
        count: stats.premisesInferred,
      }),
    },
    {
      id: 'risks',
      label: t('work.composer.readinessStats.risks', 'Risks'),
      value: t('work.composer.readinessStats.identified', '{{count}} identified', {
        count: stats.risksIdentified,
      }),
    },
    {
      id: 'doubts',
      label: t('work.composer.readinessStats.doubts', 'Open questions'),
      value: t('work.composer.readinessStats.pendingCount', '{{count}} pending', {
        count: stats.openDoubts,
      }),
      action: stats.openDoubts > 0 ? onFocusGaps : undefined,
    },
  ];

  return (
    <section className={styles.group} aria-labelledby="composer-readiness">
      <div className={styles.readinessHead}>
        <h3 id="composer-readiness" className={styles.groupTitle}>
          {t('work.composer.readiness', 'Readiness')}
        </h3>
        <strong className={styles.score}>
          {t(`work.composer.readinessState.${readinessState}`, readinessState.replaceAll('_', ' '))}
        </strong>
      </div>
      <p className={styles.summary}>
        {view.brief.readiness?.summary ||
          t('work.composer.readinessEmpty', 'O plano ainda está em elaboração.')}
      </p>
      <ul className={styles.statList}>
        {items.map((item) => (
          <li key={item.id}>
            {item.action ? (
              <button type="button" className={styles.statButton} onClick={item.action}>
                <span>{item.label}</span>
                <small>{item.value}</small>
              </button>
            ) : (
              <div className={styles.statRow}>
                <span>{item.label}</span>
                <small>{item.value}</small>
              </div>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
};
