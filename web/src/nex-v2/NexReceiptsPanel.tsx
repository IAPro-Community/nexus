import React from 'react';
import { useTranslation } from 'react-i18next';
import type { NexV2ActionReceipt } from './nexV2Types';
import styles from './NexReceiptsPanel.module.scss';

export interface NexReceiptsPanelProps {
  receipts: NexV2ActionReceipt[];
  privacyMode: boolean;
  runningAction: boolean;
  onRunReadHud: () => void;
}

export const NexReceiptsPanel: React.FC<NexReceiptsPanelProps> = ({
  receipts,
  privacyMode,
  runningAction,
  onRunReadHud,
}) => {
  const { t } = useTranslation();
  const ordered = [...receipts].reverse();

  return (
    <section
      className={styles.receiptsBox}
      aria-label={t('nexV2.receiptsToggleLabel')}
      data-nex-receipts-panel
    >
      <div className={styles.headerRow}>
        <button
          type="button"
          className={styles.actionBtn}
          disabled={runningAction}
          data-nex-action-read-hud
          onClick={onRunReadHud}
        >
          {t('nexV2.actionReadHud')}
        </button>
      </div>

      {ordered.length === 0 ? (
        <p className={styles.emptyText}>{t('nexV2.receiptsEmpty')}</p>
      ) : (
        <ul className={styles.receiptList} data-nex-receipts-list>
          {ordered.map((receipt) => (
            <li
              key={receipt.receipt_id}
              className={styles.receiptItem}
              data-nex-receipt={receipt.receipt_id}
              data-receipt-status={receipt.status}
            >
              <div className={styles.receiptHeader}>
                <strong>{receipt.action_id}</strong>
                <span className={styles.statusBadge}>
                  {t('nexV2.receiptStatusLabel', { status: receipt.status })}
                </span>
              </div>
              {receipt.summary && (
                <p className={styles.receiptSummary}>
                  {privacyMode ? t('nexV2.privacyMasked') : receipt.summary}
                </p>
              )}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
};
