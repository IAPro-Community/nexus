import React from 'react';
import { useTranslation } from 'react-i18next';
import styles from './ComposerSurface.module.scss';

/**
 * Renders the prompt review hints (missing requirements and contradictions)
 * surfaced alongside the destinations dock. Renders nothing when both empty.
 */
export const ComposerPromptReview: React.FC<{
  missing: string[];
  contradictions: string[];
}> = ({ missing, contradictions }) => {
  const { t } = useTranslation();
  if (missing.length === 0 && contradictions.length === 0) return null;

  return (
    <div className={styles.suitabilityHint}>
      {missing.length > 0 && (
        <p className={styles.suitabilityReason}>
          {t('work.composer.promptReviewMissing', 'Missing')}: {missing.join(' · ')}
        </p>
      )}
      {contradictions.length > 0 && (
        <p className={styles.suitabilityReason}>
          {t('work.composer.promptReviewContradictions', 'Contradictions')}:{' '}
          {contradictions.join(' · ')}
        </p>
      )}
    </div>
  );
};
