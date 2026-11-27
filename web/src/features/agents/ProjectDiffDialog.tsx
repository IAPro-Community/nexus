import React, { useCallback, useEffect, useState } from 'react';
import { FileDiff, RefreshCw, Send } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { toast } from 'sonner';
import { Badge, Button, Dialog, EmptyState, Spinner } from '../../design-system';
import { asArray } from '../../lib/safeArray';
import { nexus } from '../../nexus/api';
import type { Agent, GitDiffFile, GitDiffResult, GitWorktree, Project } from '../../types';
import styles from './ProjectDiffDialog.module.scss';

export function buildDiffCommentPrompt(
  diff: Pick<GitDiffResult, 'branch' | 'dir'>,
  comment: string,
  selectedFile?: string,
): string {
  const parts = [`[Git Diff Review · ${diff.branch || 'HEAD'}]`, `Workspace: ${diff.dir}`];
  if (selectedFile) {
    parts.push(`File: ${selectedFile}`);
  }
  parts.push('', comment.trim());
  return parts.join('\n');
}

export const ProjectDiffDialog: React.FC<{
  open: boolean;
  onClose: () => void;
  project: Project;
  agents: Agent[];
  onSentToAgent?: (agent: Agent) => void;
}> = ({ open, onClose, project, agents, onSentToAgent }) => {
  const { t } = useTranslation();
  const safeAgents = asArray<Agent>(agents);
  const [worktrees, setWorktrees] = useState<GitWorktree[]>([]);
  const [selectedWorktree, setSelectedWorktree] = useState('');
  const [diff, setDiff] = useState<GitDiffResult | null>(null);
  const [loading, setLoading] = useState(false);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState('');
  const [selectedFile, setSelectedFile] = useState('');
  const [targetAgentId, setTargetAgentId] = useState('');
  const [comment, setComment] = useState('');

  useEffect(() => {
    if (!targetAgentId && safeAgents.length > 0) {
      setTargetAgentId(safeAgents[0].id);
    }
  }, [safeAgents, targetAgentId]);

  const loadDiff = useCallback(
    async (wt?: string) => {
      setLoading(true);
      setError('');
      try {
        const [diffRes, branchesRes] = await Promise.all([
          nexus.getProjectGitDiff(project.id, wt),
          nexus.getProjectBranches(project.id),
        ]);
        setDiff(diffRes);
        setWorktrees(asArray<GitWorktree>(branchesRes.worktrees));
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setLoading(false);
      }
    },
    [project.id],
  );

  useEffect(() => {
    if (!open) return;
    void loadDiff(selectedWorktree || undefined);
  }, [loadDiff, open, selectedWorktree]);

  const files = asArray<GitDiffFile>(diff?.files);

  const handleSelectFile = (path: string) => {
    setSelectedFile(path);
    setComment((prev) => {
      const marker = `@${path} `;
      return prev.includes(marker) ? prev : `${marker}${prev}`;
    });
  };

  const handleSend = async () => {
    const target = safeAgents.find((a) => a.id === targetAgentId);
    if (!target || !comment.trim() || !diff) return;
    setSending(true);
    setError('');
    try {
      const prompt = buildDiffCommentPrompt(diff, comment, selectedFile || undefined);
      await nexus.askAgent(target.id, prompt, true);
      toast.success(t('agents.diffDialog.sentToast', { name: target.name }));
      setComment('');
      onSentToAgent?.(target);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSending(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} title={t('agents.diffDialog.title')} wide>
      <div className={styles.container}>
        <div className={styles.toolbar}>
          <div className={styles.metaRow}>
            <FileDiff size={15} />
            <Badge tone="default">{diff?.branch || project.default_branch || 'main'}</Badge>
            <span>{t('agents.diffDialog.filesCount', { count: files.length })}</span>
            {diff?.truncated ? (
              <Badge tone="warning">{t('agents.diffDialog.truncated')}</Badge>
            ) : null}
          </div>
          <div className={styles.feedbackRow}>
            {worktrees.length > 1 ? (
              <select
                aria-label={t('agents.diffDialog.worktreeLabel')}
                className={styles.selectInput}
                value={selectedWorktree}
                onChange={(e) => setSelectedWorktree(e.currentTarget.value)}
              >
                <option value="">{t('agents.diffDialog.primaryWorktree')}</option>
                {worktrees.map((wt) => (
                  <option key={wt.path} value={wt.path}>
                    {wt.branch || wt.path}
                  </option>
                ))}
              </select>
            ) : null}
            <Button
              size="sm"
              tone="ghost"
              disabled={loading}
              onClick={() => void loadDiff(selectedWorktree || undefined)}
            >
              <RefreshCw size={13} /> {t('common.refresh')}
            </Button>
          </div>
        </div>

        {error ? <p className={styles.errorText}>{error}</p> : null}

        {loading && !diff ? (
          <Spinner />
        ) : diff?.is_clean && files.length === 0 ? (
          <EmptyState
            title={t('agents.diffDialog.cleanTitle')}
            hint={t('agents.diffDialog.cleanHint')}
          />
        ) : (
          <div className={styles.layout}>
            <div className={styles.fileList} aria-label={t('agents.diffDialog.filesLabel')}>
              {files.map((file) => (
                <button
                  key={file.path}
                  type="button"
                  className={styles.fileButton}
                  onClick={() => handleSelectFile(file.path)}
                  title={file.path}
                >
                  <span className={styles.filePath}>{file.path}</span>
                  <span className={styles.fileStats}>
                    +{file.additions} -{file.deletions}
                  </span>
                </button>
              ))}
            </div>
            <div className={styles.diffPane}>
              <pre className={styles.diffCode}>
                {diff?.diff_text || t('agents.diffDialog.untrackedOnly')}
              </pre>
            </div>
          </div>
        )}

        <div className={styles.feedbackForm}>
          <div className={styles.feedbackRow}>
            <label htmlFor="diff-review-agent" className="nx-sr-only">
              {t('agents.diffDialog.agentLabel')}
            </label>
            <select
              id="diff-review-agent"
              className={styles.selectInput}
              value={targetAgentId}
              onChange={(e) => setTargetAgentId(e.currentTarget.value)}
            >
              {safeAgents.map((agent) => (
                <option key={agent.id} value={agent.id}>
                  {agent.name} ({agent.role || 'developer'})
                </option>
              ))}
            </select>
            <Button
              size="sm"
              tone="brand"
              disabled={sending || !targetAgentId || !comment.trim() || !diff}
              onClick={() => void handleSend()}
            >
              <Send size={13} /> {t('agents.diffDialog.sendToAgent')}
            </Button>
          </div>
          <textarea
            className={styles.textarea}
            aria-label={t('agents.diffDialog.commentLabel')}
            placeholder={t('agents.diffDialog.commentPlaceholder')}
            value={comment}
            onChange={(e) => setComment(e.currentTarget.value)}
          />
        </div>
      </div>
    </Dialog>
  );
};
