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
      <div className="nx-project-hub">
        <div className="nx-project-hub__brand">
          <span className="nx-brand-mark nx-brand-mark--hero">
            <img src="./nexus-icon.png" alt="Nexus" className="nx-brand-mark__img" />
          </span>
          <span className="nx-eyebrow">{t('projects.brandEyebrow')}</span>
          <h1>{t('projects.headline')}</h1>
          <p>{t('projects.intro')}</p>
        </div>

        {/* Primary Local Repo Input */}
        <Card className="nx-project-hub__create">
          <div className="nx-work-card-icon">
            <FolderGit2 size={20} />
          </div>
          <div className="nx-hub-create-header">
            <div>
              <strong>{t('projects.addRepo')}</strong>
              <small>{t('projects.addDescription')}</small>
            </div>
            <div className="nx-hub-os-actions">
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

          <div className="nx-hub-inputs">
            <div className="nx-input-with-action">
              <Input
                value={path}
                onChange={setPath}
                onEnter={() => create()}
                placeholder="/home/user/my-repository"
                mono
                autoFocus
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
          {error && <p className="nx-error-copy">{error}</p>}
          <Button tone="brand" disabled={!path.trim() || busy} onClick={() => create()}>
            <Plus size={14} />
            {busy ? t('projects.adding') : t('projects.add')}
          </Button>
        </Card>

        {/* Starter Templates */}
        <div className="nx-hub-templates">
          <span className="nx-hub-templates-heading">{t('projectManager.templatesTitle')}</span>
          <div className="nx-hub-templates-grid">
            <div
              className="nx-hub-template-card"
              onClick={() => create('/projetos/saas-app', 'Fullstack SaaS App')}
            >
              <div className="nx-hub-template-icon">
                <Layers size={16} />
              </div>
              <div>
                <strong>{t('projectManager.templateFullstack')}</strong>
                <small>React + Go/Node + Maestro</small>
              </div>
              <ArrowRight size={13} className="nx-hub-template-arrow" />
            </div>

            <div
              className="nx-hub-template-card"
              onClick={() => create('/projetos/api-service', 'Core API Service')}
            >
              <div className="nx-hub-template-icon">
                <Zap size={16} />
              </div>
              <div>
                <strong>{t('projectManager.templateApi')}</strong>
                <small>REST / gRPC with TDD engine</small>
              </div>
              <ArrowRight size={13} className="nx-hub-template-arrow" />
            </div>

            <div
              className="nx-hub-template-card"
              onClick={() => create('/projetos/cli-tool', 'Developer CLI Tool')}
            >
              <div className="nx-hub-template-icon">
                <Bot size={16} />
              </div>
              <div>
                <strong>{t('projectManager.templateCli')}</strong>
                <small>CLI Tool with tests</small>
              </div>
              <ArrowRight size={13} className="nx-hub-template-arrow" />
            </div>
          </div>
        </div>

        <div className="nx-project-hub__tip">
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
