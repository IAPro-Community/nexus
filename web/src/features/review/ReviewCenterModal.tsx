import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { FileDiff, RefreshCw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Dialog, InlineAlert, Spinner } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { GitChange, GitChangesResult, GitDiffResult, Project } from '../../types';
import styles from './ReviewCenterModal.module.scss';

interface ReviewCenterModalProps {
  open: boolean;
  onClose: () => void;
  project: Project;
}

function changeCode(change: GitChange): string {
  const code = `${change.index_status}${change.worktree_status}`.trim();
  return code || 'M';
}

export const ReviewCenterModal: React.FC<ReviewCenterModalProps> = ({
  open,
  onClose,
  project,
}) => {
  const { t } = useTranslation();
  const [changes, setChanges] = useState<GitChangesResult | null>(null);
  const [selectedPath, setSelectedPath] = useState('');
  const [staged, setStaged] = useState(false);
  const [diff, setDiff] = useState<GitDiffResult | null>(null);
  const [loadingChanges, setLoadingChanges] = useState(false);
  const [loadingDiff, setLoadingDiff] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const selectedChange = useMemo(
    () => changes?.changes.find((change) => change.path === selectedPath),
    [changes, selectedPath],
  );

  const loadChanges = useCallback(async () => {
    if (!project.id) return;
    setLoadingChanges(true);
    setError(null);
    try {
      const next = await nexus.getProjectChanges(project.id);
      setChanges(next);
      setSelectedPath((current) => {
        if (current && next.changes.some((change) => change.path === current)) return current;
        return next.changes[0]?.path || '';
      });
    } catch (err) {
      setError(err instanceof Error ? err.message : t('reviewCenter.loadFailed'));
    } finally {
      setLoadingChanges(false);
    }
  }, [project.id, t]);

  const loadDiff = useCallback(async () => {
    if (!project.id || !selectedPath) {
      setDiff(null);
      return;
    }
    setLoadingDiff(true);
    setError(null);
    try {
      setDiff(await nexus.getProjectDiff(project.id, selectedPath, staged));
    } catch (err) {
      setDiff(null);
      setError(err instanceof Error ? err.message : t('reviewCenter.diffFailed'));
    } finally {
      setLoadingDiff(false);
    }
  }, [project.id, selectedPath, staged, t]);

  useEffect(() => {
    if (!open) return;
    setDiff(null);
    setSelectedPath('');
    setStaged(false);
    void loadChanges();
  }, [open, loadChanges]);

  useEffect(() => {
    if (open && selectedPath) void loadDiff();
  }, [open, selectedPath, staged, loadDiff]);

  return (
    <Dialog open={open} onClose={onClose} title={t('reviewCenter.title')}>
      <div className={styles.layout}>
        <header className={styles.header}>
          <div>
            <strong>{project.name}</strong>
            <div className={styles.meta}>
              {changes?.current_branch || project.default_branch} · {changes?.head?.slice(0, 8) || '—'}
            </div>
          </div>
          <Button size="sm" onClick={loadChanges} disabled={loadingChanges}>
            <RefreshCw size={13} />
            {t('reviewCenter.refresh')}
          </Button>
        </header>

        {error && (
          <InlineAlert tone="danger" title={t('reviewCenter.error')}>
            {error}
          </InlineAlert>
        )}

        {changes?.truncated && (
          <InlineAlert tone="warning">{t('reviewCenter.changesTruncated')}</InlineAlert>
        )}

        <div className={styles.body}>
          <section className={styles.files} aria-label={t('reviewCenter.changedFiles')}>
            <div className={styles.sectionTitle}>
              <FileDiff size={14} />
              <span>{t('reviewCenter.changedFiles')}</span>
              <Badge>{changes?.changes.length || 0}</Badge>
            </div>

            {loadingChanges && !changes ? (
              <div className={styles.empty}>
                <Spinner label={t('reviewCenter.loading')} />
              </div>
            ) : (changes?.changes.length || 0) === 0 ? (
              <div className={styles.empty}>{t('reviewCenter.clean')}</div>
            ) : (
              <ul className={styles.fileList}>
                {changes?.changes.map((change) => (
                  <li key={`${change.old_path || ''}->${change.path}`}>
                    <button
                      type="button"
                      className={selectedPath === change.path ? styles.fileActive : styles.file}
                      onClick={() => {
                        setSelectedPath(change.path);
                        setStaged(false);
                      }}
                    >
                      <Badge>{changeCode(change)}</Badge>
                      <span className={styles.filePath}>
                        {change.old_path ? `${change.old_path} → ${change.path}` : change.path}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </section>

          <section className={styles.diffPane} aria-label={t('reviewCenter.diff')}>
            <div className={styles.diffToolbar}>
              <span className={styles.selectedPath}>{selectedChange?.path || t('reviewCenter.noFile')}</span>
              {selectedChange && (
                <div className={styles.modeButtons}>
                  <Button size="sm" tone={!staged ? 'brand' : 'default'} onClick={() => setStaged(false)}>
                    {t('reviewCenter.workingTree')}
                  </Button>
                  <Button size="sm" tone={staged ? 'brand' : 'default'} onClick={() => setStaged(true)}>
                    {t('reviewCenter.staged')}
                  </Button>
                </div>
              )}
            </div>

            {loadingDiff ? (
              <div className={styles.empty}>
                <Spinner label={t('reviewCenter.loadingDiff')} />
              </div>
            ) : diff?.diff ? (
              <>
                {diff.truncated && (
                  <InlineAlert tone="warning">{t('reviewCenter.diffTruncated')}</InlineAlert>
                )}
                <pre className={styles.diff}>{diff.diff}</pre>
              </>
            ) : (
              <div className={styles.empty}>
                {selectedChange ? t('reviewCenter.noTextDiff') : t('reviewCenter.selectFile')}
              </div>
            )}
          </section>
        </div>

        <footer className={styles.footer}>
          <span>{t('reviewCenter.readOnly')}</span>
          <Button onClick={onClose}>{t('common.closeDialog')}</Button>
        </footer>
      </div>
    </Dialog>
  );
};
