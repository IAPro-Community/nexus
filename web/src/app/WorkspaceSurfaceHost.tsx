import React, { useMemo, useState } from 'react';
import { Gauge, Network } from 'lucide-react';
import { Button, Card, EmptyState } from '../design-system';
import { Dashboard } from '../components/Dashboard';
import { ProvidersView } from '../components/ProvidersView';
import { EventsView } from '../components/EventsView';
import { StartModal } from '../components/StartModal';
import { HandoffModal } from '../components/HandoffModal';
import { ContinueModal } from '../components/ContinueModal';
import { TerminalPane } from '../components/TerminalPane';
import { AgentTerminal } from '../nexus/AgentTerminal';
import { MaestroPage } from '../nexus/MaestroPage';
import { ResourcePicker } from '../nexus/ResourcePicker';
import { api } from '../api';
import type { Agent, EventRecord, ProfileInfo, Project, ProviderInfo, RuntimeSession, Workspace } from '../types';
import type { WorkspaceSurface } from '../workspace/model';
import { ProjectManagerSurface } from '../features/projects/ProjectManagerSurface';
import { ProjectOverviewSurface } from '../features/overview/ProjectOverviewSurface';
import { WorkSurface } from '../features/work/WorkSurface';
import { PlanBuilderSurface } from '../features/work/PlanBuilderSurface';
import { DirectSessionLauncher, type DirectSessionRequest } from '../features/work/DirectSessionLauncher';
import { AgentsSurface } from '../features/agents/AgentsSurface';
import { AgentConfigurationSurface } from '../features/agents/AgentConfigurationSurface';
import { SessionsSurface } from '../features/sessions/SessionsSurface';
import { SettingsSurface } from '../features/settings/SettingsSurface';
import { useTranslation } from 'react-i18next';
import { ProjectShellSurface } from '../features/shell/ProjectShellSurface';
import { FlowRunSurface } from '../features/work/FlowRunSurface';
import { formatResourcePolicy } from '../i18n';
import { agentConfigSurface, agentTerminalSurface, projectSurface } from './surfaces';

