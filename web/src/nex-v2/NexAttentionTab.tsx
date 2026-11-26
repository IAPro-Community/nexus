import React from 'react';
import { useTranslation } from 'react-i18next';
import { CheckCircle2 } from 'lucide-react';
import type { NexV2EventEnvelope, NexV2HudState } from './nexV2Types';
import styles from './NexAttentionTab.module.scss';

export interface NexAttentionTabProps {
  hudState: NexV2HudState | null;
  events: NexV2EventEnvelope[];
  privacyMode: boolean;
  onAckAlert: (fingerprint: string) => void;
  onSnoozeAlert: (fingerprint: string) => void;
}

function resolveSeverityLabel(severity: string | undefined, t: (k: string) => string): string {
  const norm = (severity || 'INFO').toUpperCase();
  if (norm === 'CRITICAL') return t('nexV2.severityCritical');
  if (norm === 'WARNING') return t('nexV2.severityWarning');
  return t('nexV2.severityInfo');
}

function resolvePromptKindLabel(promptKind: string | undefined, t: (k: string) => string): string {
  const norm = (promptKind || '').toLowerCase();
  if (norm === 'yn' || norm === 'yes_no') return t('nexV2.promptKindYn');
  if (norm === 'input' || norm === 'text') return t('nexV2.promptKindInput');
  return t('nexV2.promptKindDefault');
}

function resolveModelEventLabel(kind: string, t: (k: string) => string): string {
  const norm = kind.toUpperCase();
  if (norm === 'MODEL_CATALOG_ADDED') return t('nexV2.modelChangeAdded');
  if (norm === 'MODEL_CATALOG_DEPRECATED') return t('nexV2.modelChangeDeprecated');
  if (norm === 'MODEL_CATALOG_REMOVED') return t('nexV2.modelChangeRemoved');
  return kind;
}

export const NexAttentionTab: React.FC<NexAttentionTabProps> = ({
  hudState,
  events,
  privacyMode,
  onAckAlert,
  onSnoozeAlert,
}) => {
  const { t } = useTranslation();
  const activeAlerts = [...(hudState?.alerts ?? []).filter((a) => !a.acked)].sort((a, b) => {
    const rank = (s?: string) => ((s || '').toUpperCase() === 'CRITICAL' ? 0 : 1);
    return rank(a.severity) - rank(b.severity);
  });
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

  const hasUrgentItems =
    activeAlerts.length > 0 || runtimeAttention.length > 0 || missionsNeedingUser.length > 0;
  const hasAnyItems = hasUrgentItems || modelEvents.length > 0;

  return (
    <div className={styles.stack} data-nex-attention>
      {!hasAnyItems ? (
        <div className={styles.emptyState}>
          <CheckCircle2 size={24} className={styles.emptyIcon} aria-hidden="true" />
          <strong className={styles.emptyTitle}>{t('nexV2.attentionEmptyTitle')}</strong>
          <p className={styles.emptyText}>{t('nexV2.attentionEmpty')}</p>
        </div>
      ) : (
        <>
          {hasUrgentItems && (
            <section className={styles.group}>
              <h3 className={styles.sectionLabel}>{t('nexV2.sectionActionRequired')}</h3>

              {activeAlerts.map((alert) => {
                const severityNorm = (alert.severity || 'INFO').toUpperCase();
                const label =
                  privacyMode && alert.profile
                    ? `${alert.provider || alert.category} · ${t('nexV2.privacyMasked')}`
                    : [alert.provider || alert.category, alert.profile].filter(Boolean).join(' · ');
                return (
                  <div key={alert.fingerprint} className={styles.card} data-severity={severityNorm}>
                    <div className={styles.cardHeader}>
                      <strong>{label}</strong>
                      <span className={styles.severityPill} data-severity={severityNorm}>
                        {resolveSeverityLabel(severityNorm, t)}
                      </span>
                    </div>
                    {alert.reason && <p className={styles.cardMeta}>{alert.reason}</p>}
                    <div className={styles.cardActions}>
                      <button
                        type="button"
                        className={styles.primaryBtn}
                        onClick={() => onAckAlert(alert.fingerprint)}
                      >
                        {t('nexV2.attentionAck')}
                      </button>
                      <button
                        type="button"
                        className={styles.secondaryBtn}
                        onClick={() => onSnoozeAlert(alert.fingerprint)}
                      >
                        {t('nexV2.attentionSnooze')}
                      </button>
                    </div>
                  </div>
                );
              })}

              {runtimeAttention.map((item) => (
                <div key={item.fingerprint} className={styles.card} data-severity="WARNING">
                  <div className={styles.cardHeader}>
                    <strong>{t('nexV2.attentionRuntimeLabel', { provider: item.provider })}</strong>
                    <span
                      className={styles.severityPill}
                      data-severity="WARNING"
                      data-prompt-kind={item.prompt_kind}
                    >
                      {resolvePromptKindLabel(item.prompt_kind, t)}
                    </span>
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
                <div key={mission.run_id} className={styles.card} data-severity="WARNING">
                  <div className={styles.cardHeader}>
                    <strong>
                      {privacyMode
                        ? t('nexV2.privacyMasked')
                        : mission.current_task_title || mission.run_id}
                    </strong>
                    <span className={styles.severityPill} data-severity="WARNING">
                      {t('nexV2.missionNeedsUserBadge')}
                    </span>
                  </div>
                  {mission.intervention_reason && (
                    <p className={styles.cardMeta}>{mission.intervention_reason}</p>
                  )}
                </div>
              ))}
            </section>
          )}

          {modelEvents.length > 0 && (
            <section className={styles.group}>
              <h3 className={styles.sectionLabel}>{t('nexV2.sectionUpdates')}</h3>
              {modelEvents.map((ev) => (
                <div key={ev.fingerprint} className={styles.updateRow} data-event-kind={ev.kind}>
                  <div className={styles.cardHeader}>
                    <span className={styles.updateTitle}>
                      {privacyMode
                        ? `${ev.provider || ''} · ${t('nexV2.privacyMasked')}`
                        : ev.title}
                    </span>
                    <span className={styles.infoPill}>{resolveModelEventLabel(ev.kind, t)}</span>
                  </div>
                  {ev.detail && <p className={styles.cardMeta}>{ev.detail}</p>}
                </div>
              ))}
            </section>
          )}
        </>
      )}
    </div>
  );
};
