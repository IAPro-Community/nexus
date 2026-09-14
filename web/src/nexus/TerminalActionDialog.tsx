import React from 'react';
import { useTranslation } from 'react-i18next';
import { AlertTriangle, RotateCcw, Square } from 'lucide-react';
import { Button, Dialog } from '../design-system';

export const TerminalActionDialog: React.FC<{
  mode?: 'Safe' | 'YOLO';
  close?: boolean;
  shell?: boolean;
  busy?: boolean;
  onCancel: () => void;
  onConfirmMode?: () => void;
  onCloseTab?: () => void;
  onStopRuntime?: () => void;
  promotion?: string;
  onPromotionAnswer?: (answer: 'y' | 'n') => void;
}> = ({
  mode,
  close,
  shell,
  busy,
  onCancel,
  onConfirmMode,
  onCloseTab,
  onStopRuntime,
  promotion,
  onPromotionAnswer,
}) => {
  const { t } = useTranslation();
  if (promotion) {
    return (
      <Dialog open title={t('terminal.promotion.title')} onClose={() => onPromotionAnswer?.('n')}>
        <div className="nx-terminal-action-dialog">
          <strong>{t('terminal.promotion.question')}</strong>
          <p>{promotion}</p>
          <div className="nx-dialog-actions">
            <Button tone="brand" onClick={() => onPromotionAnswer?.('y')}>
              {t('terminal.promotion.promote')}
            </Button>
            <Button onClick={() => onPromotionAnswer?.('n')}>{t('terminal.promotion.keep')}</Button>
          </div>
        </div>
      </Dialog>
    );
  }
  if (!mode && !close) return null;
  if (mode) {
    return (
      <Dialog open title={t('terminal.applyModeTitle', { mode })} onClose={onCancel}>
        <div className="nx-terminal-action-dialog">
          <div
            className="nx-terminal-action-dialog__icon"
            data-tone={mode === 'YOLO' ? 'danger' : 'warning'}
          >
            <RotateCcw size={18} />
          </div>
          <div>
            <strong>{t('terminal.runtimeWillRestart')}</strong>
            <p>{t('terminal.runtimeRestartDesc', { mode })}</p>
          </div>
          <div className="nx-dialog-actions">
            <Button onClick={onCancel} disabled={busy}>
              {t('common.cancel')}
            </Button>
            <Button
              tone={mode === 'YOLO' ? 'danger' : 'brand'}
              onClick={onConfirmMode}
              disabled={busy}
            >
              {busy ? t('terminal.applying') : t('terminal.applyModeAndRestart', { mode })}
            </Button>
          </div>
        </div>
      </Dialog>
    );
  }
  return (
    <Dialog
      open
      title={shell ? t('terminal.closeShellTitle') : t('terminal.closeTerminalTitle')}
      onClose={onCancel}
    >
      <div className="nx-terminal-action-dialog">
        <div className="nx-terminal-action-dialog__icon" data-tone="warning">
          <AlertTriangle size={18} />
        </div>
        <div>
          <strong>
            {shell ? t('terminal.shellProcessWillEnd') : t('terminal.whatDoYouWantToClose')}
          </strong>
          <p>{shell ? t('terminal.closeShellDesc') : t('terminal.closeTabDesc')}</p>
        </div>
        <div className="nx-terminal-action-dialog__choices">
          {!shell && (
            <Button onClick={onCloseTab} disabled={busy}>
              {t('terminal.closeTabOnly')}
            </Button>
          )}
          <Button tone="danger" onClick={onStopRuntime} disabled={busy}>
            <Square size={13} />
            {busy
              ? t('terminal.stopping')
              : shell
                ? t('terminal.closeAndEndProcess')
                : t('terminal.closeTabAndStopRuntime')}
          </Button>
          <Button onClick={onCancel} disabled={busy}>
            {t('common.cancel')}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
