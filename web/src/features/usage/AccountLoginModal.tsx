import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Check, Copy, KeyRound, RefreshCw, Terminal } from 'lucide-react';
import { toast } from 'sonner';
import { Button, Dialog } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ProviderAccount } from '../../types';
import styles from './AccountLoginModal.module.scss';

export interface AccountLoginModalProps {
  account: ProviderAccount | null;
  isOpen: boolean;
  onClose: () => void;
  onLoginSuccess: () => Promise<void>;
  onStartTerminalLogin?: (provider: string, profile: string) => void;
}

export const AccountLoginModal: React.FC<AccountLoginModalProps> = ({
  account,
  isOpen,
  onClose,
  onLoginSuccess,
  onStartTerminalLogin,
}) => {
  const { t } = useTranslation();
  const [copied, setCopied] = useState(false);
  const [verifying, setVerifying] = useState(false);

  if (!account || !isOpen) return null;

  const provider = account.provider || 'codex';
  const profile = account.profile || 'default';
  const cliCommand = `nexus login ${provider} ${profile}`;

  const copyCommand = async () => {
    try {
      await navigator.clipboard.writeText(cliCommand);
      setCopied(true);
      toast.success(t('common.copied', 'Copiado para a área de transferência'));
      setTimeout(() => setCopied(false), 2500);
    } catch {
      toast.error(t('common.copyFailed', 'Falha ao copiar'));
    }
  };

  const getInstructions = () => {
    switch (provider.toLowerCase()) {
      case 'codex':
        return t('usage.loginInstCodex');
      case 'agy':
        return t('usage.loginInstAgy');
      case 'claude':
        return t('usage.loginInstClaude');
      case 'gemini':
        return t('usage.loginInstGemini');
      case 'opencode':
        return t('usage.loginInstOpencode');
      case 'cursor':
        return t('usage.loginInstCursor');
      default:
        return t('usage.loginDialogSubtitle');
    }
  };

  const handleVerify = async () => {
    setVerifying(true);
    try {
      await nexus.refreshResources(provider, profile);
      await onLoginSuccess();
      toast.success(t('usage.refreshSuccess', 'Cota e autenticação verificadas com sucesso!'));
      onClose();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setVerifying(false);
    }
  };

  const handleTerminalLogin = () => {
    if (onStartTerminalLogin) {
      onStartTerminalLogin(provider, profile);
      onClose();
    } else {
      window.dispatchEvent(
        new CustomEvent('nexus:terminal-launch', {
          detail: { provider, profile, args: ['login'], title: `Login · ${provider}:${profile}` },
        }),
      );
      onClose();
    }
  };

  return (
    <Dialog
      open={isOpen}
      onClose={onClose}
      title={t('usage.loginDialogTitle', { provider: `${provider} (${profile})` })}
    >
      <div className={styles.root}>
        <p className={styles.intro}>{t('usage.loginDialogSubtitle')}</p>

        <div className={styles.instructionBox}>
          <KeyRound size={18} className={styles.instructionIcon} />
          <div>
            <strong>{t('usage.loginInstructionsTitle')}</strong>
            <span>{getInstructions()}</span>
          </div>
        </div>

        <div className={styles.actionsCard}>
          <div className={styles.terminalOption}>
            <div className={styles.optionCopy}>
              <strong>{t('usage.loginTerminalAction')}</strong>
              <small>{t('usage.loginTerminalDesc')}</small>
            </div>
            <Button tone="brand" size="sm" onClick={handleTerminalLogin}>
              <Terminal size={14} />
              {t('usage.loginTerminalAction')}
            </Button>
          </div>

          <div className={styles.cliOption}>
            <span className={styles.cliLabel}>{t('usage.loginCliAction')}</span>
            <div className={styles.cliCommand}>
              <code>{cliCommand}</code>
              <Button size="sm" onClick={() => void copyCommand()}>
                {copied ? <Check size={13} /> : <Copy size={13} />}
                {copied ? t('common.copied', 'Copiado') : t('common.copy', 'Copiar')}
              </Button>
            </div>
          </div>
        </div>

        <div className={styles.footerActions}>
          <Button onClick={onClose}>{t('common.close', 'Fechar')}</Button>
          <Button tone="success" disabled={verifying} onClick={() => void handleVerify()}>
            <RefreshCw size={13} className={verifying ? 'nx-spin' : ''} />
            {verifying ? t('usage.refreshing', 'Verificando…') : t('usage.loginVerifyCta')}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
