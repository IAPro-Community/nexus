import React from 'react';
import { useTranslation } from 'react-i18next';
import type { NexV2EventEnvelope, NexV2HudState } from './nexV2Types';
import styles from './NexAttentionTab.module.scss';

export interface NexAttentionTabProps {
  hudState: NexV2HudState | null;
  events: NexV2EventEnvelope[];
  privacyMode: boolean;
  onAckAlert: (fingerprint: string) => void;
  onSnoozeAlert: (fingerprint: string) => void;
}

export const NexAttentionTab: React.FC<NexAttentionTabProps> = ({
  hudState,
  events,
  privacyMode,
  onAckAlert,
  onSnoozeAlert,
}) => {
  const { t } = useTranslation();
  const activeAlerts = (hudState?.alerts ?? []).filter((a) => !a.acked);
  const runtimeAttention = hudState?.runtime_attention ?? [];
  const missionsNeedingUser = (hudState?.missions ?? []).filter((m) => m.needs_user);

  const seenFingerprints = new Set<string>();
  activeAlerts.forEach((a) =>
    seenFingerprints.add(`alert:${a.fingerprint}:${(a.severity || '').toUpperCase()}`),
  );
  runtimeAttention.forEach((r) =>
    seenFingerprints.add(`runtime_attention:${r.runtime_id}:${r.fingerprint}`),
  );
  const modelEvents = events.filter((ev) => {
    if (!ev.kind.startsWith('MODEL_CATALOG_')) return false;
    if (seenFingerprints.has(ev.fingerprint)) return false;
    seenFingerprints.add(ev.fingerprint);
    return true;
  });

  const hasItems =
    activeAlerts.length > 0 ||
    runtimeAttention.length > 0 ||
    missionsNeedingUser.length > 0 ||
    modelEvents.length > 0;

  return (
    <div className={styles.stack} data-nex-attention>
      {!hasItems ? (
        <p className={styles.emptyText}>{t('nexV2.attentionEmpty')}</p>
      ) : (
        <>
          {activeAlerts.map((alert) => {
            const label =
              privacyMode && alert.profile
                ? `${alert.provider || alert.category} · ${t('nexV2.privacyMasked')}`
                : [alert.provider || alert.category, alert.profile].filter(Boolean).join(' · ');
            return (
              <div key={alert.fingerprint} className={styles.card}>
                <div className={styles.cardHeader}>
                  <strong>{label}</strong>
                  <span className={styles.statusBadge}>{alert.severity || 'INFO'}</span>
                </div>
                {alert.reason && <p className={styles.cardMeta}>{alert.reason}</p>}
                <div className={styles.cardActions}>
                  <button
                    type="button"
                    className={styles.smallBtn}
                    onClick={() => onAckAlert(alert.fingerprint)}
                  >
                    {t('nexV2.attentionAck')}
                  </button>
                  <button
                    type="button"
                    className={styles.smallBtn}
                    onClick={() => onSnoozeAlert(alert.fingerprint)}
                  >
                    {t('nexV2.attentionSnooze')}
                  </button>
                </div>
              </div>
            );
          })}

          {runtimeAttention.map((item) => (
            <div key={item.fingerprint} className={styles.card}>
              <div className={styles.cardHeader}>
                <strong>{t('nexV2.attentionRuntimeLabel', { provider: item.provider })}</strong>
                <span className={styles.statusBadge}>{item.prompt_kind}</span>
              </div>
              <p className={styles.cardMeta}>
                {privacyMode
                  ? t('nexV2.privacyMasked')
                  : item.project_name || item.project_id || ''}
                {item.reason ? ` · ${item.reason}` : ''}
              </p>
            </div>
          ))}

          {missionsNeedingUser.map((mission) => (
            <div key={mission.run_id} className={styles.card}>
              <div className={styles.cardHeader}>
                <strong>
                  {privacyMode
                    ? t('nexV2.privacyMasked')
                    : mission.current_task_title || mission.run_id}
                </strong>
                <span className={styles.statusBadge}>{t('nexV2.missionNeedsUserBadge')}</span>
              </div>
              {mission.intervention_reason && (
                <p className={styles.cardMeta}>{mission.intervention_reason}</p>
              )}
            </div>
          ))}

          {modelEvents.map((ev) => (
            <div key={ev.fingerprint} className={styles.card}>
              <div className={styles.cardHeader}>
                <strong>
                  {privacyMode ? `${ev.provider || ''} · ${t('nexV2.privacyMasked')}` : ev.title}
                </strong>
                <span className={styles.statusBadge}>{ev.kind}</span>
              </div>
              {ev.detail && <p className={styles.cardMeta}>{ev.detail}</p>}
            </div>
          ))}
        </>
      )}
    </div>
  );
};
