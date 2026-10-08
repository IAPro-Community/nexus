import React, { useState } from 'react';
import {
  FolderGit2,
  Plus,
  Sparkles,
  Layers,
  Zap,
  Bot,
  ArrowRight,
  FolderOpen,
  RefreshCw,
} from 'lucide-react';
import { Button, Card, Input, IconButton } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Project } from '../../types';
import { useTranslation } from 'react-i18next';
import { DirectoryBrowserModal } from './DirectoryBrowserModal';
import { ProjectScanModal } from './ProjectScanModal';
import { selectProjectDirectory } from './projectDirectoryPicker';
import styles from './ProjectHub.module.scss';

export const ProjectHub: React.FC<{ onCreated: (project: Project) => void }> = ({ onCreated }) => {
  const { t } = useTranslation();
  const [path, setPath] = useState('');
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [dirPickerOpen, setDirPickerOpen] = useState(false);
  const [scanModalOpen, setScanModalOpen] = useState(false);

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

        {/* Primary Local Repo Input */}
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
                placeholder="/home/user/my-repository"
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

        {/* Starter Templates */}
        <div className={styles.templates}>
          <span className={styles.templatesHeading}>{t('projectManager.templatesTitle')}</span>
          <div className={styles.templatesGrid} data-testid="project-hub-templates">
            <div
              className={styles.templateCard}
              onClick={() => create('/projetos/saas-app', 'Fullstack SaaS App')}
            >
              <div className={styles.templateIcon}>
                <Layers size={16} />
              </div>
              <div className={styles.templateCopy}>
                <strong>{t('projectManager.templateFullstack')}</strong>
                <small>{t('projectManager.templateFullstackDescription')}</small>
              </div>
              <ArrowRight size={13} className={styles.templateArrow} />
            </div>

            <div
              className={styles.templateCard}
              onClick={() => create('/projetos/api-service', 'Core API Service')}
            >
              <div className={styles.templateIcon}>
                <Zap size={16} />
              </div>
              <div className={styles.templateCopy}>
                <strong>{t('projectManager.templateApi')}</strong>
                <small>{t('projectManager.templateApiDescription')}</small>
              </div>
              <ArrowRight size={13} className={styles.templateArrow} />
            </div>

            <div
              className={styles.templateCard}
              onClick={() => create('/projetos/cli-tool', 'Developer CLI Tool')}
            >
              <div className={styles.templateIcon}>
                <Bot size={16} />
              </div>
              <div className={styles.templateCopy}>
                <strong>{t('projectManager.templateCli')}</strong>
                <small>{t('projectManager.templateCliDescription')}</small>
              </div>
              <ArrowRight size={13} className={styles.templateArrow} />
            </div>
          </div>
        </div>

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
    </>
  );
};
