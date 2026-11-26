import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Check, Copy, Play, RefreshCw } from 'lucide-react';
import type { NexV2ChatRecovery } from './nexV2Types';
import { ensureNexCoreConnection } from './nexV2Client';
import styles from './NexChatRecoveryCard.module.scss';

export interface NexChatRecoveryCardProps {
  recovery: NexV2ChatRecovery;
  onRecovered: () => Promise<void> | void;
}

/**
 * Inline zero-friction recovery actions rendered inside Nex chat turns when
 * the local Core is offline, needs re-authentication, or returned a transport error.
 */
export const NexChatRecoveryCard: React.FC<NexChatRecoveryCardProps> = ({
  recovery,
  onRecovered,
}) => {
  const { t } = useTranslation();
  const [busy, setBusy] = useState(false);
  const [copied, setCopied] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  const handleReconnect = async () => {
    if (busy) return;
    setBusy(true);
    setErrorMsg('');
    const result = await ensureNexCoreConnection();
    setBusy(false);
    if (!result.ok) {
      setErrorMsg(result.error || t('nexV2.recoveryStillOffline', { cmd: recovery.command }));
      return;
    }
    await onRecovered();
  };

  const handleCopyCommand = async () => {
    try {
      await navigator.clipboard?.writeText(recovery.command);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      setCopied(false);
    }
  };

  return (
    <section
      className={styles.card}
      aria-label={t('nexV2.recoveryTitle')}
      data-nex-chat-recovery={recovery.kind}
    >
      <strong className={styles.title}>{t('nexV2.recoveryTitle')}</strong>
      <p className={styles.detail}>
        {recovery.kind === 'auth_required'
          ? t('nexV2.recoveryHintAuth')
          : recovery.kind === 'core_error'
            ? t('nexV2.recoveryHintError', {
                status: recovery.status ?? 500,
                detail: recovery.detail || '-',
              })
            : t('nexV2.recoveryHintOffline')}
      </p>

      <div className={styles.commandRow}>
        <code className={styles.cmd} data-nex-recovery-cmd>
          {recovery.command}
        </code>
        <button
          type="button"
          className={styles.secondaryBtn}
          aria-label={t('nexV2.recoveryCopyCmdBtn')}
          data-nex-recovery-copy
          onClick={() => void handleCopyCommand()}
        >
          {copied ? <Check size={12} aria-hidden="true" /> : <Copy size={12} aria-hidden="true" />}
          <span>{copied ? t('nexV2.recoveryCopied') : t('nexV2.recoveryCopyCmdBtn')}</span>
        </button>
      </div>

      <div className={styles.actionsRow}>
        <button
          type="button"
          className={styles.primaryBtn}
          disabled={busy}
          data-nex-recovery-start
          onClick={() => void handleReconnect()}
        >
          {recovery.canStartCore ? (
            <Play size={12} aria-hidden="true" />
          ) : (
            <RefreshCw size={12} aria-hidden="true" />
          )}
          <span>
            {busy
              ? t('nexV2.recoveryWorking')
              : recovery.canStartCore
                ? t('nexV2.recoveryStartCoreBtn')
                : t('nexV2.recoveryReconnectBtn')}
          </span>
        </button>

        <button
          type="button"
          className={styles.secondaryBtn}
          disabled={busy}
          data-nex-recovery-retry
          onClick={() => void onRecovered()}
        >
          <span>{t('nexV2.recoveryRetryOnlyBtn')}</span>
        </button>
      </div>

      {errorMsg && (
        <p className={styles.error} role="alert" data-nex-recovery-error>
          {errorMsg}
        </p>
      )}
    </section>
  );
};
