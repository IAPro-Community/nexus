import React from 'react';
import { useTranslation } from 'react-i18next';
import type { PublicServiceStatus } from '../../types';
import styles from './CapacityMonitorSettings.module.scss';

interface PublicProviderStatusProps {
  publicStatus: PublicServiceStatus[];
  publicStatusUnavailable: boolean;
}

/** A provider's public incident status must be shown independently of quotas. */
export const PublicProviderStatus: React.FC<PublicProviderStatusProps> = ({
  publicStatus,
  publicStatusUnavailable,
}) => {
  const { t } = useTranslation();
  return (
    <>
      {publicStatus.length > 0 ? (
        <section className={styles.section} aria-label={t('settings.publicStatus', 'Incidentes públicos dos provedores')}>
          <h3>{t('settings.publicStatus', 'Incidentes públicos dos provedores')}</h3>
          <div className={styles.statusList}>
            {publicStatus.map((item) => (
              <div className={styles.statusItem} key={item.provider}>
                <strong>{item.provider}</strong>
                <span data-indicator={item.indicator}>
                  {item.stale
                    ? t('settings.publicStatusUnknown', 'Indisponível ou desatualizado')
                    : item.indicator === 'none'
                      ? t('settings.publicStatusHealthy', 'Sem incidentes reportados')
                      : item.detail || item.indicator}
                </span>
              </div>
            ))}
          </div>
          <small>{t('settings.publicStatusHint', 'Estado público do serviço; não representa a quota da sua conta.')}</small>
        </section>
      ) : null}

      {publicStatusUnavailable ? (
        <p className={styles.empty}>
          {t('settings.publicStatusFetchFailed', 'Estado público indisponível. Isso não indica falha da sua quota.')}
        </p>
      ) : null}
    </>
  );
};
