import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Folder,
  Home,
  Monitor,
  FileText,
  HardDrive,
  CornerLeftUp,
  Search,
  Check,
  Loader2,
  FolderPlus,
} from 'lucide-react';
import { Dialog, Button, Input, IconButton, Badge } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { FSBrowseResult, FSEntry } from '../../types';
import styles from './DirectoryBrowserModal.module.scss';

export const DirectoryBrowserModal: React.FC<{
  open: boolean;
  onClose: () => void;
  initialPath?: string;
  onSelectPath: (path: string, suggestedName?: string) => void;
}> = ({ open, onClose, initialPath, onSelectPath }) => {
  const { t } = useTranslation();
  const [currentPath, setCurrentPath] = useState(initialPath || '');
  const [pathInput, setPathInput] = useState(initialPath || '');
  const [data, setData] = useState<FSBrowseResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [query, setQuery] = useState('');
  const [newFolderOpen, setNewFolderOpen] = useState(false);
  const [newFolderName, setNewFolderName] = useState('');
  const [mkdirBusy, setMkdirBusy] = useState(false);
  const [mkdirError, setMkdirError] = useState('');
  const [loadError, setLoadError] = useState('');
  const cacheRef = useRef(new Map<string, FSBrowseResult>());
  const requestRef = useRef(0);

  const loadDirectory = useCallback(
    async (path?: string) => {
      const cacheKey = path?.trim() || '';
      const cached = cacheRef.current.get(cacheKey);
      if (cached) {
        setData(cached);
        setCurrentPath(cached.current_path);
        setQuery('');
        setLoadError('');
        setNewFolderOpen(false);
        return;
      }

      const requestId = ++requestRef.current;
      setLoading(true);
      setLoadError('');
      try {
        const res = await nexus.browseFS(path);
        if (requestId !== requestRef.current) return;
        cacheRef.current.set(res.current_path, res);
        setData(res);
        setCurrentPath(res.current_path);
        setPathInput(res.current_path);
        setQuery('');
        setNewFolderOpen(false);
      } catch (err) {
        if (requestId === requestRef.current) {
          setLoadError(err instanceof Error ? err.message : t('projectManager.browseError'));
        }
      } finally {
        if (requestId === requestRef.current) setLoading(false);
      }
    },
    [t],
  );

  useEffect(() => {
    if (open) {
      cacheRef.current.clear();
      setData(null);
      void loadDirectory(initialPath);
    }
  }, [open, initialPath, loadDirectory]);

  const handleSelectCurrent = () => {
    if (!data) return;
    const name = data.current_path.split(/[/\\]/).filter(Boolean).pop() || 'Project';
    onSelectPath(data.current_path, name);
    onClose();
  };

  const handleSelectEntry = (entry: FSEntry) => {
    if (entry.is_dir) {
      void loadDirectory(entry.path);
    }
  };

  const handleCreateFolder = async () => {
    if (!newFolderName.trim() || !data) return;
    setMkdirBusy(true);
    setMkdirError('');
    try {
      const target = `${data.current_path}/${newFolderName.trim()}`;
      await nexus.mkdirFS(target);
      cacheRef.current.delete(data.current_path);
      setNewFolderName('');
      setNewFolderOpen(false);
      void loadDirectory(target);
    } catch (err) {
      setMkdirError(err instanceof Error ? err.message : String(err));
    } finally {
      setMkdirBusy(false);
    }
  };

  const filteredEntries = (data?.entries || []).filter(
    (e) => e.is_dir && e.name.toLowerCase().includes(query.toLowerCase()),
  );

  const handlePathInputSubmit = () => {
    const trimmed = pathInput.trim();
    if (trimmed) {
      void loadDirectory(trimmed);
    }
  };

  const getBookmarkIcon = (icon: string) => {
    switch (icon) {
      case 'home':
        return <Home size={14} />;
      case 'desktop':
        return <Monitor size={14} />;
      case 'documents':
        return <FileText size={14} />;
      case 'root':
        return <HardDrive size={14} />;
      default:
        return <Folder size={14} />;
    }
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('projectManager.browseOS')}
      full
      className={styles.dialog}
    >
      <div className={styles.picker}>
        {/* Top OS Breadcrumbs bar */}
        <div className={styles.breadcrumbs}>
          {data?.parent_path && (
            <IconButton
              label={t('projectManager.upDirectory')}
              onClick={() => loadDirectory(data.parent_path)}
            >
              <CornerLeftUp size={14} />
            </IconButton>
          )}

          <input
            type="text"
            className={styles.pathInput}
            value={pathInput}
            onChange={(e) => setPathInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') handlePathInputSubmit();
            }}
            onBlur={handlePathInputSubmit}
            placeholder={t('projectManager.pathPlaceholder')}
            spellCheck={false}
          />

          <div className={styles.topActions}>
            <Button size="sm" tone="ghost" onClick={() => setNewFolderOpen((prev) => !prev)}>
              <FolderPlus size={13} />
              <span>{t('projectManager.newFolder')}</span>
            </Button>
          </div>
        </div>

        {/* New Folder Inline Form */}
        {newFolderOpen && (
          <div className={styles.mkdirBox}>
            <Input
              value={newFolderName}
              onChange={setNewFolderName}
              placeholder={t('projectManager.folderName')}
              onEnter={handleCreateFolder}
              autoFocus
            />
            {mkdirError && <span className="nx-error-copy">{mkdirError}</span>}
            <div className={styles.mkdirActions}>
              <Button size="sm" onClick={() => setNewFolderOpen(false)}>
                {t('common.closeDialog')}
              </Button>
              <Button
                size="sm"
                tone="brand"
                disabled={!newFolderName.trim() || mkdirBusy}
                onClick={handleCreateFolder}
              >
                {t('projectManager.createFolder')}
              </Button>
            </div>
          </div>
        )}

        <div className={styles.body}>
          {/* OS Quick Bookmarks Sidebar */}
          <div className={styles.sidebar}>
            <span className={styles.sidebarHeading}>{t('projectManager.bookmarks')}</span>
            <div className={styles.bookmarksList}>
              {data?.bookmarks.map((b) => (
                <button
                  type="button"
                  key={b.path}
                  className={`${styles.bookmarkButton} ${currentPath === b.path ? styles.bookmarkActive : ''}`}
                  onClick={() => loadDirectory(b.path)}
                >
                  {getBookmarkIcon(b.icon)}
                  <span>{b.label}</span>
                </button>
              ))}
            </div>
          </div>

          {/* Directory Explorer Pane */}
          <div className={styles.main}>
            <div className={styles.searchBar}>
              <Search size={15} className={styles.searchIcon} />
              <Input
                className={styles.searchInput}
                value={query}
                onChange={setQuery}
                placeholder={t('common.search')}
              />
            </div>

            {loading ? (
              <div className={styles.loading}>
                <Loader2 size={24} className="nx-spin" />
                <span>{t('common.loading')}</span>
              </div>
            ) : loadError ? (
              <div className={styles.empty} role="alert">
                <p>{loadError}</p>
                <Button size="sm" onClick={() => void loadDirectory(currentPath)}>
                  {t('projectManager.retry')}
                </Button>
              </div>
            ) : (
              <div className={styles.entriesGrid}>
                {filteredEntries.map((entry) => (
                  <button
                    type="button"
                    key={entry.path}
                    className={styles.entryCard}
                    aria-label={t('projectManager.openFolder', { name: entry.name })}
                    onClick={() => handleSelectEntry(entry)}
                  >
                    <div className={styles.entryIcon}>
                      <Folder size={22} />
                    </div>
                    <div className={styles.entryMeta}>
                      <strong>{entry.name}</strong>
                      <span className={styles.entryHint}>{t('projectManager.openFolderHint')}</span>
                    </div>
                  </button>
                ))}

                {filteredEntries.length === 0 && (
                  <div className={styles.empty}>
                    <p>{t('projectManager.noProjects')}</p>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        {/* Footer with Selected Folder info & confirmation button */}
        <div className={styles.footer}>
          <div className={styles.selectedInfo}>
            <span className={styles.selectedLabel}>{t('projectManager.currentFolder')}</span>
            <code>{currentPath}</code>
            {data?.is_git && (
              <Badge tone="brand">
                {t('projectManager.gitDetected', { branch: data.git_branch || 'main' })}
              </Badge>
            )}
            {data?.tech && data.tech.length > 0 && (
              <Badge tone="default">{data.tech.join(', ')}</Badge>
            )}
          </div>

          <div className={styles.footerActions}>
            <Button onClick={onClose}>{t('common.closeDialog')}</Button>
            <Button tone="brand" onClick={handleSelectCurrent}>
              <Check size={14} />
              <span>{t('projectManager.selectFolder')}</span>
            </Button>
          </div>
        </div>
      </div>
    </Dialog>
  );
};
