import React from 'react';
import { useTranslation } from 'react-i18next';
import { Card } from '../../design-system';
import styles from './UsagePolicyPanel.module.scss';

export const UsagePolicyPanel: React.FC<{ policy: string }> = ({ policy }) => {
  const { t } = useTranslation();
  return (
    <Card className={styles.card}>
      <strong>{t('surfaces.allocation')}</strong>
      <p>{t('surfaces.allocationBody')}</p>
      <small>
        {t('usage.activePolicy', 'Política ativa')}: <code>{policy || 'BALANCED'}</code>
      </small>
    </Card>
  );
};
