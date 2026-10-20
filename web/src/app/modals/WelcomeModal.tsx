import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Compass,
  Sparkles,
  Bot,
  FolderGit2,
  TerminalSquare,
  Command,
  CheckCircle2,
  AlertCircle,
  ArrowRight,
  ShieldCheck,
} from 'lucide-react';
import { Dialog, Button, Badge } from '../../design-system';
import { nexus } from '../../nexus/api';
import {
  deriveFirstRunReadiness,
  OFFICIAL_PROVIDER_INSTALL_GUIDES,
} from '../../features/projects/firstRunReadiness';
import type { ProviderInfo } from '../../types';
import styles from './WelcomeModal.module.scss';

export const WelcomeModal: React.FC<{
  open: boolean;
  onClose: () => void;
  onStartTour: () => void;
  onOpenAccounts: () => void;
}> = ({ open, onClose, onStartTour, onOpenAccounts }) => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<'overview' | 'quickstart' | 'shortcuts' | 'system'>(
    'quickstart',
  );
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [detectionState, setDetectionState] = useState<'checking' | 'loaded' | 'unavailable'>(
    'checking',
  );
  const [sysInfo, setSysInfo] = useState<{
    nexus_version: string;
    maestro_version: string;
    maestro_available: boolean;
    update_available: boolean;
  } | null>(null);

  useEffect(() => {
    if (!open) return;
    let mounted = true;
    setDetectionState('checking');
    void Promise.allSettled([nexus.getSystemUpdates(), nexus.listProviderInstallations()]).then(
      ([updates, detected]) => {
        if (!mounted) return;
        if (updates.status === 'fulfilled') setSysInfo(updates.value);
        if (detected.status === 'fulfilled') {
          setProviders(detected.value);
          setDetectionState('loaded');
        } else {
          setDetectionState('unavailable');
        }
      },
    );
    return () => {
      mounted = false;
    };
  }, [open]);

  const readiness = deriveFirstRunReadiness(providers, detectionState);

  return (
    <Dialog open={open} onClose={onClose} title={t('welcome.title')}>
      <div className="nx-welcome-modal">
        <p className="nx-welcome-subtitle">{t('welcome.subtitle')}</p>

        {/* Tab Navigation */}
        <div className="nx-welcome-tabs" role="tablist">
          <button
            type="button"
            className={`nx-welcome-tab ${activeTab === 'overview' ? 'nx-welcome-tab--active' : ''}`}
            onClick={() => setActiveTab('overview')}
          >
            <Sparkles size={14} />
            <span>{t('welcome.tabOverview')}</span>
          </button>
          <button
            type="button"
            className={`nx-welcome-tab ${activeTab === 'quickstart' ? 'nx-welcome-tab--active' : ''}`}
            onClick={() => setActiveTab('quickstart')}
          >
            <Compass size={14} />
            <span>{t('welcome.tabQuickstart')}</span>
          </button>
          <button
            type="button"
            className={`nx-welcome-tab ${activeTab === 'shortcuts' ? 'nx-welcome-tab--active' : ''}`}
            onClick={() => setActiveTab('shortcuts')}
          >
            <Command size={14} />
            <span>{t('welcome.tabShortcuts')}</span>
          </button>
          <button
            type="button"
            className={`nx-welcome-tab ${activeTab === 'system' ? 'nx-welcome-tab--active' : ''}`}
            onClick={() => setActiveTab('system')}
          >
            <ShieldCheck size={14} />
            <span>{t('welcome.tabSystem')}</span>
          </button>
        </div>

        {/* Tab Content */}
        <div className="nx-welcome-content">
          {activeTab === 'overview' && (
            <div className="nx-welcome-panel">
              <h4>{t('welcome.overviewTitle')}</h4>
              <p>{t('welcome.overviewP1')}</p>
              <p>{t('welcome.overviewP2')}</p>
              <div className="nx-welcome-alert">
                <ShieldCheck size={16} />
                <span>{t('welcome.honestyModel')}</span>
              </div>
            </div>
          )}

          {activeTab === 'quickstart' && (
            <div className="nx-welcome-panel">
              <h4>{t('welcome.quickstartTitle')}</h4>
              <div className="nx-system-versions-card" role="status" aria-live="polite">
                <strong>{readiness.title}</strong>
                <p>{readiness.description}</p>
                {readiness.status === 'needs-provider' && (
                  <p>
                    Instalação oficial:{' '}
                    {OFFICIAL_PROVIDER_INSTALL_GUIDES.map((guide) => (
                      <a
                        key={guide.label}
                        href={guide.href}
                        target="_blank"
                        rel="noopener noreferrer"
                        style={{ marginRight: 12 }}
                      >
                        {guide.label}
                      </a>
                    ))}
                  </p>
                )}
                <Button
                  size="sm"
                  tone="brand"
                  onClick={() => {
                    onClose();
                    onOpenAccounts();
                  }}
                >
                  {readiness.status === 'ready' ? 'Gerenciar contas de IA' : 'Configurar contas de IA'}
                  <ArrowRight size={13} />
                </Button>
              </div>
              <div className="nx-quickstart-grid">
                <div className="nx-quickstart-step">
                  <div className="nx-step-badge">
                    <FolderGit2 size={16} />
                  </div>
                  <div className="nx-step-body">
                    <strong>{t('welcome.step1Title')}</strong>
                    <p>{t('welcome.step1Desc')}</p>
                  </div>
                </div>
                <div className="nx-quickstart-step">
                  <div className="nx-step-badge">
                    <Bot size={16} />
                  </div>
                  <div className="nx-step-body">
                    <strong>{t('welcome.step2Title')}</strong>
                    <p>{t('welcome.step2Desc')}</p>
                  </div>
                </div>
                <div className="nx-quickstart-step">
                  <div className="nx-step-badge">
                    <TerminalSquare size={16} />
                  </div>
                  <div className="nx-step-body">
                    <strong>{t('welcome.step3Title')}</strong>
                    <p>{t('welcome.step3Desc')}</p>
                  </div>
                </div>
              </div>
            </div>
          )}

          {activeTab === 'shortcuts' && (
            <div className="nx-welcome-panel">
              <h4>{t('welcome.shortcutsTitle')}</h4>
              <table className="nx-shortcuts-table">
                <tbody>
                  <tr>
                    <td>
                      <kbd>Ctrl</kbd> + <kbd>K</kbd> / <kbd>⌘</kbd> + <kbd>K</kbd>
                    </td>
                    <td>{t('welcome.cmdPalette')}</td>
                  </tr>
                  <tr>
                    <td>
                      <kbd>Ctrl</kbd> + <kbd>P</kbd> / <kbd>⌘</kbd> + <kbd>P</kbd>
                    </td>
                    <td>{t('welcome.projManager')}</td>
                  </tr>
                  <tr>
                    <td>
                      <code>/nexus help</code>
                    </td>
                    <td>{t('welcome.slashHelp')}</td>
                  </tr>
                  <tr>
                    <td>
                      <code>/nexus handoff &lt;account&gt;</code>
                    </td>
                    <td>{t('welcome.slashHandoff')}</td>
                  </tr>
                  <tr>
                    <td>
                      <code>/nexus continue --with &lt;prov&gt;</code>
                    </td>
                    <td>{t('welcome.slashContinue')}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          )}

          {activeTab === 'system' && (
            <div className="nx-welcome-panel">
              <h4>{t('welcome.systemTitle')}</h4>
              <div className="nx-system-versions-card" role="status" aria-live="polite">
                <strong>{readiness.title}</strong>
                <p>{readiness.description}</p>
                {readiness.status === 'needs-login' && (
                  <Button
                    size="sm"
                    tone="brand"
                    onClick={() => {
                      onClose();
                      onOpenAccounts();
                    }}
                  >
                    Concluir login em Uso/Contas
                  </Button>
                )}
              </div>
              <div className="nx-system-versions-card">
                <div className="nx-version-row">
                  <span>{t('welcome.nexusVer')}</span>
                  <Badge tone="brand">v{sysInfo?.nexus_version || 'unknown'}</Badge>
                </div>
                <div className="nx-version-row">
                  <span>{t('welcome.maestroVer')}</span>
                  <Badge tone={sysInfo?.maestro_available ? 'success' : 'warning'}>
                    v{sysInfo?.maestro_version || 'unknown'}
                  </Badge>
                </div>
                <div className="nx-version-row">
                  <span>{t('welcome.maestroStatus')}</span>
                  <Badge tone={sysInfo?.maestro_available ? 'success' : 'warning'}>
                    {sysInfo?.maestro_available ? (
                      <CheckCircle2 size={12} />
                    ) : (
                      <AlertCircle size={12} />
                    )}
                    {sysInfo?.maestro_available ? 'AVAILABLE' : 'DEGRADED'}
                  </Badge>
                </div>
                <div className="nx-version-row">
                  <span>Updates</span>
                  <span className={styles.updateStatus}>
                    {sysInfo?.update_available ? (
                      <strong className={styles.updateWarning}>
                        {t('welcome.updateAvailable')}
                      </strong>
                    ) : (
                      <span className={styles.updateSuccess}>✓ {t('welcome.upToDate')}</span>
                    )}
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>

        {/* Footer Actions */}
        <div className="nx-welcome-footer">
          <Button
            tone="brand"
            onClick={() => {
              onClose();
              onStartTour();
            }}
          >
            <Compass size={14} />
            <span>{t('welcome.startTour')}</span>
            <ArrowRight size={13} />
          </Button>
          <Button onClick={onClose}>{t('welcome.close')}</Button>
        </div>
      </div>
    </Dialog>
  );
};
