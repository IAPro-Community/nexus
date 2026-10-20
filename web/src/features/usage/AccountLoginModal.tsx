import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Check, Copy, Cpu, Globe, KeyRound, RefreshCw, Sparkles, Terminal } from 'lucide-react';
import { toast } from 'sonner';
import { Button, Dialog } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ProviderAccount } from '../../types';
import { isProviderLoginVerified } from './providerLoginVerification';
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
  const isOpencode = provider.toLowerCase() === 'opencode';
  const cliCommand = `nexus login ${provider} ${profile}`;

  const activeProvider =
    account.quota_view?.account ||
    (account.quota_view?.plan?.includes('(')
      ? account.quota_view.plan.replace(/OpenCode\s*\((.*?)\)/, '$1')
      : null);

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
      const updated = await nexus.refreshResources(provider, profile);
      if (!isProviderLoginVerified(updated.accounts || [], provider, profile)) {
        toast.error(
          'Login ainda não confirmado para esta conta. Conclua o login oficial e tente novamente.',
        );
        return;
      }
      await onLoginSuccess();
      toast.success('Autenticação confirmada. A disponibilidade de cota é verificada separadamente.');
      onClose();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setVerifying(false);
    }
  };

  const handleTerminalLogin = () => {
    if (!onStartTerminalLogin) return;
    // The project shell opens separately; the user controls the official login.
    void navigator.clipboard?.writeText(cliCommand).catch(() => undefined);
    onStartTerminalLogin(provider, profile);
    onClose();
  };

  const dialogTitle = isOpencode
    ? t('usage.opencodeDialogTitle', {
        profile,
        defaultValue: `Configuração de Provedor · opencode (${profile})`,
      })
    : t('usage.loginDialogTitle', { provider: `${provider} (${profile})` });

  const dialogSubtitle = isOpencode
    ? t(
        'usage.opencodeDialogSubtitle',
        'O OpenCode funciona sem login para modelos locais ou permite conectar provedores externos (DeepSeek, Claude, OpenAI) e a nuvem OpenCode Zen.',
      )
    : t('usage.loginDialogSubtitle');

  return (
    <Dialog open={isOpen} onClose={onClose} title={dialogTitle}>
      <div className={styles.root}>
        <p className={styles.intro}>{dialogSubtitle}</p>

        <div className={styles.instructionBox}>
          {isOpencode ? (
            <div>
              <strong>{t('usage.opencodeModesTitle', 'Modos de Operação do OpenCode')}</strong>
              <div className={styles.opencodeModes}>
                <div className={styles.opencodeModeItem}>
                  <Cpu size={14} className={styles.instructionIcon} />
                  <div>
                    <strong>
                      {t('usage.opencodeModeLocalTitle', '1. Modo Local / Self-Hosted:')}
                    </strong>
                    <span>
                      {t(
                        'usage.opencodeModeLocalDesc',
                        'Execute com Ollama ou modelos locais sem necessidade de login ou chave de nuvem.',
                      )}
                    </span>
                  </div>
                </div>
                <div className={styles.opencodeModeItem}>
                  <Globe size={14} className={styles.instructionIcon} />
                  <div>
                    <strong>
                      {t('usage.opencodeModeProvidersTitle', '2. Provedores Externos:')}
                    </strong>
                    <span>
                      {t(
                        'usage.opencodeModeProvidersDesc',
                        'Conecte chaves de API (DeepSeek, Claude, OpenAI) usando o gerenciador de provedores.',
                      )}
                    </span>
                  </div>
                </div>
                <div className={styles.opencodeModeItem}>
                  <Sparkles size={14} className={styles.instructionIcon} />
                  <div>
                    <strong>{t('usage.opencodeModeZenTitle', '3. OpenCode Zen / Go:')}</strong>
                    <span>
                      {t(
                        'usage.opencodeModeZenDesc',
                        'Modelos oficiais em nuvem da OpenCode AI com assinatura ou autenticação de conta.',
                      )}
                    </span>
                  </div>
                </div>
              </div>
              {activeProvider && (
                <div className={styles.activeProviderBadge}>
                  <Check size={12} style={{ color: 'var(--nx-success)' }} />
                  <span>
                    {t('usage.opencodeActiveProvider', {
                      provider: activeProvider,
                      defaultValue: `Provedor ativo: ${activeProvider}`,
                    })}
                  </span>
                </div>
              )}
            </div>
          ) : (
            <>
              <KeyRound size={18} className={styles.instructionIcon} />
              <div>
                <strong>{t('usage.loginInstructionsTitle')}</strong>
                <span>{getInstructions()}</span>
              </div>
            </>
          )}
        </div>

        <div className={styles.actionsCard}>
          {onStartTerminalLogin && (
          <div className={styles.terminalOption}>
            <div className={styles.optionCopy}>
              <strong>
                {isOpencode
                  ? t('usage.opencodeTerminalAction', 'Gerenciar Provedores no Terminal')
                  : t('usage.loginTerminalAction')}
              </strong>
              <small>
                {isOpencode
                  ? t(
                      'usage.opencodeTerminalDesc',
                      'Abre o terminal interativo para configurar ou alternar provedores e chaves.',
                    )
                  : t('usage.loginTerminalDesc')}
              </small>
            </div>
            <Button tone="brand" size="sm" onClick={handleTerminalLogin}>
              <Terminal size={14} />
              {isOpencode
                ? t('usage.opencodeTerminalButton', 'Abrir Provedores')
                : t('usage.loginTerminalAction')}
            </Button>
          </div>
          )}

          <div className={styles.cliOption}>
            <span className={styles.cliLabel}>
              {isOpencode
                ? t('usage.opencodeCliAction', 'Comando para gerenciar provedores no seu terminal:')
                : t('usage.loginCliAction')}
            </span>
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
