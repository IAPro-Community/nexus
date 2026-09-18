import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { FolderGit2, GitBranch, Search, Check, Plus, Loader2, RefreshCw } from 'lucide-react';
import { Dialog, Button, Input, Badge, Card } from '../../design-system';
import { toast } from 'sonner';
import { nexus } from '../../nexus/api';
import type { FSScanResult, Project } from '../../types';

export const ProjectScanModal: React.FC<{
  open: boolean;
  onClose: () => void;
  onProjectImported: (project: Project) => void;
}> = ({ open, onClose, onProjectImported }) => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [results, setResults] = useState<FSScanResult[]>([]);
  const [query, setQuery] = useState('');
  const [importingPath, setImportingPath] = useState<string | null>(null);

  const startScan = async () => {
    setLoading(true);
    try {
      const items = await nexus.scanFS();
      setResults(items);
    } catch (err) {
      console.error('Scan failed', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    if (open) {
      void startScan();
    }
  }, [open]);

  const handleImport = async (item: FSScanResult) => {
    setImportingPath(item.path);
    try {
      const proj = await nexus.createProject(item.path, item.name);
      onProjectImported(proj);
      setResults((current) =>
        current.map((r) => (r.path === item.path ? { ...r, is_imported: true } : r)),
      );
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setImportingPath(null);
    }
  };

  const filtered = results.filter(
    (r) =>
      r.name.toLowerCase().includes(query.toLowerCase()) ||
      r.path.toLowerCase().includes(query.toLowerCase()) ||
      r.branch.toLowerCase().includes(query.toLowerCase()),
  );

  return (
    <Dialog open={open} onClose={onClose} title={t('projectManager.scanOS')} wide>
      <div className="nx-scan-modal">
        <div className="nx-scan-toolbar">
          <div className="nx-scan-search">
            <Search size={14} className="nx-scan-search-icon" />
            <Input value={query} onChange={setQuery} placeholder={t('common.search')} autoFocus />
          </div>
          <Button size="sm" onClick={startScan} disabled={loading}>
            <RefreshCw size={13} className={loading ? 'nx-spin' : ''} />
            <span>{loading ? t('projectManager.scanning') : t('projectManager.rescan')}</span>
          </Button>
        </div>

        {loading ? (
          <div className="nx-scan-loading">
            <Loader2 size={32} className="nx-spin" />
            <p>{t('projectManager.scanning')}</p>
          </div>
        ) : (
          <div className="nx-scan-results-list">
            <span className="nx-scan-count">
              {t('projectManager.scanFound', { count: filtered.length })}
            </span>

            {filtered.map((item) => (
              <Card key={item.path} className="nx-scan-item-card">
                <div className="nx-scan-item-main">
                  <div className="nx-scan-avatar">
                    {(item.name || 'PR').slice(0, 2).toUpperCase()}
                  </div>
                  <div className="nx-scan-info">
                    <div className="nx-scan-title-row">
                      <strong>{item.name}</strong>
                      <span className="nx-desktop-branch">
                        <GitBranch size={11} /> {item.branch || 'main'}
                      </span>
                    </div>
                    <code className="nx-scan-path">{item.path}</code>
                    <div className="nx-scan-tags">
                      {(item.tech ?? []).map((tech) => (
                        <span key={tech} className="nx-tech-tag">
                          {tech}
                        </span>
                      ))}
                    </div>
                  </div>
                </div>

                <div className="nx-scan-actions">
                  {item.is_imported ? (
                    <Badge tone="success">
                      <Check size={11} /> {t('projectManager.alreadyImported')}
                    </Badge>
                  ) : (
                    <Button
                      size="sm"
                      tone="brand"
                      disabled={importingPath === item.path}
                      onClick={() => handleImport(item)}
                    >
                      <Plus size={13} />
                      <span>
                        {importingPath === item.path
                          ? t('projectManager.adding')
                          : t('projectManager.importProject')}
                      </span>
                    </Button>
                  )}
                </div>
              </Card>
            ))}

            {filtered.length === 0 && (
              <div className="nx-scan-empty">
                <FolderGit2 size={36} style={{ opacity: 0.3 }} />
                <p>{t('projectManager.scanEmpty')}</p>
              </div>
            )}
          </div>
        )}

        <div className="nx-dialog-actions">
          <Button onClick={onClose}>{t('common.closeDialog')}</Button>
        </div>
      </div>
    </Dialog>
  );
};
