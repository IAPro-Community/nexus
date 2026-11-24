import React, { useEffect, useMemo, useState } from 'react';
import { History, RotateCcw, Save, Trash2 } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button, Dialog, Input } from '../design-system';
import {
  WorkspaceLayoutService,
  WORKSPACE_LAYOUT_VERSION,
  type WorkspaceNamedSnapshot,
} from '../services/WorkspaceLayoutService';
import { listSurfaces, surfaceViewId } from './model';
import { useWorkspace } from './WorkspaceProvider';
import { useWorkspacePresentation } from './WorkspacePresentationProvider';
import styles from './WorkspaceSnapshotsControl.module.scss';

export const WorkspaceSnapshotsControl: React.FC<{ projectId: string }> = ({ projectId }) => {
  const { t, i18n } = useTranslation();
  const workspace = useWorkspace();
  const presentation = useWorkspacePresentation();
  const service = useMemo(() => new WorkspaceLayoutService(projectId), [projectId]);
  const [open, setOpen] = useState(false);
  const [name, setName] = useState('');
  const [snapshots, setSnapshots] = useState<WorkspaceNamedSnapshot[]>([]);
  const [error, setError] = useState('');
  const locale = i18n.resolvedLanguage || i18n.language;

  const refreshSnapshots = () => setSnapshots(service.listNamedSnapshots());

  useEffect(() => {
    setOpen(false);
    setName('');
    setError('');
    setSnapshots(service.listNamedSnapshots());
  }, [projectId, service]);

  const formatDate = (value: string) => {
    const date = new Date(value);
    if (!Number.isFinite(date.getTime())) return '';
    return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(
      date,
    );
  };

  const saveSnapshot = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmedName = name.trim();
    if (!trimmedName) return;
    const surfaces = listSurfaces(workspace.model.root);
    try {
      service.saveNamedSnapshot(trimmedName, {
        version: WORKSPACE_LAYOUT_VERSION,
        workspaceId: projectId,
        revision: 1,
        updatedAt: new Date().toISOString(),
        model: workspace.model,
        presentation: presentation.state,
        activeSurfaceId:
          workspace.model.root.kind === 'stack' ? workspace.model.root.activeId : undefined,
        openSurfaceIds: surfaces.map(surfaceViewId),
      });
      setName('');
      setError('');
      refreshSnapshots();
    } catch {
      setError(t('workspace.snapshots.saveError'));
    }
  };

  const restoreSnapshot = (snapshotId: string) => {
    try {
      const restored = service.restoreNamedSnapshot(snapshotId, workspace.model);
      workspace.replaceModel(restored.model);
      presentation.replaceState(restored.presentation);
      setError('');
      setOpen(false);
    } catch {
      setError(t('workspace.snapshots.restoreError'));
    }
  };

  const deleteSnapshot = (snapshotId: string) => {
    if (!service.deleteNamedSnapshot(snapshotId)) {
      setError(t('workspace.snapshots.deleteError'));
      return;
    }
    setError('');
    refreshSnapshots();
  };

  return (
    <div className={styles.root}>
      <Button
        size="sm"
        tone="ghost"
        className={styles.trigger}
        data-tour="workspace-snapshots"
        aria-haspopup="dialog"
        aria-expanded={open}
        onClick={() => {
          refreshSnapshots();
          setError('');
          setOpen(true);
        }}
      >
        <History size={13} aria-hidden="true" />
        <span>{t('workspace.snapshots.open')}</span>
      </Button>

      <Dialog
        open={open}
        onClose={() => setOpen(false)}
        title={t('workspace.snapshots.title')}
        className={styles.dialog}
      >
        <div className={styles.content}>
          <p className={styles.description}>{t('workspace.snapshots.description')}</p>
          <form className={styles.form} onSubmit={saveSnapshot}>
            <label className={styles.field} htmlFor="workspace-snapshot-name">
              <span>{t('workspace.snapshots.nameLabel')}</span>
              <Input
                id="workspace-snapshot-name"
                name="snapshotName"
                value={name}
                onChange={setName}
                maxLength={64}
                autoComplete="off"
                placeholder={t('workspace.snapshots.namePlaceholder')}
              />
            </label>
            <Button type="submit" size="sm" disabled={!name.trim()}>
              <Save size={14} aria-hidden="true" />
              {t('workspace.snapshots.save')}
            </Button>
          </form>

          {error && (
            <p className={styles.error} role="alert">
              {error}
            </p>
          )}

          {snapshots.length === 0 ? (
            <p className={styles.empty}>{t('workspace.snapshots.empty')}</p>
          ) : (
            <ul className={styles.list} aria-label={t('workspace.snapshots.listLabel')}>
              {snapshots.map((snapshot) => (
                <li key={snapshot.id} className={styles.snapshot}>
                  <div className={styles.metadata}>
                    <strong>{snapshot.name}</strong>
                    <time dateTime={snapshot.createdAt}>{formatDate(snapshot.createdAt)}</time>
                  </div>
                  <div className={styles.actions}>
                    <Button size="sm" tone="ghost" onClick={() => restoreSnapshot(snapshot.id)}>
                      <RotateCcw size={13} aria-hidden="true" />
                      {t('workspace.snapshots.restore')}
                    </Button>
                    <Button
                      size="sm"
                      tone="ghost"
                      aria-label={t('workspace.snapshots.deleteNamed', { name: snapshot.name })}
                      onClick={() => deleteSnapshot(snapshot.id)}
                    >
                      <Trash2 size={13} aria-hidden="true" />
                      {t('workspace.snapshots.delete')}
                    </Button>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </Dialog>
    </div>
  );
};
