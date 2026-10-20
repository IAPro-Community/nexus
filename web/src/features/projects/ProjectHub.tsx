import React, { useCallback, useEffect, useState } from 'react';
import {
  FolderGit2,
  Plus,
  Sparkles,
  FolderOpen,
  RefreshCw,
  CheckCircle2,
  AlertCircle,
} from 'lucide-react';
import { Button, Card, Input, IconButton } from '../../design-system';
import { nexus, type SystemDoctorReport } from '../../nexus/api';
import type { Project, ProviderAccount, ProviderInfo } from '../../types';
import { useTranslation } from 'react-i18next';
import { DirectoryBrowserModal } from './DirectoryBrowserModal';
import { ProjectScanModal } from './ProjectScanModal';
import { selectProjectDirectory } from './projectDirectoryPicker';
import { getRepositoryPathPlaceholderKey } from './repositoryPathPlaceholder';
import styles from './ProjectHub.module.scss';
import { RegisterProviderModal } from '../usage/RegisterProviderModal';
import { AccountLoginModal } from '../usage/AccountLoginModal';
import { deriveFirstRunReadiness, OFFICIAL_PROVIDER_INSTALL_GUIDES } from './firstRunReadiness';

export const ProjectHub: React.FC<{ onCreated: (project: Project) => void }> = ({ onCreated }) => {
  const { t } = useTranslation();
  const pathPlaceholder = t(
    getRepositoryPathPlaceholderKey(typeof navigator === 'undefined' ? '' : navigator.userAgent),
  );
  const [path, setPath] = useState('');
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [dirPickerOpen, setDirPickerOpen] = useState(false);
  const [scanModalOpen, setScanModalOpen] = useState(false);
  const [accountModalOpen, setAccountModalOpen] = useState(false);
  const [loginAccount, setLoginAccount] = useState<ProviderAccount | null>(null);
  const [loginBusy, setLoginBusy] = useState(false);
  const [setupError, setSetupError] = useState('');
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [doctor, setDoctor] = useState<SystemDoctorReport | null>(null);
  const [detectionState, setDetectionState] = useState<'checking' | 'loaded' | 'unavailable'>(
    'checking',
  );

  // Read-only detection; never create profiles, install third-party tools,
  // inspect transcript contents or modify a user's existing authentication.
  const detectEnvironment = useCallback(async () => {
    setDetectionState('checking');
    const [providersResult, doctorResult] = await Promise.allSettled([
      nexus.listProviderInstallations(),
      nexus.getSystemDoctor(),
    ]);
    if (providersResult.status === 'fulfilled') {
      setProviders(providersResult.value);
      setDetectionState('loaded');
    } else {
      setDetectionState('unavailable');
    }
    setDoctor(doctorResult.status === 'fulfilled' ? doctorResult.value : null);
  }, []);

  useEffect(() => {
    void detectEnvironment();
  }, [detectEnvironment]);

  // Do not manufacture account identity from a discovered CLI. Read the backend ledger.
  const openExistingLogin = async (provider?: string, profile?: string) => {
    setSetupError('');
    setLoginBusy(true);
    try {
      const data = await nexus.listResources();
      const matches = (data.accounts || []).filter(
        (account) => !provider || account.provider === provider,
      );
      const account = profile
        ? matches.find((item) => item.profile === profile)
        : matches.find((item) => !item.authenticated) || matches[0];
      if (account) {
        setLoginAccount(account);
      } else {
        setSetupError(
          'Ainda não foi possível localizar esse perfil em Uso/Contas. ' +
            'Verifique novamente ou use nexus profiles no terminal para identificar o perfil.',
        );
      }
    } catch (err) {
      setSetupError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoginBusy(false);
    }
  };

  const readiness = deriveFirstRunReadiness(providers, detectionState);
  const desktopPrerequisite = doctor?.checks.find(
    (check) =>
      (check.id === 'platform.webview2' || check.id === 'platform.webkitgtk') &&
      (check.status === 'WARN' || check.status === 'FAIL'),
  );

  const create = async (customPath?: string, customName?: string) => {
    const targetPath = (customPath || path).trim();
    const targetName = (customName || name).trim();
    if (!targetPath) return;
    setBusy(true);
    setError('');
    try {
      const proj = await nexus.createProject(targetPath, targetName || undefined);
      onCreated(proj);
      setPath('');
      setName('');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const handleDirectorySelected = (selectedPath: string, suggestedName?: string) => {
    setPath(selectedPath);
    if (suggestedName && !name) {
      setName(suggestedName);
    }
  };

  const handleBrowseDirectory = async () => {
    const selection = await selectProjectDirectory(t('projectManager.browseOS'));
    if (!selection.supported) {
      setDirPickerOpen(true);
      return;
    }
    if (selection.path) {
      handleDirectorySelected(
        selection.path,
        selection.path.split(/[/\\]/).filter(Boolean).pop() || 'Project',
      );
    }
  };

  return (
    <>
      <div className={styles.root} data-testid="project-hub-root">
        <div className={styles.brand}>
          <span className="nx-brand-mark nx-brand-mark--hero">
            <img src="./nexus-icon.png" alt="Nexus" className="nx-brand-mark__img" />
          </span>
          <span className="nx-eyebrow">{t('projects.brandEyebrow')}</span>
          <h1>{t('projects.headline')}</h1>
          <p>{t('projects.intro')}</p>
        </div>

        {/* A project is chosen from a real local directory; never invent a path. */}
        <Card className={styles.create} data-testid="project-hub-create">
          <div className="nx-work-card-icon">
            <FolderGit2 size={20} />
          </div>
          <div className={styles.createHeader}>
            <div>
              <strong>{t('projects.addRepo')}</strong>
              <small>{t('projects.addDescription')}</small>
            </div>
            <div className={styles.osActions}>
              <Button size="sm" tone="ghost" onClick={() => void handleBrowseDirectory()}>
                <FolderOpen size={13} />
                <span>{t('projectManager.browseOS')}</span>
              </Button>
              <Button size="sm" tone="ghost" onClick={() => setScanModalOpen(true)}>
                <RefreshCw size={13} />
                <span>{t('projectManager.scanOS')}</span>
              </Button>
            </div>
          </div>

          <div className={styles.inputs}>
            <div className="nx-input-with-action">
              <Input
                value={path}
                onChange={setPath}
                onEnter={() => create()}
                placeholder={pathPlaceholder}
                mono
              />
              <IconButton
                label={t('projectManager.browseOS')}
                onClick={() => void handleBrowseDirectory()}
              >
                <FolderOpen size={14} />
              </IconButton>
            </div>
            <Input
              value={name}
              onChange={setName}
              onEnter={() => create()}
              placeholder={t('rail.namePlaceholder')}
            />
          </div>
          {error && (
            <p className={`nx-error-copy ${styles.error}`} role="alert">
              {error}
            </p>
          )}
          <Button
            className={styles.submit}
            tone="brand"
            disabled={!path.trim() || busy}
            onClick={() => create()}
          >
            <Plus size={14} />
            {busy ? t('projects.adding') : t('projects.add')}
          </Button>
        </Card>

        {/* Show a truthful, unobtrusive setup assessment before the first project. */}
        <Card className={styles.readiness} data-testid="project-hub-readiness">
          <div className={styles.readinessHeader}>
            <span className={styles.readinessTitle}>
              {readiness.status === 'ready' ? (
                <CheckCircle2 size={17} aria-hidden="true" />
              ) : (
                <AlertCircle size={17} aria-hidden="true" />
              )}
              Ambiente de execução
            </span>
            <Button size="sm" tone="ghost" onClick={() => void detectEnvironment()}>
              <RefreshCw size={13} aria-hidden="true" />
              Verificar novamente
            </Button>
          </div>
          <div role="status" aria-live="polite" className={styles.readinessMessage}>
            <strong>{readiness.title}</strong>
            <p>{readiness.description}</p>
          </div>
          {readiness.status === 'needs-account' && readiness.suggestedProvider && (
            <Button size="sm" tone="brand" onClick={() => setAccountModalOpen(true)}>
              <Plus size={14} aria-hidden="true" />
              Adicionar conta {readiness.suggestedProvider}
            </Button>
          )}
          {readiness.status === 'needs-login' && (
            <Button
              size="sm"
              tone="brand"
              disabled={loginBusy}
              onClick={() => void openExistingLogin(readiness.suggestedProvider)}
            >
              Concluir login da conta existente
            </Button>
          )}
          {readiness.status === 'needs-provider' && (
            <div className={styles.installGuides}>
              <span>Instalação oficial:</span>
              {OFFICIAL_PROVIDER_INSTALL_GUIDES.map((guide) => (
                <a key={guide.label} href={guide.href} target="_blank" rel="noopener noreferrer">
                  {guide.label}
                </a>
              ))}
            </div>
          )}
          {setupError && (
            <p role="alert" className={styles.readinessWarning}>
              {setupError}
            </p>
          )}
          {desktopPrerequisite && (
            <p className={styles.readinessWarning}>
              Desktop: dependência nativa ainda não confirmada. O terminal e o navegador podem
              continuar disponíveis; consulte o diagnóstico do sistema.
            </p>
          )}
        </Card>

        <div className={styles.tip}>
          <Sparkles size={15} />
          <span>{t('projects.tip')}</span>
        </div>
      </div>

      <DirectoryBrowserModal
        open={dirPickerOpen}
        onClose={() => setDirPickerOpen(false)}
        initialPath={path || undefined}
        onSelectPath={handleDirectorySelected}
      />

      <ProjectScanModal
        open={scanModalOpen}
        onClose={() => setScanModalOpen(false)}
        onProjectImported={(proj) => onCreated(proj)}
      />

      <RegisterProviderModal
        isOpen={accountModalOpen}
        initialProvider={readiness.suggestedProvider}
        onClose={() => setAccountModalOpen(false)}
        onRegistered={async (provider, profile) => {
          await detectEnvironment();
          await openExistingLogin(provider, profile);
        }}
      />
      <AccountLoginModal
        account={loginAccount}
        isOpen={Boolean(loginAccount)}
        onClose={() => setLoginAccount(null)}
        onLoginSuccess={detectEnvironment}
      />
    </>
  );
};
