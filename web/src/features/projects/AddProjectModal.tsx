import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { FolderOpen, RefreshCw, Check, Plus } from 'lucide-react';
import { Button, Dialog, IconButton, Input, Badge } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { FSInspectResult, Project } from '../../types';
import { DirectoryBrowserModal } from './DirectoryBrowserModal';
import { ProjectScanModal } from './ProjectScanModal';
import { selectProjectDirectory } from './projectDirectoryPicker';

export const AddProjectModal: React.FC<{
  open: boolean;
  onClose: () => void;
  onCreated: (project: Project) => void;
}> = ({ open, onClose, onCreated }) => {
  const { t } = useTranslation();
  const [path, setPath] = useState('');
  const [name, setName] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [inspectInfo, setInspectInfo] = useState<FSInspectResult | null>(null);
  const [dirPickerOpen, setDirPickerOpen] = useState(false);
  const [scanModalOpen, setScanModalOpen] = useState(false);

  // Reset form when modal opens
  useEffect(() => {
    if (open) {
      setError('');
      setInspectInfo(null);
    }
  }, [open]);

  // Live inspect when path changes
  useEffect(() => {
    if (!path.trim()) {
      setInspectInfo(null);
      return;
    }
    const timer = setTimeout(() => {
      nexus
        .inspectFS(path.trim())
        .then((res) => {
          setInspectInfo(res);
          if (res.suggested_name && !name) {
            setName(res.suggested_name);
          }
        })
        .catch(() => setInspectInfo(null));
    }, 300);
    return () => clearTimeout(timer);
  }, [path, name]);

  const handleCreate = async () => {
    if (!path.trim() || busy) return;
    setBusy(true);
    setError('');
    try {
      const created = await nexus.createProject(path.trim(), name.trim() || undefined);
      onCreated(created);
      setPath('');
      setName('');
      setInspectInfo(null);
      onClose();
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
      <Dialog open={open} onClose={onClose} title={t('projectManager.addNew')}>
        <div className="nx-form-stack">
          {/* OS Integration Helper Buttons */}
          <div className="nx-pm-add-helpers">
            <Button size="sm" tone="ghost" onClick={() => void handleBrowseDirectory()}>
              <FolderOpen size={13} />
              <span>{t('projectManager.browseOS')}</span>
            </Button>
            <Button size="sm" tone="ghost" onClick={() => setScanModalOpen(true)}>
              <RefreshCw size={13} />
              <span>{t('projectManager.scanOS')}</span>
            </Button>
          </div>

          {/* Repository Path Input */}
          <label>
            {t('projectManager.addPath')}
            <div className="nx-input-with-action">
              <Input
                value={path}
                onChange={setPath}
                onEnter={handleCreate}
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
          </label>

          {/* Live Git & Tech Stack Inspection Badge */}
          {inspectInfo && inspectInfo.exists && (
            <div className="nx-inspect-badge-row">
              {inspectInfo.is_git ? (
                <Badge tone="success">
                  <Check size={11} />{' '}
                  {t('projectManager.gitDetected', { branch: inspectInfo.git_branch || 'main' })}
                </Badge>
              ) : (
                <Badge tone="default">{t('projectManager.localNoGit')}</Badge>
              )}
              {(inspectInfo.tech ?? []).map((tech) => (
                <span key={tech} className="nx-tech-tag">
                  {tech}
                </span>
              ))}
            </div>
          )}

          {/* Display Name Input */}
          <label>
            {t('projectManager.addName')}
            <Input
              value={name}
              onChange={setName}
              onEnter={handleCreate}
              placeholder="Payment Backend Service"
            />
          </label>

          {error && <p className="nx-error-copy">{error}</p>}

          <div className="nx-dialog-actions">
            <Button onClick={onClose}>{t('rail.cancel')}</Button>
            <Button tone="brand" disabled={!path.trim() || busy} onClick={handleCreate}>
              <Plus size={13} />
              <span>{busy ? t('rail.adding') : t('projectManager.createButton')}</span>
            </Button>
          </div>
        </div>
      </Dialog>

      {/* Directory Browser Modal */}
      <DirectoryBrowserModal
        open={dirPickerOpen}
        onClose={() => setDirPickerOpen(false)}
        onSelectPath={handleDirectorySelected}
      />

      {/* Project Auto-Scan Modal */}
      <ProjectScanModal
        open={scanModalOpen}
        onClose={() => setScanModalOpen(false)}
        onProjectImported={(imported: Project) => {
          onCreated(imported);
          onClose();
        }}
      />
    </>
  );
};
