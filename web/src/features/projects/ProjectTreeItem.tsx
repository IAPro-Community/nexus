import React, { useMemo } from 'react';
import { AlertTriangle, ChevronDown, ChevronRight, TerminalSquare } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { Agent, AttentionItem, Project, ProjectAgentSummary } from '../../types';
import { contextMenuFromEvent, type ContextMenuPoint } from '../../design-system';
import { formatOpenedAgo } from './projectRailRelativeTime';
import type { ProjectOutlineAttention } from './useProjectRailOutline';
import styles from './ProjectTreeItem.module.scss';

const MAX_FAILED = 3;

export type ProjectTreeItemProps = {
  project: Project;
  selected: boolean;
  expanded: boolean;
  summary?: ProjectAgentSummary;
  attention?: ProjectOutlineAttention;
  agents: Agent[];
  agentsLoading: boolean;
  onToggleExpand: () => void;
  onSelect: () => void;
  onOpenAgent: (agent: Agent) => void;
  onOpenAttention: (item: AttentionItem) => void;
  onProjectContextMenu: (project: Project, point: ContextMenuPoint) => void;
  onAgentContextMenu: (agent: Agent, point: ContextMenuPoint) => void;
};

export const ProjectTreeItem: React.FC<ProjectTreeItemProps> = ({
  project,
  selected,
  expanded,
  summary,
  attention,
  agents,
  agentsLoading,
  onToggleExpand,
  onSelect,
  onOpenAgent,
  onOpenAttention,
  onProjectContextMenu,
  onAgentContextMenu,
}) => {
  const { t, i18n } = useTranslation();
  const needsCount = attention?.needsYou.length ?? 0;
  const workingCount = summary?.working_count ?? 0;
  const failed = (attention?.failed ?? []).slice(0, MAX_FAILED);
  const needsYou = attention?.needsYou ?? [];

  const openedLabel = useMemo(
    () => formatOpenedAgo(project.last_opened_at, i18n.language || 'en', t('rail.neverOpened')),
    [project.last_opened_at, i18n.language, t],
  );

  const tooltip = [
    project.canonical_path,
    project.default_branch ? `${t('rail.branch')}: ${project.default_branch}` : null,
  ]
    .filter(Boolean)
    .join(' · ');

  const childrenEmpty =
    !agentsLoading && needsYou.length === 0 && failed.length === 0 && agents.length === 0;

  const onOpenKeyDown = (event: React.KeyboardEvent) => {
    if (event.key === 'ArrowRight' && !expanded) {
      event.preventDefault();
      onToggleExpand();
    } else if (event.key === 'ArrowLeft' && expanded) {
      event.preventDefault();
      onToggleExpand();
    } else if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault();
      onSelect();
    }
  };

  return (
    <div className={styles.item} role="treeitem" aria-expanded={expanded} aria-selected={selected}>
      <div
        className={styles.header}
        data-active={selected ? 'true' : 'false'}
        title={tooltip}
        onContextMenu={(event) => {
          event.preventDefault();
          onProjectContextMenu(project, contextMenuFromEvent(event));
        }}
      >
        <button
          type="button"
          className={styles.chevron}
          aria-label={expanded ? t('rail.collapseProject') : t('rail.expandProject')}
          aria-expanded={expanded}
          onClick={(event) => {
            event.stopPropagation();
            onToggleExpand();
          }}
        >
          {expanded ? (
            <ChevronDown size={14} aria-hidden />
          ) : (
            <ChevronRight size={14} aria-hidden />
          )}
        </button>
        <button type="button" className={styles.open} onClick={onSelect} onKeyDown={onOpenKeyDown}>
          <span className={styles.avatar} aria-hidden>
            {(project.name || 'PR').slice(0, 2).toUpperCase()}
          </span>
          <span className={styles.copy}>
            <span className={styles.name}>{project.name || project.id}</span>
            <span className={styles.meta}>{openedLabel}</span>
          </span>
          <span className={styles.signals}>
            {needsCount > 0 && (
              <span
                className={`${styles.signal} ${styles.signalNeeds}`}
                title={t('rail.needsYouCount', { count: needsCount })}
              >
                <AlertTriangle size={10} aria-hidden />
                {needsCount}
              </span>
            )}
            {workingCount > 0 && (
              <span
                className={`${styles.signal} ${styles.signalWorking}`}
                title={t('rail.workingAgentsCount', { count: workingCount })}
              >
                {workingCount}
              </span>
            )}
            {selected && <span className={styles.activeDot} aria-hidden />}
          </span>
        </button>
      </div>

      {expanded && (
        <div className={styles.children} role="group" aria-label={t('rail.projectOutline')}>
          {agentsLoading && <p className={styles.loading}>{t('common.loading')}</p>}
          {!agentsLoading &&
            needsYou.map((item) => (
              <button
                type="button"
                key={`need-${item.mission_id}`}
                className={styles.child}
                onClick={() => onOpenAttention(item)}
                title={item.question || item.summary}
              >
                <AlertTriangle size={12} className={styles.childToneNeeds} aria-hidden />
                <span className={styles.childCopy}>
                  <strong>{item.summary || t('rail.needsYou')}</strong>
                  <small>{item.question || item.state}</small>
                </span>
              </button>
            ))}
          {!agentsLoading &&
            failed.map((item) => (
              <button
                type="button"
                key={`fail-${item.mission_id}`}
                className={styles.child}
                onClick={() => onOpenAttention(item)}
                title={item.summary}
              >
                <AlertTriangle size={12} className={styles.childToneFailed} aria-hidden />
                <span className={styles.childCopy}>
                  <strong>{item.summary || t('rail.failedRun')}</strong>
                  <small>{item.state}</small>
                </span>
              </button>
            ))}
          {!agentsLoading &&
            agents.map((agent) => (
              <button
                type="button"
                key={agent.id}
                className={styles.child}
                onClick={() => onOpenAgent(agent)}
                onContextMenu={(event) => {
                  event.preventDefault();
                  onAgentContextMenu(agent, contextMenuFromEvent(event));
                }}
                title={`${agent.name} · ${agent.role || t('agents.developer')} (${agent.status})`}
              >
                <span className="nx-status-dot" data-status={agent.status} />
                <span className={styles.childCopy}>
                  <strong>{agent.name}</strong>
                  <small>{agent.role || t('agents.developer')}</small>
                </span>
                <TerminalSquare size={12} aria-hidden />
              </button>
            ))}
          {childrenEmpty && <p className={styles.empty}>{t('rail.emptyOutline')}</p>}
        </div>
      )}
    </div>
  );
};
