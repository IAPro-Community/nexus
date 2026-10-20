import React, { Suspense, useEffect, useMemo, useState } from 'react';
import { Network } from 'lucide-react';
import { Button, EmptyState } from '../design-system';
import { recoverOrStartAgent } from '../nexus/agentRecover';
import { isLiveRuntimeState, runtimeIdFromRecoverResult } from '../nexus/agentTerminalModel';
import { nexus } from '../nexus/api';
import { api } from '../api';
import type {
  Agent,
  EventRecord,
  ProfileInfo,
  Project,
  ProviderInfo,
  RuntimeSession,
  MissionRun,
  Workspace,
} from '../types';
import type { WorkspaceSurface } from '../workspace/model';
import { useTranslation } from 'react-i18next';
import {
  agentConfigSurface,
  agentTerminalSurface,
  flowRunSurface,
  projectSurface,
  projectShellSurface,
} from './surfaces';
import { surfaceViewId } from '../workspace/model';
import { useWorkspacePresentation } from '../workspace/WorkspacePresentationProvider';
import { SurfaceLoadingFallback } from './SurfaceLoadingFallback';
import { mapDurableActivity } from './activityModel';
import { COMPOSER_FLOW_ENABLED, isExperimentalComposerFlowSurface } from './composerFlowFlag';

// Lazy-loaded Surfaces to isolate heavy dependencies (xterm.js, DAG canvas, settings)
const ProjectManagerSurface = React.lazy(() =>
  import('../features/projects/ProjectManagerSurface').then((m) => ({
    default: m.ProjectManagerSurface,
  })),
);
const ProjectOverviewSurface = React.lazy(() =>
  import('../features/overview/ProjectOverviewSurface').then((m) => ({
    default: m.ProjectOverviewSurface,
  })),
);
const WorkSurface = React.lazy(() =>
  import('../features/work/WorkSurface').then((m) => ({ default: m.WorkSurface })),
);
const FlowRunSurface = React.lazy(() =>
  import('../features/work/FlowRunSurface').then((m) => ({ default: m.FlowRunSurface })),
);
const FlowRunsHistorySurface = React.lazy(() =>
  import('../features/work/FlowRunsHistorySurface').then((m) => ({
    default: m.FlowRunsHistorySurface,
  })),
);
const AgentsSurface = React.lazy(() =>
  import('../features/agents/AgentsSurface').then((m) => ({ default: m.AgentsSurface })),
);
const AgentConfigurationSurface = React.lazy(() =>
  import('../features/agents/AgentConfigurationSurface').then((m) => ({
    default: m.AgentConfigurationSurface,
  })),
);
const SessionsSurface = React.lazy(() =>
  import('../features/sessions/SessionsSurface').then((m) => ({ default: m.SessionsSurface })),
);
const SettingsSurface = React.lazy(() =>
  import('../features/settings/SettingsSurface').then((m) => ({ default: m.SettingsSurface })),
);
const MaestroSurface = React.lazy(() =>
  import('../features/maestro/MaestroSurface').then((m) => ({ default: m.MaestroSurface })),
);
const ProjectShellSurface = React.lazy(() =>
  import('../features/shell/ProjectShellSurface').then((m) => ({ default: m.ProjectShellSurface })),
);
const AgentTerminal = React.lazy(() =>
  import('../nexus/AgentTerminal').then((m) => ({ default: m.AgentTerminal })),
);
const UsageSurface = React.lazy(() =>
  import('../features/usage/UsageSurface').then((m) => ({ default: m.UsageSurface })),
);
const TerminalPane = React.lazy(() =>
  import('../components/TerminalPane').then((m) => ({ default: m.TerminalPane })),
);
const Dashboard = React.lazy(() =>
  import('../components/Dashboard').then((m) => ({ default: m.Dashboard })),
);
const ProvidersView = React.lazy(() =>
  import('../components/ProvidersView').then((m) => ({ default: m.ProvidersView })),
);
const EventsView = React.lazy(() =>
  import('../components/EventsView').then((m) => ({ default: m.EventsView })),
);
const StartModal = React.lazy(() =>
  import('../components/StartModal').then((m) => ({ default: m.StartModal })),
);
const HandoffModal = React.lazy(() =>
  import('../components/HandoffModal').then((m) => ({ default: m.HandoffModal })),
);
const ContinueModal = React.lazy(() =>
  import('../components/ContinueModal').then((m) => ({ default: m.ContinueModal })),
);

