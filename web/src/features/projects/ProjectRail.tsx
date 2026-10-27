import {
  COMPOSER_FLOW_ENABLED,
  isExperimentalComposerFlowSurface,
} from '../../app/composerFlowFlag';
import React, { useCallback, useMemo, useState } from 'react';
import {
  Gauge,
  History,
  Home,
  Layers,
  LayoutGrid,
  Images,
  Plus,
  Settings,
  Sparkles,
  TerminalSquare,
  Workflow,
  X,
  Pencil,
  Trash2,
} from 'lucide-react';
import {
  Button,
  IconButton,
  ContextMenu,
  Dialog,
  ConfirmDialog,
  Input,
  InlineAlert,
  type ContextMenuPoint,
} from '../../design-system';
import { AddProjectModal } from './AddProjectModal';
import { ProjectTreeItem } from './ProjectTreeItem';
import { useProjectRailOutline } from './useProjectRailOutline';
import type { Agent, AttentionItem, Project } from '../../types';
import { useTranslation } from 'react-i18next';
import { nexus, NexusAPIError } from '../../nexus/api';
import styles from './ProjectRail.module.scss';
import { toast } from 'sonner';

export const ProjectRail: React.FC<{
  projects: Project[];
  selected?: Project | null;
  open: boolean;
  onClose: () => void;
  onSelect: (project: Project) => void;
  onCreated: (project: Project) => void;
  onProjectUpdated?: (project: Project) => void;
  onProjectDeleted?: (project: Project) => void;
  onOpenGlobal: (
    kind:
      | 'projects'
      | 'overview'
      | 'agents'
      | 'outputs'
      | 'resources'
      | 'maestro'
      | 'sessions'
      | 'settings'
      | 'work'
      | 'missions'
      | 'terminals',
  ) => void;
  onOpenAgent?: (agent: Agent) => void;
  onOpenAttention?: (project: Project, item: AttentionItem) => void;
  onNewAgent?: () => void;
  onNewAISession?: () => void;
  onProjectShell?: () => void;
}> = ({
  projects,
  selected,
  open,
  onClose,
  onSelect,
  onCreated,
  onProjectUpdated,
  onProjectDeleted,
  onOpenGlobal,
  onOpenAgent,
  onOpenAttention,
  onNewAgent: _onNewAgent,
  onNewAISession: _onNewAISession,
  onProjectShell: _onProjectShell,
}) => {
  const { t } = useTranslation();
  const outline = useProjectRailOutline();
  const { toggleExpanded } = outline;
  const [addOpen, setAddOpen] = useState(false);
  const [renameProject, setRenameProject] = useState<Project | null>(null);
  const [renameValue, setRenameValue] = useState('');
  const [renameBusy, setRenameBusy] = useState(false);
  const [renameError, setRenameError] = useState<string | null>(null);
  const [deleteProject, setDeleteProject] = useState<Project | null>(null);
  const [deleteBusy, setDeleteBusy] = useState(false);
  const [railMenu, setRailMenu] = useState<
    | { kind: 'project'; project: Project; point: ContextMenuPoint }
    | { kind: 'agent'; agent: Agent; point: ContextMenuPoint }
    | null
  >(null);

  const [projectsExpanded, setProjectsExpanded] = useState<boolean>(() => {
    try {
      const saved = localStorage.getItem('nx_rail_projects_open');
      return saved !== null ? saved === 'true' : true;
    } catch {
      return true;
    }
  });
  const [toolsExpanded, setToolsExpanded] = useState<boolean>(() => {
    try {
      const saved = localStorage.getItem('nx_rail_tools_open');
      return saved !== null ? saved === 'true' : true;
    } catch {
      return true;
    }
  });

  const [searchQuery, setSearchQuery] = useState('');

  const toggleProjects = () => {
    setProjectsExpanded((prev) => {
      const next = !prev;
      try {
        localStorage.setItem('nx_rail_projects_open', String(next));
      } catch {
        /* ignore */
      }
      return next;
    });
  };

  const toggleTools = () => {
    setToolsExpanded((prev) => {
      const next = !prev;
      try {
        localStorage.setItem('nx_rail_tools_open', String(next));
      } catch {
        /* ignore */
      }
      return next;
    });
  };

  const q = searchQuery.trim().toLowerCase();
  const filteredProjects = useMemo(() => {
    if (!q) return projects;
    return projects.filter((project) => {
      const nameHit =
        (project.name || project.id).toLowerCase().includes(q) ||
        (project.canonical_path || '').toLowerCase().includes(q);
      if (nameHit) return true;
      const agents = outline.agentsByProject.get(project.id) ?? [];
      if (
        agents.some(
          (agent) =>
            (agent.name || '').toLowerCase().includes(q) ||
            (agent.role || '').toLowerCase().includes(q),
        )
      ) {
        return true;
      }
      const attention = outline.attentionByProject.get(project.id);
      const attentionItems = [...(attention?.needsYou ?? []), ...(attention?.failed ?? [])];
      return attentionItems.some(
        (item) =>
          (item.summary || '').toLowerCase().includes(q) ||
          (item.question || '').toLowerCase().includes(q),
      );
    });
  }, [projects, q, outline.agentsByProject, outline.attentionByProject]);

  const optionalTools = (
    [
      { id: 'overview', label: t('nav.overview'), icon: Home },
      { id: 'terminals', label: t('nav.terminals'), icon: TerminalSquare },
      { id: 'maestro', label: t('nav.maestro'), icon: Sparkles },
      { id: 'work', label: t('nav.work'), icon: Layers },
      { id: 'missions', label: t('nav.missions'), icon: Workflow },
      { id: 'agents', label: t('nav.agents'), icon: TerminalSquare },
      { id: 'outputs', label: 'Outputs · Arquivos', icon: Images },
      { id: 'resources', label: t('nav.resources'), icon: Gauge },
      { id: 'sessions', label: t('nav.sessions'), icon: History },
      { id: 'projects', label: t('projectManager.desktopsTitle'), icon: LayoutGrid },
      { id: 'settings', label: t('nav.settings'), icon: Settings },
    ] as const
  ).filter((item) => COMPOSER_FLOW_ENABLED || !isExperimentalComposerFlowSurface(item.id));

  const selectProject = useCallback(
    (project: Project) => {
      onSelect(project);
      onClose();
    },
    [onSelect, onClose],
  );

  const handleToggleExpand = useCallback(
    (projectId: string) => {
      toggleExpanded(projectId);
    },
    [toggleExpanded],
  );

  const handleOpenAgent = useCallback(
    (agent: Agent) => {
      onOpenAgent?.(agent);
      onClose();
    },
    [onOpenAgent, onClose],
  );

  const openRename = (project: Project) => {
    setRenameProject(project);
    setRenameValue(project.name || '');
    setRenameError(null);
  };

  const saveRename = async () => {
    if (!renameProject) return;
    const name = renameValue.trim();
    if (!name) {
      setRenameError(t('rail.renameRequired'));
      return;
    }
    setRenameBusy(true);
    setRenameError(null);
    try {
      const updated = await nexus.updateProject(renameProject.id, { name });
      onProjectUpdated?.(updated);
      setRenameProject(null);
    } catch (error) {
      setRenameError(error instanceof Error ? error.message : t('rail.renameError'));
    } finally {
      setRenameBusy(false);
    }
  };

  const executeDelete = async () => {
    if (!deleteProject) return;
    setDeleteBusy(true);
    try {
      await nexus.deleteProject(deleteProject.id);
      onProjectDeleted?.(deleteProject);
      setDeleteProject(null);
    } catch (error) {
      setDeleteProject(null);
      const message =
        error instanceof NexusAPIError && error.status === 409
          ? t('rail.deleteBlocked')
          : t('rail.deleteError');
      toast.error(message);
    } finally {
      setDeleteBusy(false);
    }
  };

  return (
    <>
      <aside
        className={`${styles.rail} nx-project-rail`}
        data-open={open ? 'true' : 'false'}
        data-tour="projects"
        aria-label={t('rail.aria')}
      >
        <div className="nx-project-rail__brand">
          <span className="nx-brand-mark">
            <img src="./nexus-icon.png" alt="Nexus" className="nx-brand-mark__img" />
          </span>
          <span>
            <strong>IAPro Nexus</strong>
            <small>Workspace OS</small>
          </span>
          <IconButton
            className="nx-project-rail__mobile-close"
            label={t('rail.close')}
            onClick={onClose}
          >
            <X size={15} />
          </IconButton>
        </div>

        {(projects.length > 3 || searchQuery) && (
          <div className="nx-project-rail__search-wrap">
            <input
              type="search"
              className="nx-project-rail__search-input"
              placeholder={t('rail.filterPlaceholder')}
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              aria-label={t('rail.filterPlaceholder')}
            />
          </div>
        )}

        <div
          className="nx-rail-accordion-section"
          data-expanded={projectsExpanded ? 'true' : 'false'}
        >
          <div className="nx-project-rail__heading nx-rail-section-header">
            <button
              type="button"
              className="nx-rail-header-label"
              onClick={toggleProjects}
              aria-expanded={projectsExpanded}
            >
              <span className="nx-rail-chevron">{projectsExpanded ? '▾' : '▸'}</span>
              <span>{t('rail.projects')}</span>
              <span className="nx-rail-count">({filteredProjects.length})</span>
            </button>
            <span className="nx-rail-header-actions" onClick={(e) => e.stopPropagation()}>
              <IconButton label={t('rail.add')} onClick={() => setAddOpen(true)}>
                <Plus size={13} />
              </IconButton>
            </span>
          </div>

          {projectsExpanded && (
            <div
              className="nx-project-list nx-rail-scrollable nx-project-list--compact"
              role="tree"
              aria-label={t('rail.projects')}
            >
              {filteredProjects.map((project) => {
                const expanded = outline.expandedIds.has(project.id);
                return (
                  <ProjectTreeItem
                    key={project.id}
                    project={project}
                    selected={selected?.id === project.id}
                    expanded={expanded}
                    summary={outline.summaryMap.get(project.id)}
                    attention={
                      COMPOSER_FLOW_ENABLED ? outline.attentionByProject.get(project.id) : undefined
                    }
                    agents={outline.agentsByProject.get(project.id) ?? []}
                    agentsLoading={outline.loadingAgents.has(project.id)}
                    onToggleExpand={() => handleToggleExpand(project.id)}
                    onSelect={() => selectProject(project)}
                    onOpenAgent={handleOpenAgent}
                    onOpenAttention={(item) => {
                      onOpenAttention?.(project, item);
                      onClose();
                    }}
                    onProjectContextMenu={(proj, point) =>
                      setRailMenu({ kind: 'project', project: proj, point })
                    }
                    onAgentContextMenu={(agent, point) =>
                      setRailMenu({ kind: 'agent', agent, point })
                    }
                  />
                );
              })}
              {filteredProjects.length === 0 && (
                <p className="nx-rail-empty-msg">
                  {searchQuery ? t('rail.noProjectsFound') : t('rail.empty')}
                </p>
              )}
            </div>
          )}
        </div>

        <div
          className="nx-rail-accordion-section nx-rail-section-tools"
          data-expanded={toolsExpanded ? 'true' : 'false'}
        >
          <div className="nx-project-rail__heading nx-rail-section-header nx-rail-section-header--tools">
            <button
              type="button"
              className="nx-rail-header-label"
              onClick={toggleTools}
              aria-expanded={toolsExpanded}
            >
              <span className="nx-rail-chevron">{toolsExpanded ? '▾' : '▸'}</span>
              <span>{t('rail.tools')}</span>
            </button>
          </div>

          {toolsExpanded && (
            <div className="nx-project-rail__global nx-rail-tools-grid">
              {optionalTools.map((item) => (
                <button
                  type="button"
                  key={item.id}
                  className="nx-rail-tool-btn"
                  onClick={() => {
                    onOpenGlobal(item.id);
                    onClose();
                  }}
                  title={item.label}
                >
                  <item.icon size={13} />
                  <span>{item.label}</span>
                </button>
              ))}
            </div>
          )}
        </div>
      </aside>
      <ContextMenu
        open={railMenu?.point ?? null}
        onClose={() => setRailMenu(null)}
        label={t('workspace.menu')}
        items={
          railMenu?.kind === 'project'
            ? [
                {
                  type: 'item',
                  id: 'open-project',
                  label: t('workspace.openProject'),
                  icon: <LayoutGrid size={14} />,
                  onSelect: () => selectProject(railMenu.project),
                },
                {
                  type: 'item',
                  id: 'expand-project',
                  label: outline.expandedIds.has(railMenu.project.id)
                    ? t('rail.collapseProject')
                    : t('rail.expandProject'),
                  onSelect: () => outline.toggleExpanded(railMenu.project.id),
                },
                { type: 'separator', id: 'project-actions-separator' },
                {
                  type: 'item',
                  id: 'rename-project',
                  label: t('rail.rename'),
                  icon: <Pencil size={14} />,
                  onSelect: () => openRename(railMenu.project),
                },
                {
                  type: 'item',
                  id: 'delete-project',
                  label: t('rail.delete'),
                  icon: <Trash2 size={14} />,
                  danger: true,
                  onSelect: () => setDeleteProject(railMenu.project),
                },
              ]
            : railMenu?.kind === 'agent'
              ? [
                  {
                    type: 'item',
                    id: 'open-agent',
                    label: t('workspace.openAgent'),
                    onSelect: () => {
                      onOpenAgent?.(railMenu.agent);
                      onClose();
                    },
                  },
                ]
              : []
        }
      />

      <div
        className="nx-project-rail-overlay"
        data-open={open ? 'true' : 'false'}
        onClick={onClose}
      />

      <AddProjectModal
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onCreated={(project) => {
          onCreated(project);
          setAddOpen(false);
          void outline.refreshOutline();
        }}
      />

      <Dialog
        open={Boolean(renameProject)}
        onClose={() => setRenameProject(null)}
        title={t('rail.renameTitle')}
      >
        <form
          onSubmit={(event) => {
            event.preventDefault();
            void saveRename();
          }}
        >
          <label className="nx-field-label" htmlFor="project-rename-input">
            {t('rail.name')}
          </label>
          <Input
            id="project-rename-input"
            value={renameValue}
            onChange={setRenameValue}
            autoFocus
            maxLength={120}
          />
          {renameError && <InlineAlert tone="danger">{renameError}</InlineAlert>}
          <div className="nx-dialog-actions">
            <Button type="button" onClick={() => setRenameProject(null)}>
              {t('common.cancel')}
            </Button>
            <Button type="submit" tone="brand" disabled={renameBusy}>
              {t('rail.rename')}
            </Button>
          </div>
        </form>
      </Dialog>

      <ConfirmDialog
        open={Boolean(deleteProject)}
        title={t('rail.deleteTitle')}
        description={t('rail.deleteDescription', {
          name: deleteProject?.name || deleteProject?.id,
        })}
        confirmLabel={deleteBusy ? t('rail.deleting') : t('rail.delete')}
        onCancel={() => setDeleteProject(null)}
        onConfirm={() => void executeDelete()}
      />
    </>
  );
};