export const WorkspaceSurfaceHost: React.FC<{
  surface: WorkspaceSurface;
  project: Project;
  projects: Project[];
  agents: Agent[];
  workspaces: Workspace[];
  runtimes: RuntimeSession[];
  providers: ProviderInfo[];
  profiles: ProfileInfo[];
  events: EventRecord[];
  refreshAgents: () => Promise<void>;
  refreshGlobal: () => Promise<void>;
  onSelectProject: (project: Project) => void;
  onProjectCreated: (project: Project) => void;
  onProjectUpdated: (project: Project) => void;
  openSurface: (surface: WorkspaceSurface) => void;
  onTour: () => void;
}> = ({
  surface,
  project,
  projects,
  agents,
  workspaces,
  runtimes,
  providers,
  profiles,
  events,
  refreshAgents,
  refreshGlobal,
  onSelectProject,
  onProjectCreated,
  onProjectUpdated,
  openSurface,
  onTour,
}) => {
  const { t } = useTranslation();
  const agent = useMemo(
    () => agents.find((item) => item.id === surface.data?.agentId),
    [agents, surface.data?.agentId]
  );
  const [showStart, setShowStart] = useState(false);
  const [handoff, setHandoff] = useState<RuntimeSession | null>(null);
  const [cont, setCont] = useState<RuntimeSession | null>(null);
  const [directSession, setDirectSession] = useState<DirectSessionRequest | null>(null);

  const open = (kind: string) =>
    openSurface(projectSurface(project.id, kind as Parameters<typeof projectSurface>[1]));
  const terminal = (target: Agent, initialPrompt = '') => openSurface(agentTerminalSurface(target.id, target.name, initialPrompt));
  const config = (target: Agent) => openSurface(agentConfigSurface(target.id, target.name));

  if (surface.type === 'projects') {
    return (
      <ProjectManagerSurface
        projects={projects}
        selectedProject={project}
        agents={agents}
        onSelectProject={onSelectProject}
        onProjectCreated={onProjectCreated}
        onProjectUpdated={onProjectUpdated}
        onOpenOverview={() => open('overview')}
      />
    );
  }

  if (surface.type === 'overview')
    return (
      <ProjectOverviewSurface
        project={project}
        agents={agents}
        onOpenAgent={terminal}
        onOpenWork={() => open('work')}
        onOpenPlan={() => open('missions')}
      />
    );

  if (surface.type === 'work')
    return (
      <>
        <WorkSurface
          project={project}
          agents={agents}
          onDirect={terminal}
          onStartSession={(mode, prompt) => setDirectSession({ mode, prompt })}
          onPlan={() => open('missions')}
          onMaestro={() => open('maestro')}
        />
        <DirectSessionLauncher
          open={!!directSession}
          project={project}
          request={directSession}
          onClose={() => setDirectSession(null)}
          refreshAgents={refreshAgents}
          onStarted={(created, prompt) => terminal(created, prompt)}
        />
      </>
    );

  if (surface.type === 'agents')
    return (
      <AgentsSurface
        project={project}
        agents={agents}
        refresh={refreshAgents}
        onTerminal={terminal}
        onConfigure={config}
      />
    );

  if (surface.type === 'agent-config')
    return <AgentConfigurationSurface agent={agent} onApplied={refreshAgents} />;

  if (surface.type === 'terminal')
    return surface.data?.agentId ? (
      <div className="nx-agent-terminal-surface">
        <AgentTerminal agentId={surface.data.agentId} initialPrompt={surface.data.initialPrompt} />
      </div>
    ) : (
      <EmptyState title={t('surfaces.terminalUnavailable')} />
    );

  if (surface.type === 'project-shell') return <ProjectShellSurface project={project} />;
  if (surface.type === 'flow-run') return <FlowRunSurface runId={surface.data?.runId} onOpenAgent={(agentId) => openSurface(agentTerminalSurface(agentId, agentId))} />;

  if (surface.type === 'resources')
    return (
      <div className="nx-surface-scroll">
        <div className="nx-page-header">
          <div>
            <span className="nx-eyebrow">{t('surfaces.resourcesEyebrow')}</span>
            <h1>{t('surfaces.resourcesTitle')}</h1>
            <p>{t('surfaces.resourcesIntro')}</p>
          </div>
        </div>
        <div className="nx-resource-layout">
          <Card className="nx-resource-card">
            <ResourcePicker />
          </Card>
          <Card className="nx-resource-card">
            <div className="nx-feature-heading">
              <Gauge size={18} />
              <div>
                <strong>{t('surfaces.allocation')}</strong>
                <small>{formatResourcePolicy(project.resource_policy)}</small>
              </div>
            </div>
            <p className="nx-muted-copy">{t('surfaces.allocationBody')}</p>
          </Card>
        </div>
      </div>
    );

  if (surface.type === 'maestro')
    return (
      <div className="nx-surface-scroll">
        <div className="nx-page-header">
          <div>
            <span className="nx-eyebrow">{t('surfaces.maestroEyebrow')}</span>
            <h1>Maestro</h1>
            <p>{t('surfaces.maestroIntro')}</p>
          </div>
        </div>
        <MaestroPage projectId={project.id} />
      </div>
    );

  if (surface.type === 'missions')
    return (
      <div className="nx-surface-scroll">
        <div className="nx-page-header">
          <div>
            <span className="nx-eyebrow">{t('surfaces.missionsEyebrow')}</span>
            <h1>{t('surfaces.missionsTitle')}</h1>
            <p>{t('surfaces.missionsIntro')}</p>
          </div>
        </div>
        <PlanBuilderSurface project={project} agents={agents} onOpenAgent={terminal} />
      </div>
    );

  if (surface.type === 'sessions') return <SessionsSurface agents={agents} />;

  if (surface.type === 'settings') return <SettingsSurface onTour={onTour} />;

  if (surface.type === 'legacy-providers')
    return (
      <div className="nx-legacy-surface">
        <ProvidersView providers={providers} />
      </div>
    );

  if (surface.type === 'legacy-events')
    return (
      <div className="nx-legacy-surface">
        <EventsView events={events} />
      </div>
    );

  if (surface.type === 'legacy-terminal') {
    const runtime = runtimes.find((item) => item.runtime_id === surface.data?.runtimeId);
    if (!runtime) return <EmptyState title={t('surfaces.runtimeUnavailable')} />;
    return (
      <TerminalPane
        runtimeId={runtime.runtime_id}
        title={runtime.title}
        provider={runtime.provider_id || runtime.provider || 'AI'}
        profile={runtime.profile_id || runtime.profile || 'default'}
        onUpdateTitle={async (id, title) => {
          await api.updateRuntimeTitle(id, title);
          await refreshGlobal();
        }}
      />
    );
  }

  if (surface.type === 'legacy-runtimes')
    return (
      <div className="nx-legacy-surface">
        <Dashboard
          runtimes={runtimes}
          providers={providers}
          workspaces={workspaces}
          activeWorkspace={project.canonical_path}
          onOpenTerminal={(runtimeId) => {
            const runtime = runtimes.find((item) => item.runtime_id === runtimeId);
            openSurface({
              id: `runtime:${runtimeId}:terminal`,
              type: 'legacy-terminal',
              title: runtime?.title || runtimeId,
              data: { runtimeId },
            });
          }}
          onOpenStartModal={() => setShowStart(true)}
          onOpenHandoffModal={setHandoff}
          onOpenContinueModal={setCont}
          onStopRuntime={async (id) => {
            await api.stopRuntime(id);
            await refreshGlobal();
          }}
          onDeleteRuntime={async (id) => {
            await api.deleteRuntime(id);
            await refreshGlobal();
          }}
          onCleanInactive={async () => {
            await api.cleanRuntimes();
            await refreshGlobal();
          }}
        />
        {showStart && (
          <StartModal
            providers={providers}
            profiles={profiles}
            workspace={project.canonical_path}
            workspaces={workspaces}
            onClose={() => setShowStart(false)}
            onSuccess={async () => {
              setShowStart(false);
              await refreshGlobal();
            }}
          />
        )}
        {handoff && (
          <HandoffModal
            runtime={handoff}
            profiles={profiles}
            onClose={() => setHandoff(null)}
            onSuccess={async () => {
              setHandoff(null);
              await refreshGlobal();
            }}
          />
        )}
        {cont && (
          <ContinueModal
            runtime={cont}
            providers={providers}
            profiles={profiles}
            onClose={() => setCont(null)}
            onSuccess={async () => {
              setCont(null);
              await refreshGlobal();
            }}
          />
        )}
      </div>
    );

  return (
    <div className="nx-surface-center">
      <EmptyState
        icon={<Network size={22} />}
        title={t('surfaces.unavailable')}
        hint={t('surfaces.unknown', { type: surface.type })}
        action={<Button onClick={() => open('overview')}>{t('surfaces.openOverview')}</Button>}
      />
    </div>
  );
};
