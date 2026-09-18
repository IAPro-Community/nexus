import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { FolderGit2, GitBranch, Wifi, AlertTriangle } from 'lucide-react';
import { nexus } from '../nexus/api';
import { BranchSwitcherModal } from '../features/projects/BranchSwitcherModal';
import { Tooltip } from '../design-system';
import type { Agent, Project } from '../types';
import styles from './WorkspaceTaskbar.module.scss';

export const WorkspaceTaskbar: React.FC<{
  project?: Project;
  agents?: Agent[];
  onFocusAgent?: (agentId: string) => void;
}> = ({ project, agents = [], onFocusAgent }) => {
  const { t } = useTranslation();
  const [branchModalOpen, setBranchModalOpen] = useState(false);
  const [currentBranch, setCurrentBranch] = useState<string>('');
  const [sysInfo, setSysInfo] = useState<{
    nexus_version: string;
    maestro_version: string;
    maestro_available: boolean;
  } | null>(null);

  const [intelligence, setIntelligence] = useState<{
    available: boolean;
    provider?: string;
  } | null>(null);

  useEffect(() => {
    nexus
      .getSystemUpdates()
      .then(setSysInfo)
      .catch(() => undefined);
    nexus
      .getIntelligence(project?.id)
      .then(setIntelligence)
      .catch(() => undefined);
  }, [project?.id]);

  useEffect(() => {
    if (project?.default_branch) {
      setCurrentBranch(project.default_branch);
    }
  }, [project?.default_branch]);

  const working = agents.filter((a) => a.status === 'WORKING').length;
  // Agent health (FAILED/STALE/…) is not the same as Radar "needs you" waits.
  const degraded = agents.filter((a) =>
    ['FAILED', 'STALE', 'RECOVERABLE', 'RATE_LIMITED'].includes(a.status),
  ).length;

  return (
    <div
      className={`nx-workspace-statusbar ${styles.statusbar}`}
      data-tour="taskbar"
      role="status"
      aria-label="Status Bar"
    >
      {/* Left: Project & Branch */}
      <div className="nx-statusbar-left">
        {project && (
          <>
            <Tooltip content={project.canonical_path} side="top">
              <span className={`nx-statusbar-item nx-statusbar-project ${styles.projectItem}`}>
                <FolderGit2 size={12} className={styles.projectIcon} />
                <span className={styles.projectName}>{project.name}</span>
                {project.canonical_path && (
                  <span className={styles.canonicalPath}>{project.canonical_path}</span>
                )}
              </span>
            </Tooltip>
            <Tooltip
              content={t('git.switchBranchTooltip', 'Clique para alternar ou criar branches Git')}
              side="top"
            >
              <button
                type="button"
                className="nx-statusbar-item nx-statusbar-branch nx-statusbar-btn"
                onClick={() => setBranchModalOpen(true)}
              >
                <GitBranch size={11} />
                <span>{currentBranch || project.default_branch || 'unknown'}</span>
              </button>
            </Tooltip>

            <BranchSwitcherModal
              open={branchModalOpen}
              onClose={() => setBranchModalOpen(false)}
              project={project}
              onBranchChanged={(b) => setCurrentBranch(b)}
            />
          </>
        )}
      </div>

      {/* Center: Agents status */}
      <div className="nx-statusbar-center">
        {degraded > 0 ? (
          <Tooltip content="Clique para focar no agente com erro ou atenção" side="top">
            <button
              type="button"
              className={`nx-statusbar-item nx-statusbar-btn ${styles.degradedButton}`}
              onClick={() => {
                const degradedAgent = agents.find((a) =>
                  ['FAILED', 'STALE', 'RECOVERABLE', 'RATE_LIMITED'].includes(a.status),
                );
                if (degradedAgent && onFocusAgent) {
                  onFocusAgent(degradedAgent.id);
                }
              }}
            >
              <AlertTriangle size={12} />
              <span>{t('statusBar.degraded', { count: degraded })}</span>
            </button>
          </Tooltip>
        ) : working > 0 ? (
          <span className="nx-statusbar-item nx-statusbar-working">
            <span className="nx-status-dot nx-status-dot--working" />
            <span>{t('statusBar.working', { count: working })}</span>
          </span>
        ) : (
          <span className="nx-statusbar-item">
            <span className="nx-status-dot nx-status-dot--idle" />
            <span>{t('statusBar.ready')}</span>
          </span>
        )}
      </div>

      {/* Right: Versions & Connection */}
      <div className="nx-statusbar-right">
        <Tooltip content="Nexus Core Runtime" side="top">
          <span className="nx-statusbar-item nx-statusbar-version">
            <strong>Nexus</strong> v{sysInfo?.nexus_version || 'unknown'}
          </span>
        </Tooltip>
        {intelligence?.available && (
          <Tooltip
            content={`Inteligência Nexus ativa: ${intelligence.provider || 'default'}`}
            side="top"
          >
            <span className={`nx-statusbar-item ${styles.intelligenceStatus}`}>
              <span className={`nx-status-dot ${styles.accentDot}`} />
              <span>{intelligence.provider || 'Intelligence'}</span>
            </span>
          </Tooltip>
        )}
        <span className="nx-statusbar-item nx-statusbar-local">
          <Wifi size={11} />
          <span>{t('statusBar.local')}</span>
        </span>
      </div>
    </div>
  );
};
