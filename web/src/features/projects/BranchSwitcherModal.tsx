import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Check, GitBranch, Plus, RefreshCw } from 'lucide-react';
import {
  Button,
  Dialog,
  Input,
  SearchInput,
  Badge,
  Spinner,
  InlineAlert,
} from '../../design-system';
import { nexus } from '../../nexus/api';
import type { GitBranchesResult, Project } from '../../types';

interface BranchSwitcherModalProps {
  open: boolean;
  onClose: () => void;
  project: Project;
  onBranchChanged?: (newBranch: string) => void;
}

export const BranchSwitcherModal: React.FC<BranchSwitcherModalProps> = ({
  open,
  onClose,
  project,
  onBranchChanged,
}) => {
  const { t } = useTranslation();
  const [data, setData] = useState<GitBranchesResult | null>(null);
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(false);
  const [switching, setSwitching] = useState(false);
  const [showCreate, setShowCreate] = useState(false);
  const [newBranchName, setNewBranchName] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [successMsg, setSuccessMsg] = useState<string | null>(null);

  const loadBranches = useCallback(async () => {
    if (!project?.id) return;
    setLoading(true);
    setError(null);
    try {
      const res = await nexus.getProjectBranches(project.id);
      setData(res);
    } catch (err: any) {
      setError(err?.message || t('git.listFailed'));
    } finally {
      setLoading(false);
    }
  }, [project?.id, t]);

  useEffect(() => {
    if (open) {
      setSearch('');
      setShowCreate(false);
      setNewBranchName('');
      setError(null);
      setSuccessMsg(null);
      void loadBranches();
    }
  }, [open, loadBranches]);

  const handleCheckout = async (branchName: string, isNew = false) => {
    if (!branchName.trim() || switching) return;
    setSwitching(true);
    setError(null);
    setSuccessMsg(null);
    try {
      const res = await nexus.checkoutProjectBranch(project.id, branchName.trim(), isNew);
      if (res.success) {
        setSuccessMsg(t('git.switchedSuccess', { branch: res.current_branch }));
        onBranchChanged?.(res.current_branch);
        setTimeout(() => {
          onClose();
        }, 500);
      }
    } catch (err: any) {
      setError(err?.message || t('git.switchFailed'));
    } finally {
      setSwitching(false);
    }
  };

  const currentBranch = data?.current_branch || project.default_branch || 'main';
  const allBranches = data?.branches || [currentBranch];
  const filtered = allBranches.filter((b) => b.toLowerCase().includes(search.toLowerCase().trim()));

  return (
    <Dialog open={open} onClose={onClose} title={`${t('git.switchBranch')} · ${project.name}`}>
      <div className="nx-branch-switcher">
        {/* Current status bar */}
        <div className="nx-branch-status-row">
          <div className="nx-branch-current">
            <GitBranch size={14} className="nx-accent-icon" />
            <span>{t('git.currentBranch')}:</span>
            <strong>{currentBranch}</strong>
          </div>
          {data && (
            <Badge tone={data.is_clean ? 'success' : 'warning'}>
              {data.is_clean
                ? t('git.cleanTree')
                : t('git.dirtyTree', { count: data.modified_count })}
            </Badge>
          )}
        </div>

        {error && (
          <InlineAlert tone="danger" title={t('git.error')}>
            {error}
          </InlineAlert>
        )}

        {successMsg && <InlineAlert tone="success">{successMsg}</InlineAlert>}

        {/* Search Input & Action */}
        <div className="nx-branch-search-row">
          <SearchInput
            value={search}
            onChange={setSearch}
            placeholder={t('git.searchBranches')}
            autoFocus
          />
          <Button
            size="sm"
            tone={showCreate ? 'brand' : 'default'}
            onClick={() => setShowCreate(!showCreate)}
            title={t('git.newBranch')}
          >
            <Plus size={13} /> {t('git.newBranchShort')}
          </Button>
          <Button size="sm" onClick={loadBranches} disabled={loading} title={t('common.refresh')}>
            <RefreshCw size={13} className={loading ? 'nx-spin' : ''} />
          </Button>
        </div>

        {/* Create new branch input form */}
        {showCreate && (
          <div className="nx-branch-create-box">
            <label className="nx-field-label">
              <span>{t('git.createAndCheckout')}</span>
              <div className="nx-branch-create-input-group">
                <Input
                  value={newBranchName}
                  onChange={setNewBranchName}
                  placeholder="ex: feature/nova-tela, fix/bug-123"
                  mono
                  onEnter={() => handleCheckout(newBranchName, true)}
                />
                <Button
                  tone="brand"
                  size="sm"
                  disabled={!newBranchName.trim() || switching}
                  onClick={() => handleCheckout(newBranchName, true)}
                >
                  {switching ? <Spinner label="" /> : t('git.create')}
                </Button>
              </div>
            </label>
          </div>
        )}

        {/* Branches list */}
        <div className="nx-branch-list-container">
          {loading && !data ? (
            <div className="nx-branch-loading">
              <Spinner label={t('git.loadingBranches')} />
            </div>
          ) : filtered.length === 0 ? (
            <div className="nx-branch-empty">
              <p>{t('git.noBranchesFound')}</p>
              {search.trim() && !allBranches.includes(search.trim()) && (
                <Button size="sm" tone="brand" onClick={() => handleCheckout(search.trim(), true)}>
                  <Plus size={12} /> {t('git.createBranchNamed', { name: search.trim() })}
                </Button>
              )}
            </div>
          ) : (
            <ul className="nx-branch-items" role="listbox" aria-label={t('git.branchesAriaLabel')}>
              {filtered.map((branch) => {
                const isActive = branch === currentBranch;
                return (
                  <li key={branch}>
                    <button
                      type="button"
                      role="option"
                      aria-selected={isActive}
                      className={`nx-branch-item ${isActive ? 'nx-branch-item--active' : ''}`}
                      onClick={() => handleCheckout(branch, false)}
                      disabled={switching}
                    >
                      <div className="nx-branch-item-name">
                        <GitBranch size={13} className="nx-branch-icon" />
                        <span className="nx-branch-text">{branch}</span>
                      </div>
                      {isActive && (
                        <span className="nx-branch-active-badge">
                          <Check size={12} /> {t('git.active')}
                        </span>
                      )}
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </div>

        {/* Remote branches hint */}
        {data && data.remote_branches && data.remote_branches.length > 0 && (
          <div className="nx-branch-footer-info">
            <small className="nx-muted-copy">
              {t('git.remotesDetected', { count: data.remote_branches.length })}
            </small>
          </div>
        )}

        <div className="nx-dialog-actions">
          <Button onClick={onClose}>{t('common.closeDialog')}</Button>
        </div>
      </div>
    </Dialog>
  );
};
