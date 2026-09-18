import React, { useState, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import {
  LayoutGrid,
  List,
  Search,
  Plus,
  FolderGit2,
  GitBranch,
  Bot,
  Activity,
  Settings,
  ArrowRight,
  Trash2,
  ExternalLink,
  Copy,
  Check,
  Sparkles,
  Layers,
  Zap,
  FolderOpen,
  RefreshCw,
  Folder,
  Terminal,
  Code2,
} from 'lucide-react';
import {
  Button,
  Card,
  ConfirmDialog,
  Dialog,
  EmptyState,
  IconButton,
  Input,
  Badge,
  Segmented,
  Select,
  InlineAlert,
  Spinner,
} from '../../design-system';
import { toast } from 'sonner';
import { nexus } from '../../nexus/api';
import type { Agent, Project } from '../../types';
import { AddProjectModal } from './AddProjectModal';
import { DirectoryBrowserModal } from './DirectoryBrowserModal';
import { ProjectScanModal } from './ProjectScanModal';
import { selectProjectDirectory } from './projectDirectoryPicker';
import styles from './ProjectManagerSurface.module.scss';

export const ProjectManagerSurface: React.FC<{
  projects: Project[];
  selectedProject: Project;
  agents: Agent[];
  onSelectProject: (project: Project) => void;
  onProjectCreated: (project: Project) => void;
  onProjectUpdated: (project: Project) => void;
  onOpenOverview: () => void;
}> = ({
  projects,
  selectedProject,
  agents,
  onSelectProject,
  onProjectCreated,
  onProjectUpdated,
  onOpenOverview,
}) => {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');
  const [sortBy, setSortBy] = useState<'mru' | 'name'>('mru');
  const [addOpen, setAddOpen] = useState(false);
  const [templateOpen, setTemplateOpen] = useState(false);
  const [configProject, setConfigProject] = useState<Project | null>(null);
  const [copiedId, setCopiedId] = useState<string | null>(null);
  const [dirPickerOpen, setDirPickerOpen] = useState(false);
  const [scanModalOpen, setScanModalOpen] = useState(false);
  const [projectToDelete, setProjectToDelete] = useState<Project | null>(null);
  const [newPath, setNewPath] = useState('');
  const [newName, setNewName] = useState('');

  // Project configuration state
  const [cfgName, setCfgName] = useState('');
  const [cfgBranch, setCfgBranch] = useState('');
  const [cfgIsolation, setCfgIsolation] = useState('project');
  const [cfgPolicy, setCfgPolicy] = useState('BALANCED');
  const [cfgBusy, setCfgBusy] = useState(false);
  const [cfgSuccess, setCfgSuccess] = useState(false);
  const [cfgBranchList, setCfgBranchList] = useState<string[]>([]);
  const [cfgBranchStatus, setCfgBranchStatus] = useState<{
    isClean: boolean;
    count: number;
  } | null>(null);
  const [_cfgBranchLoading, setCfgBranchLoading] = useState(false);
  const [cfgBranchSwitching, setCfgBranchSwitching] = useState(false);
  const [cfgBranchAlert, setCfgBranchAlert] = useState<{
    tone: 'success' | 'danger';
    message: string;
  } | null>(null);

  // Stats calculation
  const totalAgents = agents.length;
  const workingAgents = agents.filter((a) => a.status === 'WORKING').length;

  const handleCopyPath = (id: string, path: string) => {
    navigator.clipboard.writeText(path);
    setCopiedId(id);
    setTimeout(() => setCopiedId(null), 2000);
  };

  const filtered = useMemo(() => {
    let list = projects.filter(
      (p) =>
        p.name.toLowerCase().includes(query.toLowerCase()) ||
        p.canonical_path.toLowerCase().includes(query.toLowerCase()) ||
        (p.default_branch || '').toLowerCase().includes(query.toLowerCase()),
    );
    if (sortBy === 'name') {
      list = [...list].sort((a, b) => a.name.localeCompare(b.name));
    }
    return list;
  }, [projects, query, sortBy]);

  const handleOpenConfig = async (p: Project) => {
    setConfigProject(p);
    setCfgName(p.name);
    setCfgBranch(p.default_branch || 'main');
    setCfgIsolation(p.default_isolation || 'project');
    setCfgPolicy(p.resource_policy || 'BALANCED');
    setCfgSuccess(false);
    setCfgBranchAlert(null);
    setCfgBranchLoading(true);
    try {
      const res = await nexus.getProjectBranches(p.id);
      setCfgBranchList(res.branches || []);
      setCfgBranchStatus({ isClean: res.is_clean, count: res.modified_count });
      if (res.current_branch) {
        setCfgBranch(res.current_branch);
      }
    } catch {
      setCfgBranchList([]);
      setCfgBranchStatus(null);
    } finally {
      setCfgBranchLoading(false);
    }
  };

  const handleDirectCheckout = async (targetBranch: string) => {
    if (!configProject || !targetBranch.trim() || cfgBranchSwitching) return;
    setCfgBranchSwitching(true);
    setCfgBranchAlert(null);
    try {
      const res = await nexus.checkoutProjectBranch(configProject.id, targetBranch.trim());
      if (res.success) {
        setCfgBranch(res.current_branch);
        setCfgBranchAlert({
          tone: 'success',
          message: `✓ Alternado com sucesso para branch ${res.current_branch}`,
        });
        const updated = { ...configProject, default_branch: res.current_branch };
        onProjectUpdated(updated);
      }
    } catch (err: any) {
      setCfgBranchAlert({ tone: 'danger', message: err?.message || 'Falha ao trocar de branch' });
    } finally {
      setCfgBranchSwitching(false);
    }
  };

  const handleSaveConfig = async () => {
    if (!configProject) return;
    setCfgBusy(true);
    try {
      const updated = await nexus.updateProject(configProject.id, {
        name: cfgName.trim() || configProject.name,
        default_branch: cfgBranch.trim() || 'main',
        default_isolation: cfgIsolation,
        resource_policy: cfgPolicy,
      });
      onProjectUpdated(updated);
      setCfgSuccess(true);
      setTimeout(() => {
        setConfigProject(null);
      }, 800);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setCfgBusy(false);
    }
  };

  const handleDelete = (p: Project) => {
    setProjectToDelete(p);
  };

  const executeDelete = async () => {
    if (!projectToDelete) return;
    const p = projectToDelete;
    setProjectToDelete(null);
    try {
      await nexus.deleteProject(p.id);
      window.location.reload();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  };

  const handleOpenOS = (p: Project, action: 'filemanager' | 'terminal' | 'editor') => {
    nexus.openProjectInOS(p.id, action).catch((err) => {
      console.error('Failed to launch OS action', err);
    });
  };

  const handleDirectorySelected = (path: string, suggestedName?: string) => {
    setNewPath(path);
    if (suggestedName && !newName) {
      setNewName(suggestedName);
    }
    setAddOpen(true);
  };

  return (
    <div className="nx-surface-scroll nx-projects-surface">
      {/* Header & Overview Stats */}
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">{t('projectManager.desktopsTitle')}</span>
          <h1>{t('projectManager.title')}</h1>
          <p>{t('projectManager.desktopsSubtitle')}</p>
        </div>
        <div className="nx-page-header__actions">
          <Button tone="brand" onClick={() => setAddOpen(true)}>
            <Plus size={14} />
            <span>{t('projectManager.addNew')}</span>
          </Button>
          <Button
            tone="ghost"
            onClick={async () => {
              const selection = await selectProjectDirectory(t('projectManager.browseOS'));
              if (selection.supported) {
                if (selection.path) {
                  setNewPath(selection.path);
                  setNewName(selection.path.split(/[/\\]/).filter(Boolean).pop() || 'Project');
                  setAddOpen(true);
                }
                return;
              }
              setDirPickerOpen(true);
            }}
          >
            <FolderOpen size={14} />
            <span>{t('projectManager.browseOS')}</span>
          </Button>
          <Button tone="ghost" onClick={() => setScanModalOpen(true)}>
            <RefreshCw size={14} />
            <span>{t('projectManager.scanOS')}</span>
          </Button>
          <Button onClick={() => setTemplateOpen(true)}>
            <Sparkles size={14} />
            <span>{t('projectManager.templatesTitle')}</span>
          </Button>
        </div>
      </div>

      {/* OS Metrics Grid */}
      <div className="nx-metric-grid">
        <Card className="nx-metric-card">
          <span className="nx-metric-card__icon">
            <FolderGit2 size={18} />
          </span>
          <div>
            <strong>{projects.length}</strong>
            <span>{t('projectManager.statsWorkspaces')}</span>
          </div>
        </Card>

        <Card className="nx-metric-card">
          <span className="nx-metric-card__icon">
            <Bot size={18} />
          </span>
          <div>
            <strong>{totalAgents}</strong>
            <span>{t('projectManager.statsAgents')}</span>
          </div>
        </Card>

        <Card className="nx-metric-card">
          <span className="nx-metric-card__icon">
            <Activity size={18} />
          </span>
          <div>
            <strong>{workingAgents}</strong>
            <span>{t('projectManager.statsWorking')}</span>
          </div>
        </Card>

        <Card className="nx-metric-card">
          <span className="nx-metric-card__icon">
            <Layers size={18} />
          </span>
          <div>
            <strong>{selectedProject.default_isolation || 'project'}</strong>
            <span>{t('projectManager.isolation')}</span>
          </div>
        </Card>
      </div>

      {/* Action Toolbar */}
      <div className="nx-projects-toolbar">
        <div className="nx-projects-search">
          <Search size={14} className="nx-projects-search-icon" />
          <Input
            value={query}
            onChange={setQuery}
            placeholder={t('projectManager.searchPlaceholder')}
          />
        </div>

        <div className="nx-projects-toolbar-actions">
          <Segmented
            options={[
              { value: 'mru', label: t('projectManager.sortMRU') },
              { value: 'name', label: t('projectManager.sortName') },
            ]}
            value={sortBy}
            onChange={(v) => setSortBy(v as any)}
            ariaLabel={t('projectManager.sortBy')}
          />

          <div className="nx-view-toggles">
            <IconButton
              label={t('projectManager.viewGrid')}
              onClick={() => setViewMode('grid')}
              tone={viewMode === 'grid' ? 'brand' : 'ghost'}
            >
              <LayoutGrid size={15} />
            </IconButton>
            <IconButton
              label={t('projectManager.viewList')}
              onClick={() => setViewMode('list')}
              tone={viewMode === 'list' ? 'brand' : 'ghost'}
            >
              <List size={15} />
            </IconButton>
          </div>
        </div>
      </div>

      {/* Grid View */}
      {viewMode === 'grid' && (
        <div className="nx-desktop-grid">
          {filtered.map((proj) => {
            const isActive = proj.id === selectedProject.id;
            return (
              <Card
                key={proj.id}
                className={`nx-desktop-card ${isActive ? 'nx-desktop-card--active' : ''}`}
                onClick={() => onSelectProject(proj)}
              >
                <div className="nx-desktop-card__header">
                  <div className="nx-desktop-card__brand">
                    <span className="nx-desktop-avatar">
                      {(proj.name || 'PR').slice(0, 2).toUpperCase()}
                    </span>
                    <div className="nx-desktop-title-stack">
                      <strong>{proj.name || proj.id}</strong>
                      <span className="nx-desktop-branch">
                        <GitBranch size={11} />
                        {proj.default_branch || 'main'}
                      </span>
                    </div>
                  </div>

                  {isActive ? (
                    <Badge tone="success">
                      <Check size={11} />
                      {t('projectManager.activeDesktop')}
                    </Badge>
                  ) : (
                    <Badge tone="default">{proj.default_isolation || 'project'}</Badge>
                  )}
                </div>

                <div
                  className="nx-desktop-card__path"
                  onClick={(e) => {
                    e.stopPropagation();
                    handleCopyPath(proj.id, proj.canonical_path);
                  }}
                  title="Click to copy canonical path"
                >
                  <code>{proj.canonical_path}</code>
                  {copiedId === proj.id ? <Check size={12} /> : <Copy size={12} />}
                </div>

                <div className="nx-desktop-card__tags">
                  <span className="nx-tag-pill">
                    <Zap size={11} />
                    {proj.resource_policy || 'BALANCED'}
                  </span>
                </div>

                {/* OS Integration Quick Launchers */}
                <div className="nx-os-quick-launchers" onClick={(e) => e.stopPropagation()}>
                  <span className="nx-os-launchers-label">OS:</span>
                  <IconButton
                    label={t('projectManager.openInFM')}
                    onClick={() => handleOpenOS(proj, 'filemanager')}
                  >
                    <Folder size={13} />
                  </IconButton>
                  <IconButton
                    label={t('projectManager.openInTerm')}
                    onClick={() => handleOpenOS(proj, 'terminal')}
                  >
                    <Terminal size={13} />
                  </IconButton>
                  <IconButton
                    label={t('projectManager.openInEditor')}
                    onClick={() => handleOpenOS(proj, 'editor')}
                  >
                    <Code2 size={13} />
                  </IconButton>
                </div>

                <div className="nx-desktop-card__footer" onClick={(e) => e.stopPropagation()}>
                  {isActive ? (
                    <Button size="sm" tone="ghost" onClick={onOpenOverview}>
                      <ExternalLink size={12} />
                      <span>{t('projectManager.openOverview')}</span>
                    </Button>
                  ) : (
                    <Button size="sm" tone="brand" onClick={() => onSelectProject(proj)}>
                      <span>{t('projectManager.switchDesktop')}</span>
                      <ArrowRight size={12} />
                    </Button>
                  )}

                  <div className="nx-desktop-card__actions">
                    <IconButton
                      label={t('projectManager.configTitle')}
                      onClick={() => handleOpenConfig(proj)}
                    >
                      <Settings size={13} />
                    </IconButton>
                    {projects.length > 1 && !isActive && (
                      <IconButton
                        label={t('projectManager.remove')}
                        onClick={() => handleDelete(proj)}
                      >
                        <Trash2 size={13} />
                      </IconButton>
                    )}
                  </div>
                </div>
              </Card>
            );
          })}
        </div>
      )}

      {/* List View */}
      {viewMode === 'list' && (
        <Card className="nx-desktop-list-card">
          <table className="nx-desktop-table">
            <thead>
              <tr>
                <th>{t('projectManager.desktopsTitle')}</th>
                <th>{t('projectManager.branch')}</th>
                <th>{t('projectManager.resourcePolicy')}</th>
                <th>OS</th>
                <th className={styles.actionsHeader}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((proj) => {
                const isActive = proj.id === selectedProject.id;
                return (
                  <tr
                    key={proj.id}
                    className={isActive ? 'nx-row--active' : ''}
                    onClick={() => onSelectProject(proj)}
                  >
                    <td>
                      <div className="nx-table-proj-cell">
                        <span className="nx-desktop-avatar nx-desktop-avatar--sm">
                          {(proj.name || 'PR').slice(0, 2).toUpperCase()}
                        </span>
                        <div>
                          <strong>{proj.name || proj.id}</strong>
                          <small>{proj.canonical_path}</small>
                        </div>
                      </div>
                    </td>
                    <td>
                      <span className="nx-desktop-branch">
                        <GitBranch size={11} /> {proj.default_branch || 'main'}
                      </span>
                    </td>
                    <td>
                      <span className="nx-tag-pill">{proj.resource_policy || 'BALANCED'}</span>
                    </td>
                    <td onClick={(e) => e.stopPropagation()}>
                      <div className="nx-table-os-actions">
                        <IconButton
                          label={t('projectManager.openInFM')}
                          onClick={() => handleOpenOS(proj, 'filemanager')}
                        >
                          <Folder size={12} />
                        </IconButton>
                        <IconButton
                          label={t('projectManager.openInTerm')}
                          onClick={() => handleOpenOS(proj, 'terminal')}
                        >
                          <Terminal size={12} />
                        </IconButton>
                        <IconButton
                          label={t('projectManager.openInEditor')}
                          onClick={() => handleOpenOS(proj, 'editor')}
                        >
                          <Code2 size={12} />
                        </IconButton>
                      </div>
                    </td>
                    <td className={styles.actionsCell} onClick={(e) => e.stopPropagation()}>
                      <div className="nx-table-actions">
                        {isActive ? (
                          <Badge tone="success">{t('projectManager.active')}</Badge>
                        ) : (
                          <Button size="sm" tone="ghost" onClick={() => onSelectProject(proj)}>
                            {t('projectManager.switch')}
                          </Button>
                        )}
                        <IconButton
                          label={t('projectManager.configTitle')}
                          onClick={() => handleOpenConfig(proj)}
                        >
                          <Settings size={13} />
                        </IconButton>
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </Card>
      )}

      {/* Empty State */}
      {filtered.length === 0 && (
        <EmptyState
          icon={<FolderGit2 size={32} />}
          title={t('projectManager.noProjects')}
          action={
            <Button tone="brand" onClick={() => setAddOpen(true)}>
              {t('projectManager.addNew')}
            </Button>
          }
        />
      )}

      {/* Add Project Modal with OS Explorer & Scanner integration */}
      <AddProjectModal
        open={addOpen}
        onClose={() => setAddOpen(false)}
        onCreated={(proj) => {
          onProjectCreated(proj);
          onSelectProject(proj);
        }}
      />

      {/* Starter Templates Modal */}
      <Dialog
        open={templateOpen}
        onClose={() => setTemplateOpen(false)}
        title={t('projectManager.templatesTitle')}
      >
        <div className="nx-templates-modal">
          <div
            className="nx-template-card"
            onClick={() => {
              setNewName('Fullstack SaaS App');
              setNewPath('/projetos/saas-app');
              setTemplateOpen(false);
              setAddOpen(true);
            }}
          >
            <div className="nx-template-icon">
              <Layers size={20} />
            </div>
            <div>
              <strong>{t('projectManager.templateFullstack')}</strong>
              <p>React + Go/Node microservices with Maestro SaaS architecture guidelines.</p>
            </div>
            <Button size="sm" tone="brand">
              {t('projectManager.useTemplate')}
            </Button>
          </div>

          <div
            className="nx-template-card"
            onClick={() => {
              setNewName('Core Microservice API');
              setNewPath('/projetos/api-service');
              setTemplateOpen(false);
              setAddOpen(true);
            }}
          >
            <div className="nx-template-icon">
              <Zap size={20} />
            </div>
            <div>
              <strong>{t('projectManager.templateApi')}</strong>
              <p>REST / gRPC microservice scaffolded with TDD loop and schema verification.</p>
            </div>
            <Button size="sm" tone="brand">
              {t('projectManager.useTemplate')}
            </Button>
          </div>

          <div
            className="nx-template-card"
            onClick={() => {
              setNewName('Developer CLI Tool');
              setNewPath('/projetos/cli-tool');
              setTemplateOpen(false);
              setAddOpen(true);
            }}
          >
            <div className="nx-template-icon">
              <Bot size={20} />
            </div>
            <div>
              <strong>{t('projectManager.templateCli')}</strong>
              <p>Cobra / Commander command-line application with automated integration tests.</p>
            </div>
            <Button size="sm" tone="brand">
              {t('projectManager.useTemplate')}
            </Button>
          </div>
        </div>
      </Dialog>

      {/* Project Configuration Modal */}
      {configProject && (
        <Dialog
          open={Boolean(configProject)}
          onClose={() => setConfigProject(null)}
          title={`${t('projectManager.configTitle')} · ${configProject.name}`}
        >
          <div className="nx-form-stack">
            <p className="nx-muted-copy">{t('projectManager.configSubtitle')}</p>

            <label>
              {t('projectManager.addName')}
              <Input value={cfgName} onChange={setCfgName} placeholder={configProject.name} />
            </label>

            {/* Git Branch Management & Selector */}
            <div className="nx-config-branch-section">
              <div className={`nx-config-branch-header ${styles.branchSelectRow}`}>
                <span className={styles.configSectionTitle}>
                  {t('projectManager.branch', 'Branch Padrão / Galho')}
                </span>
                {cfgBranchStatus && (
                  <Badge tone={cfgBranchStatus.isClean ? 'success' : 'warning'}>
                    {cfgBranchStatus.isClean
                      ? t('git.cleanTree', '● Árvore limpa')
                      : t('git.dirtyTree', `● ${cfgBranchStatus.count} alterações`)}
                  </Badge>
                )}
              </div>

              {cfgBranchAlert && (
                <div className={styles.configSectionSpacing}>
                  <InlineAlert tone={cfgBranchAlert.tone}>{cfgBranchAlert.message}</InlineAlert>
                </div>
              )}

              <div className={`nx-config-branch-controls ${styles.branchSelectRow}`}>
                {cfgBranchList.length > 0 ? (
                  <Select
                    value={cfgBranch}
                    onChange={setCfgBranch}
                    options={cfgBranchList.map((b) => ({ value: b, label: b }))}
                    className="nx-flex-1"
                  />
                ) : (
                  <Input
                    value={cfgBranch}
                    onChange={setCfgBranch}
                    placeholder="main"
                    mono
                    className="nx-flex-1"
                  />
                )}
                <Button
                  size="sm"
                  tone="brand"
                  disabled={cfgBranchSwitching || !cfgBranch.trim()}
                  onClick={() => handleDirectCheckout(cfgBranch)}
                  title={t('git.checkoutNow', 'Trocar de branch agora no repositório local')}
                >
                  {cfgBranchSwitching ? (
                    <Spinner label="" />
                  ) : (
                    <>
                      <GitBranch size={12} /> {t('git.checkoutBtn', 'Checkout')}
                    </>
                  )}
                </Button>
              </div>
            </div>

            <label>
              {t('projectManager.resourcePolicy')}
              <Segmented
                options={[
                  { value: 'BALANCED', label: 'BALANCED' },
                  { value: 'PERFORMANCE', label: 'PERFORMANCE' },
                  { value: 'COST', label: 'COST' },
                ]}
                value={cfgPolicy}
                onChange={(v) => setCfgPolicy(v as any)}
                ariaLabel={t('projectManager.resourcePolicy')}
              />
            </label>

            {cfgSuccess && <InlineAlert tone="success">{t('projectManager.saved')}</InlineAlert>}

            <div className="nx-dialog-actions">
              <Button onClick={() => setConfigProject(null)}>{t('common.closeDialog')}</Button>
              <Button tone="brand" disabled={cfgBusy} onClick={handleSaveConfig}>
                {cfgBusy ? t('projectManager.saving') : t('projectManager.saveChanges')}
              </Button>
            </div>
          </div>
        </Dialog>
      )}

      {/* Directory Browser Modal */}
      <DirectoryBrowserModal
        open={dirPickerOpen}
        onClose={() => setDirPickerOpen(false)}
        initialPath={newPath || undefined}
        onSelectPath={handleDirectorySelected}
      />

      {/* Project Scanner Modal */}
      <ProjectScanModal
        open={scanModalOpen}
        onClose={() => setScanModalOpen(false)}
        onProjectImported={(proj) => {
          onProjectCreated(proj);
          onSelectProject(proj);
          setScanModalOpen(false);
        }}
      />

      <ConfirmDialog
        open={Boolean(projectToDelete)}
        title={t('projectManager.confirmRemoveTitle', 'Excluir Projeto')}
        description={t('projectManager.confirmRemove')}
        onConfirm={executeDelete}
        onCancel={() => setProjectToDelete(null)}
      />
    </div>
  );
};
