import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { FileText, Paperclip, X } from 'lucide-react';
import { Button, IconButton, Select } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ContextFileRef, FSEntry } from '../../types';
import { asArray } from '../../lib/safeArray';
import styles from './ComposerSurface.module.scss';

const PARENT_VALUE = '__parent__';
const DIR_PREFIX = 'dir:';
const FILE_PREFIX = 'file:';

/**
 * Attached-file affordance for the elaboration conversation: shows the files
 * already grounding the prompt (from brief.context.referenced_files) and a
 * lightweight browser (via browseFS) to attach a new project file. Mutations
 * are delegated to the orchestrator through onAttach/onDetach so the session
 * view stays the single source of truth.
 */
export const ComposerContextChips: React.FC<{
  projectPath: string;
  files: ContextFileRef[];
  busy: boolean;
  onAttach: (path: string) => void;
  onDetach: (path: string) => void;
}> = ({ projectPath, files, busy, onAttach, onDetach }) => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [entries, setEntries] = useState<FSEntry[]>([]);
  const [parentPath, setParentPath] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const loadBrowse = useCallback(async (path?: string) => {
    setLoading(true);
    setError('');
    try {
      const result = await nexus.browseFS(path);
      setEntries(asArray<FSEntry>(result.entries));
      setParentPath(result.parent_path || '');
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setEntries([]);
      setParentPath('');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (open) void loadBrowse(projectPath);
  }, [open, projectPath, loadBrowse]);

  const options = useMemo(() => {
    const opts: { value: string; label: string }[] = [];
    if (parentPath) {
      opts.push({ value: PARENT_VALUE, label: t('work.composer.contextParent', 'Parent folder') });
    }
    for (const entry of entries) {
      if (entry.is_dir) {
        opts.push({ value: `${DIR_PREFIX}${entry.path}`, label: `${entry.name}/` });
      }
    }
    for (const entry of entries) {
      if (!entry.is_dir) {
        opts.push({ value: `${FILE_PREFIX}${entry.path}`, label: entry.name });
      }
    }
    return opts;
  }, [entries, parentPath, t]);

  const onPick = (value: string) => {
    if (!value) return;
    if (value === PARENT_VALUE) {
      void loadBrowse(parentPath);
      return;
    }
    if (value.startsWith(DIR_PREFIX)) {
      void loadBrowse(value.slice(DIR_PREFIX.length));
      return;
    }
    if (value.startsWith(FILE_PREFIX)) {
      onAttach(value.slice(FILE_PREFIX.length));
      setOpen(false);
    }
  };

  const attached = asArray<ContextFileRef>(files);
  const noFiles = !loading && !error && !entries.some((entry) => !entry.is_dir);

  return (
    <div className={styles.contextChips}>
      <div className={styles.contextChipsHeader}>
        <span className={styles.contextChipsLabel}>
          {t('work.composer.contextFiles', 'Context files')}
        </span>
        <Button
          size="sm"
          disabled={busy}
          onClick={() => setOpen((value) => !value)}
          title={t(
            'work.composer.attachFileHint',
            'Attach a project file so its content grounds the prompt.',
          )}
        >
          <Paperclip size={13} /> {t('work.composer.attachFile', 'Attach file')}
        </Button>
      </div>

      {attached.length === 0 ? (
        <p className={styles.contextEmpty}>
          {t('work.composer.contextEmpty', 'No files attached yet.')}
        </p>
      ) : (
        <ul className={styles.contextChipList}>
          {attached.map((file) => (
            <li key={file.path} className={styles.contextChip}>
              <FileText size={12} />
              <span className={styles.contextChipPath}>{file.path}</span>
              <IconButton
                label={t('work.composer.removeContextFile', 'Remove {{path}}', { path: file.path })}
                disabled={busy}
                onClick={() => onDetach(file.path)}
                className={styles.contextChipRemove}
              >
                <X size={12} />
              </IconButton>
            </li>
          ))}
        </ul>
      )}

      {open && (
        <div className={styles.contextPicker}>
          <small className={styles.contextPickerTitle}>
            {t('work.composer.contextPickerTitle', 'Attach a project file')}
          </small>
          {error ? (
            <p className={styles.contextPickerHint}>
              {t('work.composer.contextBrowseError', 'Could not read this folder.')}
            </p>
          ) : (
            <>
              <Select
                value=""
                onChange={onPick}
                options={options}
                disabled={busy || loading}
                placeholder={t('work.composer.contextSelectPlaceholder', 'Browse and pick a file…')}
                aria-label={t('work.composer.contextPickerTitle', 'Attach a project file')}
              />
              {noFiles && (
                <p className={styles.contextPickerHint}>
                  {t('work.composer.contextNoFiles', 'No files in this folder.')}
                </p>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
};