export const WorkspaceSurfaceHost: React.FC<{
  surface: WorkspaceSurface;
  project: Project;
  projects: Project[];
  agents: Agent[];
  workspaces: Workspace[];
  runtimes: RuntimeSession[];
  flowRuns?: MissionRun[];
  providers: ProviderInfo[];
  profiles: ProfileInfo[];
  events: EventRecord[];
  refreshAgents: () => Promise<void>;
  refreshGlobal: () => Promise<{ runtimes: RuntimeSession[] }>;
  onSelectProject: (project: Project) => void;
  onProjectCreated: (project: Project) => void;
  onProjectUpdated: (project: Project) => void;
  openSurface: (surface: WorkspaceSurface) => void;
  updateSurface?: (surfaceId: string, patch: Partial<WorkspaceSurface>) => void;
  closeSurface: (surfaceId: string) => void;
  onTour: () => void;
  settingsInitialTab?: 'appearance' | 'accessibility' | 'updates' | 'remote' | 'hud';
}> = ({
  surface,
  project,
  projects,
  agents,
  workspaces,
  runtimes,
  flowRuns = [],
  providers,
  profiles,
  events,
  refreshAgents,
  refreshGlobal,
  onSelectProject,
  onProjectCreated,
  onProjectUpdated,
  openSurface,
  updateSurface,
  closeSurface,
  onTour,
  settingsInitialTab,
}) => {
  const { t } = useTranslation();
  const presentation = useWorkspacePresentation();
  const terminalChrome =
    presentation.state.mode === 'DESKTOP' ||
    presentation.state.mode === 'MOSAIC' ||
    presentation.state.zenMode
      ? 'window'
      : 'full';
  const agent = useMemo(
    () => agents.find((item) => item.id === surface.data?.agentId),
    [agents, surface.data?.agentId],
  );
  const [showStart, setShowStart] = useState(false);
  const [handoff, setHandoff] = useState<RuntimeSession | null>(null);
  const [cont, setCont] = useState<RuntimeSession | null>(null);
  const [activityEvents, setActivityEvents] = useState<EventRecord[]>([]);

  useEffect(() => {
    if (surface.type !== 'legacy-events') return;
    let active = true;
    void nexus
      .listProjectEvents(project.id)
      .then((items) => {
        if (active) setActivityEvents((items || []).map(mapDurableActivity));
      })
      .catch(() => {
        if (active) setActivityEvents([]);
      });
    return () => {
      active = false;
    };
  }, [project.id, surface.type]);

  const open = (kind: string) =>
    openSurface(projectSurface(project.id, kind as Parameters<typeof projectSurface>[1]));
  const terminal = (target: Agent, initialPrompt = '', runtimeId = '') => {
    openSurface(agentTerminalSurface(target.id, target.name, initialPrompt, runtimeId));
    openSurface(projectSurface(project.id, 'terminals'));
  };
  const config = (target: Agent) => openSurface(agentConfigSurface(target.id, target.name));

  const renderSurfaceContent = () => {
    if (!COMPOSER_FLOW_ENABLED && isExperimentalComposerFlowSurface(surface.type)) {
      return (
        <EmptyState
          title={t('surfaces.unavailable', 'Feature unavailable')}
          hint={t('surfaces.experimentalDisabled', 'Composer and Flow are under validation.')}
          action={<Button onClick={() => open('overview')}>{t('surfaces.openOverview')}</Button>}
        />
      );
    }
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

    if (surface.type === 'overview') {
      return (
        <ProjectOverviewSurface
          project={project}
          agents={agents}
          runtimes={runtimes}
          flowRuns={COMPOSER_FLOW_ENABLED ? flowRuns : []}
          onOpenAgent={(agent, runtimeId) => terminal(agent, '', runtimeId || '')}
          refreshAgents={refreshAgents}
          onNewAgent={() => window.dispatchEvent(new CustomEvent('nexus:new-agent'))}
          onConfigureAgent={config}
          onNewAISession={() => window.dispatchEvent(new CustomEvent('nexus:new-ai-session'))}
          onProjectShell={() => window.dispatchEvent(new CustomEvent('nexus:project-shell'))}
          onOpenComposer={COMPOSER_FLOW_ENABLED ? () => open('work') : undefined}
          onOpenFlow={COMPOSER_FLOW_ENABLED ? () => open('missions') : undefined}
          onOpenSettings={() => open('settings')}
        />
      );
    }

    if (surface.type === 'work') {
      return (
        <WorkSurface
          project={project}
          agents={agents}
          onDirect={terminal}
          onFlowRun={(run) =>
            openSurface(flowRunSurface(run.id, `Flow Run · ${(run.id || '').slice(-6)}`))
          }
          onOpenFlowDrafts={() => open('missions')}
        />
      );
    }

    if (surface.type === 'flow-run') {
      return surface.data?.runId ? (
        <FlowRunSurface
          runId={surface.data.runId}
          project={project}
          agents={agents}
          onOpenAgent={terminal}
          onOpenResources={() => open('resources')}
        />
      ) : (
        <EmptyState title="Flow Run unavailable" />
      );
    }

    if (surface.type === 'agents') {
      return (
        <AgentsSurface
          project={project}
          agents={agents}
          refresh={refreshAgents}
          onTerminal={terminal}
          onConfigure={config}
          onRemoved={(agentId) => closeSurface(`agent:${agentId}:terminal`)}
        />
      );
    }

    if (surface.type === 'agent-config') {
      return <AgentConfigurationSurface agent={agent} onApplied={refreshAgents} />;
    }

    if (surface.type === 'project-shell') {
      return surface.data?.runtimeId ? (
        <ProjectShellSurface
          runtimeId={surface.data.runtimeId}
          title={surface.title}
          liveTitleKey={surfaceViewId(surface)}
          onRuntimeChanged={async (nextTitle) => {
            updateSurface?.(surface.id, { title: nextTitle });
            await refreshGlobal();
          }}
          onRestart={async () => {
            const result = await nexus.startProjectShell(project.id);
            closeSurface(surface.id);
            openSurface(
              projectShellSurface(
                project.id,
                result.runtime.runtime_id,
                result.runtime.title || surface.title || 'Terminal',
              ),
            );
            await refreshGlobal();
          }}
        />
      ) : (
        <EmptyState title="Project shell unavailable" />
      );
    }

    if (surface.type === 'terminal') {
      const agentId = surface.data?.agentId;
      const preferredRuntimeId = (surface.data?.runtimeId || '').trim();
      const agentRuntimes = runtimes.filter((r) => r.agent_id === agentId);
      const preferredLive = preferredRuntimeId
        ? agentRuntimes.find(
            (r) => r.runtime_id === preferredRuntimeId && isLiveRuntimeState(r.state),
          )
        : undefined;
      const liveRuntime =
        preferredLive || agentRuntimes.find((r) => isLiveRuntimeState(r.state)) || agentRuntimes[0];
      return agentId ? (
        <div className="nx-agent-terminal-surface">
          <AgentTerminal
            agentId={agentId}
            projectId={project.id}
            runtimeId={liveRuntime?.runtime_id}
            preferredRuntimeId={preferredLive?.runtime_id}
            initialPrompt={surface.data?.initialPrompt}
            provider={liveRuntime?.provider_id || liveRuntime?.provider || 'claude'}
            profile={liveRuntime?.profile_id || liveRuntime?.profile || 'default'}
            agentName={agent?.name || surface.title}
            chrome={terminalChrome}
            liveTitleKey={surfaceViewId(surface)}
            onBoundRuntimeChange={(nextId) => {
              updateSurface?.(surface.id, {
                data: {
                  ...surface.data,
                  agentId,
                  runtimeId: nextId,
                },
              });
            }}
            onRecover={async () => {
              try {
                const result = await recoverOrStartAgent(agentId);
                const nextRuntimeId = runtimeIdFromRecoverResult(result);
                if (nextRuntimeId) {
                  updateSurface?.(surface.id, {
                    data: {
                      ...surface.data,
                      agentId,
                      runtimeId: nextRuntimeId,
                    },
                  });
                }
                await refreshAgents();
                await refreshGlobal();
                return result?.runtime;
              } catch (err) {
                throw err instanceof Error
                  ? err
                  : new Error(
                      'Não foi possível iniciar ou recuperar o runtime do agente. Verifique o provedor e a conta configurados.',
                    );
              }
            }}
            onRestartWithMode={async (newMode: 'Safe' | 'YOLO') => {
              if (!agent) return;
              const currentCfg = await nexus.getAgentConfig(agent.id);
              const configuredArgs = Array.isArray(currentCfg.config.options?.extra_args)
                ? currentCfg.config.options.extra_args
                : [];
              const extraArgs = configuredArgs.filter(
                (arg) => arg !== '--plan' && arg !== '--yolo' && arg !== '-y',
              );
              if (newMode === 'YOLO') extraArgs.push('--yolo');
              await nexus.applyAgentConfig(agent.id, {
                ...currentCfg.config,
                continuity_policy: 'new_session',
                options: {
                  ...currentCfg.config.options,
                  mode: newMode,
                  extra_args: extraArgs,
                },
              });
              await refreshAgents();
              const globalData = await refreshGlobal();
              const latestRuntime = (globalData?.runtimes || []).find(
                (item: RuntimeSession) => item.agent_id === agent.id && item.state === 'RUNNING',
              );
              if (latestRuntime?.runtime_id) {
                updateSurface?.(surface.id, {
                  data: {
                    ...surface.data,
                    agentId: agent.id,
                    runtimeId: latestRuntime.runtime_id,
                  },
                });
              }
              return latestRuntime;
            }}
            onClose={async (stopRuntime) => {
              if (stopRuntime) {
                const currentRuntimeId =
                  liveRuntime?.runtime_id ||
                  runtimes.find(
                    (item) => item.agent_id === agentId && isLiveRuntimeState(item.state),
                  )?.runtime_id;
                if (currentRuntimeId) await api.stopRuntime(currentRuntimeId);
                else await nexus.stopAgent(agentId);
                await refreshGlobal();
              }
              closeSurface(surface.id);
            }}
            onDelete={async () => {
              try {
                await nexus.stopAgent(agentId);
              } catch {
                /* may already be dead / unreachable */
              }
              await nexus.deleteAgent(agentId);
              await refreshAgents();
              await refreshGlobal();
              closeSurface(surface.id);
            }}
          />
        </div>
      ) : (
        <EmptyState title={t('surfaces.terminalUnavailable')} />
      );
    }

    if (surface.type === 'resources') {
      return <UsageSurface />;
    }

    if (surface.type === 'maestro') {
      return <MaestroSurface />;
    }

    if (surface.type === 'missions') {
      return (
        <FlowRunsHistorySurface
          project={project}
          agents={agents}
          onOpenRun={(run) => openSurface(flowRunSurface(run.id, `Flow Run · ${run.id.slice(-6)}`))}
          onOpenComposer={() => open('work')}
        />
      );
    }

    if (surface.type === 'sessions') return <SessionsSurface agents={agents} />;

    if (surface.type === 'settings')
      return <SettingsSurface onTour={onTour} initialTab={settingsInitialTab} />;

    if (surface.type === 'legacy-providers') {
      return (
        <div className="nx-legacy-surface">
          <ProvidersView providers={providers} />
        </div>
      );
    }

    if (surface.type === 'legacy-events') {
      return (
        <div className="nx-legacy-surface">
          <EventsView events={activityEvents.length > 0 ? activityEvents : events} />
        </div>
      );
    }

    if (surface.type === 'legacy-terminal') {
      const runtime = runtimes.find((item) => item.runtime_id === surface.data?.runtimeId);
      if (!runtime) return <EmptyState title={t('surfaces.runtimeUnavailable')} />;
      return (
        <TerminalPane
          runtimeId={runtime.runtime_id}
          title={runtime.title}
          provider={runtime.provider_id || runtime.provider || 'AI'}
          profile={runtime.profile_id || runtime.profile || 'default'}
          hideHeader={
            presentation.state.zenMode ||
            presentation.state.mode === 'DESKTOP' ||
            presentation.state.mode === 'MOSAIC'
          }
          liveTitleKey={surfaceViewId(surface)}
          onUpdateTitle={async (id, title) => {
            await api.updateRuntimeTitle(id, title);
            updateSurface?.(surface.id, { title });
            await refreshGlobal();
          }}
        />
      );
    }

    if (surface.type === 'legacy-runtimes') {
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
    }

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

  return (
    <Suspense fallback={<SurfaceLoadingFallback label={surface.title || surface.type} />}>
      {renderSurfaceContent()}
    </Suspense>
  );
};
