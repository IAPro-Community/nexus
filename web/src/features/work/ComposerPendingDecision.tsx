import React from 'react';
import { useTranslation } from 'react-i18next';
import { AlertTriangle } from 'lucide-react';
import styles from './ComposerInspector.module.scss';

export const ComposerPendingDecision: React.FC<{
  count: number;
  onFocus?: () => void;
}> = ({ count, onFocus }) => {
  const { t } = useTranslation();
  if (count <= 0) return null;
  return (
    <div className={styles.pendingBar} role="status">
      <AlertTriangle size={14} aria-hidden />
      <div>
        <strong>
          {t('work.composer.pendingDecision', '{{count}} pending decisions', { count })}
        </strong>
        <p>{t('work.composer.pendingDecisionHint', 'Answer in the conversation to continue.')}</p>
      </div>
      {onFocus ? (
        <button type="button" className={styles.pendingAction} onClick={onFocus}>
          {t('work.composer.answerToContinue', 'Answer to continue')}
        </button>
      ) : null}
    </div>
  );
};
