import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { InlineAlert, Switch } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { CapacityPreferences } from '../../types';
import { cacheNotificationCategories } from '../../notifications/notificationPrefs';
import styles from './NotificationCategorySettings.module.scss';

const categoryFields = [
  ['notify_action_required', 'settings.notificationActionRequired'],
  ['notify_auth_permission', 'settings.notificationAuthPermission'],
  ['notify_critical_failure', 'settings.notificationCriticalFailure'],
  ['notify_quota_exhausted', 'settings.notificationQuotaExhausted'],
  ['notify_quota_low', 'settings.notificationQuotaLow'],
  ['notify_completion_recovery', 'settings.notificationCompletionRecovery'],
  ['notify_runtime_connection', 'settings.notificationRuntimeConnection'],
] as const;

type CategoryField = (typeof categoryFields)[number][0];

export const NotificationCategorySettings: React.FC = () => {
  const { t } = useTranslation();
  const [preferences, setPreferences] = useState<CapacityPreferences | null>(null);
  const [error, setError] = useState('');
  const [savingField, setSavingField] = useState<CategoryField | null>(null);

  const refresh = useCallback(async () => {
    try {
      setError('');
      const loaded = await nexus.getCapacityPreferences();
      setPreferences(loaded);
      cacheNotificationCategories({
        action_required: loaded.notify_action_required,
        auth_permission: loaded.notify_auth_permission,
        critical_failure: loaded.notify_critical_failure,
        quota_exhausted: loaded.notify_quota_exhausted,
        quota_low: loaded.notify_quota_low,
        completion_recovery: loaded.notify_completion_recovery,
        runtime_connection: loaded.notify_runtime_connection,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const update = async (field: CategoryField, checked: boolean) => {
    if (!preferences || savingField) return;
    setSavingField(field);
    setError('');
    try {
      const saved = await nexus.patchNotificationCategories({ [field]: checked });
      setPreferences(saved);
      cacheNotificationCategories({
        action_required: saved.notify_action_required,
        auth_permission: saved.notify_auth_permission,
        critical_failure: saved.notify_critical_failure,
        quota_exhausted: saved.notify_quota_exhausted,
        quota_low: saved.notify_quota_low,
        completion_recovery: saved.notify_completion_recovery,
        runtime_connection: saved.notify_runtime_connection,
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSavingField(null);
    }
  };

  return (
    <section
      className={styles.root}
      data-tour="notification-category-settings"
      aria-busy={savingField !== null}
      aria-label={t('settings.notificationCategoriesTitle', 'Pop-ups by category')}
    >
      <div className={styles.heading}>
        <h3>{t('settings.notificationCategoriesTitle', 'Pop-ups by category')}</h3>
        <p>
          {t(
            'settings.notificationCategoriesDescription',
            'Choose which events may open a system pop-up. Hidden pop-ups stay in the Attention Center.',
          )}
        </p>
      </div>
      {error ? (
        <InlineAlert tone="danger" title={t('common.error', 'Error')}>
          {error}
        </InlineAlert>
      ) : null}
      <div className={styles.categories}>
        {preferences
          ? categoryFields.map(([field, key]) => (
              <Switch
                key={field}
                checked={preferences[field]}
                onChange={(checked) => void update(field, checked)}
                label={t(key)}
              />
            ))
          : null}
      </div>
    </section>
  );
};
