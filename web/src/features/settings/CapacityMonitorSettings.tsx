import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Card, InlineAlert, Input, Switch } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { CapacityPreferences, CapacityStatus, PublicServiceStatus } from '../../types';
import { asArray } from '../../lib/safeArray';
import styles from './CapacityMonitorSettings.module.scss';
import { downloadAuthenticatedCsv } from '../../lib/downloadAuthenticatedCsv';
import { PublicProviderStatus } from './PublicProviderStatus';

const num = (raw: string, fallback: number): number => {
  const parsed = Number.parseFloat(raw);
  return Number.isFinite(parsed) ? parsed : fallback;
};

export const CapacityMonitorSettings: React.FC = () => {
  const { t } = useTranslation();
  const [prefs, setPrefs] = useState<CapacityPreferences | null>(null);
  const [status, setStatus] = useState<CapacityStatus | null>(null);
  const [publicStatus, setPublicStatus] = useState<PublicServiceStatus[]>([]);
  const [publicStatusUnavailable, setPublicStatusUnavailable] = useState(false);
  const [draft, setDraft] = useState({
    warning: '',
    critical: '',
    exhausted: '',
    recovery: '',
    reminder: '',
  });
  const [quietStart, setQuietStart] = useState('');
  const [quietEnd, setQuietEnd] = useState('');
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);
  const [saving, setSaving] = useState(false);

  const refresh = useCallback(async () => {
    // Service incidents are independent of account capacity: a quota API
    // error must not suppress public incident diagnostics.
    void nexus
      .getPublicServiceStatus()
      .then((rows) => {
        setPublicStatus(rows);
        setPublicStatusUnavailable(false);
      })
      .catch(() => {
        setPublicStatus([]);
        setPublicStatusUnavailable(true);
      });
    try {
      setError('');
      // Degraded quota reads cannot prevent editing capacity preferences or
      // viewing independently sourced public incident diagnostics.
      const [p, s] = await Promise.all([
        nexus.getCapacityPreferences(),
        nexus.getCapacityStatus().catch(() => null),
      ]);
      setPrefs(p);
      setStatus(s);
      if (s === null) {
        setError(
          t(
            'settings.capacityStatusFailed',
            'Quota data temporarily unavailable. Preferences are still editable.',
          ),
        );
      }
      setDraft({
        warning: String(p.warning_threshold),
        critical: String(p.critical_threshold),
        exhausted: String(p.exhausted_threshold),
        recovery: String(p.recovery_threshold),
        reminder: String(p.reminder_interval_seconds),
      });
      setQuietStart(p.quiet_hours_start_utc);
      setQuietEnd(p.quiet_hours_end_utc);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [t]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const patch = (next: Partial<CapacityPreferences>) => {
    setPrefs((prev) => (prev ? { ...prev, ...next } : prev));
    setSaved(false);
  };

  const save = async () => {
    if (!prefs) return;
    const warning = num(draft.warning, prefs.warning_threshold);
    const critical = num(draft.critical, prefs.critical_threshold);
    const exhausted = num(draft.exhausted, prefs.exhausted_threshold);
    const recovery = num(draft.recovery, prefs.recovery_threshold);
    if (!(warning > critical && critical > exhausted && recovery > critical)) {
      setError(
        t(
          'settings.capacityInvalidThresholds',
          'Thresholds must order warning > critical > exhausted, recovery above critical.',
        ),
      );
      return;
    }
    setSaving(true);
    try {
      setError('');
      const savedPrefs = await nexus.putCapacityPreferences({
        ...prefs,
        warning_threshold: warning,
        critical_threshold: critical,
        exhausted_threshold: exhausted,
        recovery_threshold: recovery,
        reminder_interval_seconds: Math.max(
          0,
          Math.trunc(num(draft.reminder, prefs.reminder_interval_seconds)),
        ),
        quiet_hours_start_utc: quietStart,
        quiet_hours_end_utc: quietEnd,
      });
      setPrefs(savedPrefs);
      setSaved(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  if (!prefs) {
    return (
      <Card className={styles.root}>
        <PublicProviderStatus
          publicStatus={publicStatus}
          publicStatusUnavailable={publicStatusUnavailable}
        />
        {error ? (
          <InlineAlert tone="danger" title={t('common.error', 'Error')}>
            {error}
          </InlineAlert>
        ) : (
          <p className={styles.empty}>
            {t('settings.capacityLoadError', 'Capacity status unavailable')}
          </p>
        )}
      </Card>
    );
  }

  const resources = status ? asArray(status.resources) : [];

  return (
    <Card className={styles.root}>
      <div className={styles.header}>
        <strong>{t('settings.capacity', 'Capacity Monitor')}</strong>
        <p>{t('settings.capacityDescription', 'Proactive quota alerts with hysteresis.')}</p>
      </div>
      <PublicProviderStatus
        publicStatus={publicStatus}
        publicStatusUnavailable={publicStatusUnavailable}
      />
      {error ? (
        <InlineAlert tone="danger" title={t('common.error', 'Error')}>
          {error}
        </InlineAlert>
      ) : null}
      {saved ? (
        <InlineAlert
          tone="success"
          title={t('settings.capacitySaved', 'Capacity preferences saved.')}
        >
          {t('settings.capacitySaved', 'Capacity preferences saved.')}
        </InlineAlert>
      ) : null}
      <Switch
        checked={prefs.enabled}
        onChange={(checked) => patch({ enabled: checked })}
        label={t('settings.capacityEnabled', 'Monitor enabled')}
        description={t(
          'settings.capacityEnabledDescription',
          'Collect observations and evaluate threshold states.',
        )}
      />
      <Switch
        checked={prefs.background_enabled}
        onChange={(checked) => patch({ background_enabled: checked })}
        label={t('settings.capacityBackground', 'Background monitoring')}
        description={t(
          'settings.capacityBackgroundDescription',
          'Host the monitor loop in the Nexus process.',
        )}
      />
      <section
        aria-label={t('settings.capacityThresholds', 'Thresholds')}
        className={styles.section}
      >
        <h3>{t('settings.capacityThresholds', 'Thresholds')}</h3>
        <div className={styles.grid}>
          <label>
            <span>{t('settings.capacityWarning', 'Warning below (%)')}</span>
            <Input
              value={draft.warning}
              onChange={(v) => setDraft((d) => ({ ...d, warning: v }))}
              inputMode="decimal"
            />
          </label>
          <label>
            <span>{t('settings.capacityCritical', 'Critical below (%)')}</span>
            <Input
              value={draft.critical}
              onChange={(v) => setDraft((d) => ({ ...d, critical: v }))}
              inputMode="decimal"
            />
          </label>
          <label>
            <span>{t('settings.capacityExhausted', 'Exhausted below (%)')}</span>
            <Input
              value={draft.exhausted}
              onChange={(v) => setDraft((d) => ({ ...d, exhausted: v }))}
              inputMode="decimal"
            />
          </label>
          <label>
            <span>{t('settings.capacityRecovery', 'Recovery above (%)')}</span>
            <Input
              value={draft.recovery}
              onChange={(v) => setDraft((d) => ({ ...d, recovery: v }))}
              inputMode="decimal"
            />
          </label>
          <label>
            <span>{t('settings.capacityReminder', 'Reminder interval (seconds)')}</span>
            <Input
              value={draft.reminder}
              onChange={(v) => setDraft((d) => ({ ...d, reminder: v }))}
              inputMode="numeric"
            />
          </label>
        </div>
      </section>
      <section
        aria-label={t('settings.capacityQuietHours', 'Quiet hours (UTC)')}
        className={styles.section}
      >
        <h3>{t('settings.capacityQuietHours', 'Quiet hours (UTC)')}</h3>
        <Switch
          checked={prefs.quiet_hours_enabled}
          onChange={(checked) => patch({ quiet_hours_enabled: checked })}
          label={t('settings.capacityQuietHours', 'Quiet hours (UTC)')}
        />
        <div className={styles.grid}>
          <label>
            <span>{t('settings.capacityQuietStart', 'Start')}</span>
            <Input value={quietStart} onChange={setQuietStart} placeholder="22:00" mono />
          </label>
          <label>
            <span>{t('settings.capacityQuietEnd', 'End')}</span>
            <Input value={quietEnd} onChange={setQuietEnd} placeholder="08:00" mono />
          </label>
        </div>
        <Switch
          checked={prefs.exhausted_bypass_quiet_hours}
          onChange={(checked) => patch({ exhausted_bypass_quiet_hours: checked })}
          label={t('settings.capacityExhaustedBypass', 'Exhausted alerts bypass quiet hours')}
        />
      </section>
      <section aria-label={t('settings.capacityChannels', 'Channels')} className={styles.section}>
        <h3>{t('settings.capacityChannels', 'Channels')}</h3>
        <Switch
          checked={prefs.browser_notifications_enabled}
          onChange={(checked) => patch({ browser_notifications_enabled: checked })}
          label={t('settings.capacityBrowser', 'Browser notifications')}
        />
        <Switch
          checked={prefs.native_notifications_enabled}
          onChange={(checked) => patch({ native_notifications_enabled: checked })}
          label={t('settings.capacityNative', 'Native notifications')}
        />
        <Switch
          checked={prefs.lock_screen_safe_copy}
          onChange={(checked) => patch({ lock_screen_safe_copy: checked })}
          label={t('settings.capacityLockSafe', 'Lock-screen-safe alert copy')}
        />
      </section>
      <div className={styles.actions}>
        <Button
          onClick={() =>
            void downloadAuthenticatedCsv(
              '/api/v1/capacity/pace.csv',
              'nexus-capacity-pace.csv',
            ).catch((err) => setError(err instanceof Error ? err.message : String(err)))
          }
        >
          {t('settings.capacityExportPace', 'Exportar histórico de consumo (CSV)')}
        </Button>
        <Button tone="brand" onClick={() => void save()} disabled={saving}>
          {saving
            ? t('settings.capacitySaving', 'Saving…')
            : t('settings.capacitySave', 'Save Capacity Monitor')}
        </Button>
        {status ? (
          <Badge tone={resources.length > 0 ? 'info' : 'default'}>
            {status.lease_owner
              ? t('settings.capacityLeader', 'Monitor leader: {{owner}}').replace(
                  '{{owner}}',
                  status.lease_owner,
                )
              : t('settings.capacityNoData', 'No capacity observations yet.')}
          </Badge>
        ) : null}
      </div>
    </Card>
  );
};
