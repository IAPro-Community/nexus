import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { LoaderCircle } from 'lucide-react';
import { api, initSession, rotateSession, tryLoopbackReauth, type BrowserSession } from '../api';
import { setNexusCSRF, nexus } from '../nexus/api';
import { ThemeProvider } from '../design-system';
import { NexusSplashScreen } from './NexusSplashScreen';
import unauthorizedStyles from './NexusUnauthorized.module.scss';
import { WorkspaceProvider, useWorkspace } from '../workspace/WorkspaceProvider';
import {
  WorkspacePresentationProvider,
  useWorkspacePresentation,
} from '../workspace/WorkspacePresentationProvider';
import { PtyLiveChromeProvider } from '../workspace/PtyLiveChromeContext';
import { WorkspaceRenderer } from '../workspace/WorkspaceRenderer';
import {
  createWorkspace,
  isSurfaceMatch,
  listStacks,
  listSurfaces,
  surfaceViewId,
  type WorkspaceSurface,
} from '../workspace/model';
import { serializeWorkspace } from '../workspace/state';
import { ProjectRail } from '../features/projects/ProjectRail';
import { ProjectHub } from '../features/projects/ProjectHub';
import { NexusHud } from '../features/hud/NexusHud';
import type { NexusCommand } from './commands/registry';
import type { DirectSessionRequest } from '../features/work/DirectSessionLauncher';
import { NexusShell } from './NexusShell';
import shellStyles from './NexusShell.module.scss';
import { WorkspaceSurfaceHost } from './WorkspaceSurfaceHost';

const CommandPalette = React.lazy(() =>
  import('./commands/CommandPalette').then((m) => ({ default: m.CommandPalette })),
);
const ProductTour = React.lazy(() =>
  import('./tour/ProductTour').then((m) => ({ default: m.ProductTour })),
);
const WelcomeModal = React.lazy(() =>
  import('./modals/WelcomeModal').then((m) => ({ default: m.WelcomeModal })),
);
const MaestroControlModal = React.lazy(() =>
  import('./modals/MaestroControlModal').then((m) => ({ default: m.MaestroControlModal })),
);
const NewAgentModal = React.lazy(() =>
  import('../features/agents/NewAgentModal').then((m) => ({ default: m.NewAgentModal })),
);
const DirectSessionLauncher = React.lazy(() =>
  import('../features/work/DirectSessionLauncher').then((m) => ({
    default: m.DirectSessionLauncher,
  })),
);
const TerminalActionDialog = React.lazy(() =>
  import('../nexus/TerminalActionDialog').then((m) => ({ default: m.TerminalActionDialog })),
);
import { useLocation, useNavigate } from 'react-router-dom';
import {
  agentConfigSurface,
  agentTerminalSurface,
  flowRunSurface,
  isPtySurface,
  projectShellSurface,
  projectSurface,
} from './surfaces';
import {
  buildProjectRoute,
  parseRouteLocation,
  routeToWorkspaceSurface,
  globalSurfaceToProjectSurface,
  validProjectSurfaces,
  type GlobalSurfaceKind,
  type ParsedRoute,
} from './routes';
import { attentionFingerprintOf, buildDocumentTitle, isHonestNeedsUser } from './documentTitle';
import { planFocusAttention, type RadarRuntimeItem } from './attentionRadarModel';
import { resolveProjectSelection, resolveStaleProjectRoute } from './projectSelection';
import { useNexusData } from './useNexusData';
import type { Agent, ExecutionRole, MissionRun, Project } from '../types';
import { currentRuntimeForAgent } from './runtimeAgentMapping';
import { useTranslation } from 'react-i18next';
import { shouldMarkUnread, surfaceTitleFromAgent } from '../workspace/surfaceAttention';
import { KeyboardShortcutRegistry } from '../keyboard/KeyboardShortcutRegistry';
import { pushNotifications } from '../notifications/PushNotificationManager';
import { loadNotificationPrefs, playAttentionSound } from '../notifications/notificationPrefs';
import { formatAttentionPushBody } from '../notifications/attentionPushCopy';
import { isPtyAttentionFocused } from '../notifications/attentionDelivery';
import { interpretNexusVoiceCommand } from '../features/voice/voiceCommand';

const selectedProjectKey = 'iapro:nexus:selected-project:v1';
const tourKey = 'iapro:nexus:tour-complete:v1';

export const NexusWorkspaceApp: React.FC<{
  popoutSurface?: WorkspaceSurface;
  initialGlobalSurface?: GlobalSurfaceKind;
}> = ({ popoutSurface, initialGlobalSurface }) => {
  const { t } = useTranslation();
  const [sessionReady, setSessionReady] = useState(false);
  const [authenticated, setAuthenticated] = useState(true);

  useEffect(() => {
    let rotationTimer: number | undefined;
    const scheduleRotation = (session: BrowserSession) => {
      if (!session.expires_at) return;
      const expiresAt = Date.parse(session.expires_at);
      if (!Number.isFinite(expiresAt)) return;
      const delay = Math.max(30_000, expiresAt - Date.now() - 30 * 60 * 1000);
      rotationTimer = window.setTimeout(async () => {
        const rotated = await rotateSession();
        setAuthenticated(rotated.authenticated);
        if (rotated.authenticated) scheduleRotation(rotated);
      }, delay);
    };
    const onExpired = () => setAuthenticated(false);
    window.addEventListener('nexus:session-expired', onExpired);
    initSession()
      .then((session) => {
        setAuthenticated(session.authenticated);
        if (session.csrf_token) setNexusCSRF(session.csrf_token);
        if (session.authenticated) scheduleRotation(session);
      })
      .finally(() => setSessionReady(true));
    return () => {
      window.removeEventListener('nexus:session-expired', onExpired);
      if (rotationTimer) window.clearTimeout(rotationTimer);
    };
  }, []);

  if (!sessionReady) {
    return <NexusSplashScreen stage="starting" />;
  }

  if (!authenticated) {
    return (
      <ThemeProvider>
        <div className={`nx-app-unauthorized ${unauthorizedStyles.container}`}>
          <div className={unauthorizedStyles.card}>
            <span className="nx-brand-mark nx-brand-mark--hero">
              <img src="./nexus-icon.png" alt="Nexus" className="nx-brand-mark__img" />
            </span>
            <h2 className={unauthorizedStyles.title}>{t('auth.sessionExpired')}</h2>
            <p className={unauthorizedStyles.hint}>
              {window.location.protocol === 'http:' &&
              !['localhost', '127.0.0.1', '[::1]', 'nexus.dev', 'wails.localhost'].includes(window.location.hostname.toLowerCase())
                ? t('auth.networkHttpsRequired', {
                    defaultValue: 'Acesso LAN por HTTP não permite autenticação segura. Configure HTTPS confiável ou utilize o túnel HTTPS.',
                  })
                : t('auth.remoteInvitationNotice', {
                    defaultValue: 'Sessão inválida, expirada ou convite já utilizado. Para um novo dispositivo, gere um convite de acesso no Nexus principal.',
                  })}
            </p>
            <button
              type="button"
              className="nx-button"
              data-tone="brand"
              onClick={() => {
                void (async () => {
                  const reauth = await tryLoopbackReauth();
                  if (reauth?.authenticated) {
                    if (reauth.csrf_token) setNexusCSRF(reauth.csrf_token);
                    setAuthenticated(true);
                    return;
                  }
                  window.location.reload();
                })();
              }}
            >
              {t('auth.reloadApp')}
            </button>
          </div>
        </div>
      </ThemeProvider>
    );
  }

  return (
    <ThemeProvider>
      <NexusWorkspaceSession
        popoutSurface={popoutSurface}
        initialGlobalSurface={initialGlobalSurface}
      />
    </ThemeProvider>
  );
};

const NexusWorkspaceSession: React.FC<{
  popoutSurface?: WorkspaceSurface;
  initialGlobalSurface?: GlobalSurfaceKind;
}> = ({ popoutSurface: explicitPopout, initialGlobalSurface }) => {
  const location = useLocation();
  const navigate = useNavigate();
  const data = useNexusData();
  const { refreshAgents } = data;

  const parsedRoute = useMemo(
    () => parseRouteLocation(location.pathname, location.search),
    [location.pathname, location.search],
  );

  const popoutSurface = useMemo(() => {
    if (explicitPopout) return explicitPopout;
    if (parsedRoute.kind === 'popout') {
      return projectSurface(parsedRoute.projectId, parsedRoute.surface as any);
    }
    return undefined;
  }, [explicitPopout, parsedRoute]);

  const routeProjectId =
    parsedRoute.kind === 'project' || parsedRoute.kind === 'popout'
      ? parsedRoute.projectId
      : undefined;

  const [selectedId, setSelectedId] = useState(
    () => routeProjectId || window.localStorage.getItem(selectedProjectKey) || '',
  );

  useEffect(() => {
    const decision = resolveStaleProjectRoute(data.projects, routeProjectId);
    if (decision.kind === 'redirect') {
      const surface = parsedRoute.kind === 'project' ? parsedRoute.surface : 'overview';
      navigate(buildProjectRoute(decision.projectId, surface), { replace: true });
      return;
    }
    if (decision.kind === 'keep' && routeProjectId && routeProjectId !== selectedId) {
      setSelectedId(routeProjectId);
    }
  }, [data.projects, navigate, parsedRoute, routeProjectId, selectedId]);

  const resolvedSelection = resolveProjectSelection(data.projects, selectedId);
  // Hold the last known project across empty polls so WorkspaceProvider does not remount.
  const stickyProjectRef = useRef<Project | null>(null);
  if (resolvedSelection) {
    stickyProjectRef.current = resolvedSelection;
  } else if (!selectedId) {
    stickyProjectRef.current = null;
  } else if (stickyProjectRef.current && stickyProjectRef.current.id !== selectedId) {
    stickyProjectRef.current = null;
  }
  const selected =
    resolvedSelection ??
    (selectedId && stickyProjectRef.current?.id === selectedId ? stickyProjectRef.current : null);
  const selectedProjectId = selected?.id;
  const [layout, setLayout] = useState<string | undefined>();
  const [layoutRevision, setLayoutRevision] = useState<number | undefined>();
  const [layoutReady, setLayoutReady] = useState(false);

  useEffect(() => {
    if (!selectedProjectId) return;
    let cancelled = false;
    setLayoutReady(false);
    setSelectedId(selectedProjectId);
    window.localStorage.setItem(selectedProjectKey, selectedProjectId);
    void refreshAgents(selectedProjectId);
    nexus
      .getProject(selectedProjectId)
      .then((detail) => {
        if (cancelled) return;
        setLayout(detail.layout || undefined);
        setLayoutRevision(detail.revision);
      })
      .catch(() => {
        if (cancelled) return;
        setLayout(undefined);
        setLayoutRevision(undefined);
      })
      .finally(() => {
        if (!cancelled) setLayoutReady(true);
      });
    return () => {
      cancelled = true;
    };
  }, [refreshAgents, selectedProjectId]);

  useEffect(() => {
    if (data.loading) return;
    if (parsedRoute.kind === 'root') {
      if (selected) {
        navigate(buildProjectRoute(selected.id, 'overview'), { replace: true });
      } else if (data.projects.length === 0) {
        navigate('/projects', { replace: true });
      }
    }
  }, [data.loading, parsedRoute.kind, selected, data.projects.length, navigate]);

  if (data.loading) {
    return <NexusSplashScreen stage="loading" />;
  }

  if (initialGlobalSurface === 'projects' || (!selected && data.projects.length === 0)) {
    return (
      <ProjectHub
        onCreated={(project) => {
          data.setProjects((current) => [project, ...current]);
          setSelectedId(project.id);
          navigate(buildProjectRoute(project.id, 'overview'));
        }}
      />
    );
  }

  if (!selected) {
    return (
      <ProjectHub
        onCreated={(project) => {
          data.setProjects((current) => [project, ...current]);
          setSelectedId(project.id);
          navigate(buildProjectRoute(project.id, 'overview'));
        }}
      />
    );
  }

  const initial = popoutSurface ? serializeWorkspace(createWorkspace(popoutSurface)) : layout;

  return (
    <WorkspaceProvider
      key={`${selected.id}:${popoutSurface?.id || 'main'}`}
      projectId={selected.id}
      initialLayout={initial}
      initialRevision={layoutRevision}
      saveLayout={
        !layoutReady || popoutSurface
          ? undefined
          : (next, revision) => nexus.saveLayout(selected.id, next, revision)
      }
    >
      <WorkspacePresentationProvider projectId={selected.id}>
        <PtyLiveChromeProvider>
          <WorkspaceCoordinator
            project={selected}
            setProject={(project) => {
              setSelectedId(project.id);
              navigate(buildProjectRoute(project.id, 'overview'));
            }}
            data={data}
            popout={Boolean(popoutSurface)}
            parsedRoute={parsedRoute}
            layoutReady={layoutReady}
          />
        </PtyLiveChromeProvider>
      </WorkspacePresentationProvider>
    </WorkspaceProvider>
  );
};

const WorkspaceCoordinator: React.FC<{
  project: Project;
  setProject: (project: Project) => void;
  data: ReturnType<typeof useNexusData>;
  popout: boolean;
  parsedRoute: ParsedRoute;
  layoutReady: boolean;
}> = ({ project, setProject, data, popout, parsedRoute, layoutReady }) => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const workspace = useWorkspace();
  const presentation = useWorkspacePresentation();
  const [railOpen, setRailOpen] = useState(false);
  const [palette, setPalette] = useState(false);
  const [settingsInitialTab, setSettingsInitialTab] = useState<
    'appearance' | 'accessibility' | 'updates' | 'remote' | 'hud'
  >('appearance');
  const [welcomeOpen, setWelcomeOpen] = useState(false);
  const [maestroControlOpen, setMaestroControlOpen] = useState(false);
  const [newAgentOpen, setNewAgentOpen] = useState(false);
  const [directSession, setDirectSession] = useState<DirectSessionRequest | null>(null);
  const [tour, setTour] = useState(false);
  const [shellError, setShellError] = useState('');
  const [flowRuns, setFlowRuns] = useState<MissionRun[]>([]);
  const [allFlowRuns, setAllFlowRuns] = useState<MissionRun[]>([]);
  const [executionRoles, setExecutionRoles] = useState<ExecutionRole[]>([]);
  const [closeTarget, setCloseTarget] = useState<WorkspaceSurface | null>(null);
  const shellInFlight = useRef(false);

  useEffect(() => {
    let mounted = true;
    void nexus
      .listExecutionRoles()
      .then((roles) => {
        if (mounted) setExecutionRoles(Array.isArray(roles) ? roles : []);
      })
      .catch(() => {
        if (mounted) setExecutionRoles([]);
      });
    return () => {
      mounted = false;
    };
  }, []);

  useEffect(() => {
    let mounted = true;
    void nexus
      .getRuns()
      .then((runs) => {
        if (!mounted) return;
        setAllFlowRuns(runs);
        const active = runs.filter(
          (run) =>
            !['COMPLETED_VERIFIED', 'CANCELED_BY_USER', 'FAILED_BUDGET_EXCEEDED'].includes(
              run.state,
            ),
        );
        setFlowRuns(active);
      })
      .catch(() => mounted && setFlowRuns([]));
    return () => {
      mounted = false;
    };
  }, [project.id]);

  const open = useCallback(
    (surface: WorkspaceSurface, updateUrl = true) => {
      workspace.open(surface);
      if (updateUrl && !popout) {
        if (surface.type === 'flow-run' && surface.data?.runId) {
          navigate(buildProjectRoute(project.id, 'missions', surface.data.runId));
        } else if (surface.type === 'project-shell' && surface.data?.runtimeId) {
          navigate(buildProjectRoute(project.id, 'terminals', surface.data.runtimeId));
        } else if (surface.type === 'terminal' && surface.data?.agentId) {
          navigate(buildProjectRoute(project.id, 'agents', surface.data.agentId));
        } else if (surface.type === 'agent-config' && surface.data?.agentId) {
          navigate(buildProjectRoute(project.id, 'agents', surface.data.agentId, 'config'));
        } else if (validProjectSurfaces.has(surface.type as any)) {
          navigate(buildProjectRoute(project.id, surface.type as any));
        }
      }
    },
    [workspace, popout, navigate, project.id],
  );

  const requestCloseSurface = (surface: WorkspaceSurface) => setCloseTarget(surface);
  const closeConfirmed = async (stopRuntime: boolean) => {
    if (!closeTarget) return;
    const target = closeTarget;
    const runtimeId = target.data?.runtimeId;
    const agentId = target.data?.agentId;
    setCloseTarget(null);
    workspace.close(target.id);
    if (!stopRuntime) return;
    void (async () => {
      try {
        if (target.type === 'project-shell' && runtimeId) {
          await api.stopRuntime(runtimeId);
        } else if (target.type === 'terminal' && agentId) {
          await nexus.stopAgent(agentId);
        }
      } finally {
        await data.refreshGlobal().catch(() => undefined);
      }
    })();
  };
  const openKind = useCallback(
    (kind: string) => open(projectSurface(project.id, kind as any)),
    [open, project.id],
  );
  const openTerminals = useCallback(
    () => open(projectSurface(project.id, 'terminals')),
    [open, project.id],
  );
  const openNewAISession = useCallback(() => setDirectSession({ mode: 'direct', prompt: '' }), []);
  const terminal = useCallback(
    (agent: Agent, initialPrompt = '', runtimeId = '') => {
      const live = runtimeId || currentRuntimeForAgent(data.runtimes, agent.id)?.runtime_id || '';
      const surface = agentTerminalSurface(agent.id, agent.name, initialPrompt, live);
      open(surface);
      if (live || initialPrompt) {
        const existing = listSurfaces(workspace.model.root).find((entry) =>
          isSurfaceMatch(entry, surface.id),
        );
        workspace.updateSurface(surface.id, {
          data: {
            ...(existing?.data || {}),
            agentId: agent.id,
            ...(live ? { runtimeId: live } : {}),
            ...(initialPrompt ? { initialPrompt } : {}),
          },
        });
      }
      openTerminals();
    },
    [open, openTerminals, data.runtimes, workspace],
  );
  const config = useCallback(
    (agent: Agent) => open(agentConfigSurface(agent.id, agent.name)),
    [open],
  );

  const lastSyncedRouteRef = useRef<string>('');
  useEffect(() => {
    if (!layoutReady) return;
    const timer = window.setTimeout(() => {
      const routeKey = `${parsedRoute.kind}:${parsedRoute.kind === 'project' ? parsedRoute.surface : ''}:${location.pathname}:${location.search}`;
      const routeChanged = lastSyncedRouteRef.current !== routeKey;

      if (parsedRoute.kind === 'project' && parsedRoute.projectId === project.id) {
        const targetSurface = routeToWorkspaceSurface(parsedRoute, { agents: data.agents });
        if (targetSurface && routeChanged) {
          lastSyncedRouteRef.current = routeKey;
          workspace.open(targetSurface);
        }
      } else if (parsedRoute.kind === 'global') {
        const projectSurfaceKind = globalSurfaceToProjectSurface(parsedRoute.surface);
        if (projectSurfaceKind && routeChanged) {
          lastSyncedRouteRef.current = routeKey;
          workspace.open(projectSurface(project.id, projectSurfaceKind));
        }
        if (parsedRoute.surface === 'welcome') setWelcomeOpen(true);
      }
    }, 0);
    return () => window.clearTimeout(timer);
  }, [
    parsedRoute,
    project.id,
    data.agents,
    location.pathname,
    location.search,
    workspace,
    layoutReady,
  ]);
  const shell = useCallback(async () => {
    if (shellInFlight.current) return;
    shellInFlight.current = true;
    setShellError('');
    try {
      const result = await nexus.startProjectShell(project.id);
      workspace.open(
        projectShellSurface(
          project.id,
          result.runtime.runtime_id,
          result.runtime.title || 'Terminal',
        ),
      );
      workspace.open(projectSurface(project.id, 'terminals'));
      await data.refreshGlobal().catch(() => undefined);
    } catch (error) {
      setShellError(error instanceof Error ? error.message : String(error));
    } finally {
      shellInFlight.current = false;
    }
  }, [project.id, workspace, data]);

  const activeAgentRuntimeId = useMemo(() => {
    const activeViewId = presentation.state.activePtyViewId;
    if (!activeViewId) return '';
    const surface = listSurfaces(workspace.model.root).find(
      (entry) =>
        entry.type === 'terminal' && isPtySurface(entry) && surfaceViewId(entry) === activeViewId,
    );
    return String(surface?.data?.runtimeId || '');
  }, [presentation.state.activePtyViewId, workspace.model.root]);

  // Ensure pinned product tabs exist without stealing Overview focus.
  useEffect(() => {
    workspace.ensure(projectSurface(project.id, 'terminals'));
    workspace.ensure(projectSurface(project.id, 'work'));
  }, [project.id, workspace]);

  useEffect(() => {
    const handleNewAgentEvent = () => setNewAgentOpen(true);
    const handleProjectShellEvent = () => {
      void shell();
    };
    const handleNewAISession = () => setDirectSession({ mode: 'direct', prompt: '' });
    window.addEventListener('nexus:new-agent', handleNewAgentEvent);
    window.addEventListener('nexus:new-ai-session', handleNewAISession);
    window.addEventListener('nexus:project-shell', handleProjectShellEvent);
    return () => {
      window.removeEventListener('nexus:new-agent', handleNewAgentEvent);
      window.removeEventListener('nexus:new-ai-session', handleNewAISession);
      window.removeEventListener('nexus:project-shell', handleProjectShellEvent);
    };
  }, [shell]);

  const commands = useMemo<NexusCommand[]>(
    () => [
      {
        id: 'projects',
        label: t('commands.open', { name: t('projectManager.desktopsTitle') }),
        group: t('commands.project'),
        keywords: ['workspace', 'desktops', 'hub'],
        run: () => openKind('projects'),
      },
      {
        id: 'overview',
        label: t('commands.open', { name: t('nav.overview') }),
        group: t('commands.project'),
        keywords: ['home'],
        run: () => openKind('overview'),
      },
      {
        id: 'terminals',
        label: t('commands.open', { name: t('nav.terminals') }),
        group: t('commands.project'),
        keywords: ['pty', 'shell'],
        run: () => openKind('terminals'),
      },
      {
        id: 'new-ai-session',
        label: t('commands.newAiSession', { defaultValue: 'New AI Session' }),
        group: t('commands.project'),
        keywords: ['agent', 'session', 'direct', 'create', 'terminal'],
        run: openNewAISession,
      },
      {
        id: 'project-shell',
        label: t('commands.newTerminal', { defaultValue: 'New Terminal' }),
        group: t('commands.project'),
        keywords: ['shell', 'terminal', 'bash', 'powershell'],
        run: () => void shell(),
      },
      {
        id: 'agents',
        label: t('commands.open', { name: t('nav.agents') }),
        group: t('commands.project'),
        keywords: ['fleet', 'workers', 'terminals'],
        run: () => openKind('agents'),
      },
      ...data.agents.flatMap((agent) => [
        {
          id: `terminal-${agent.id}`,
          label: t('commands.open', { name: `${agent.name} terminal` }),
          group: t('nav.agents'),
          keywords: ['terminal', agent.role],
          run: () => terminal(agent),
        },
        {
          id: `config-${agent.id}`,
          label: t('commands.configure', { name: agent.name }),
          group: t('nav.agents'),
          keywords: ['settings', agent.role],
          run: () => config(agent),
        },
      ]),
      ...(activeAgentRuntimeId
        ? executionRoles.map((role) => ({
            id: `execution-role-${role.id}`,
            label: t('commands.switchExecutionRole', {
              defaultValue: 'Switch execution role: {{role}}',
              role: role.label,
            }),
            group: t('nav.agents'),
            description: t('commands.switchExecutionRoleDescription', {
              defaultValue: 'Change the operational role of the active AI terminal',
            }),
            keywords: ['role', 'agent', 'review', 'investigate', 'verify', role.id],
            run: () =>
              window.dispatchEvent(
                new CustomEvent('nexus:execution-role-switch', {
                  detail: { runtimeId: activeAgentRuntimeId, roleId: role.id },
                }),
              ),
          }))
        : []),
      {
        id: 'work',
        label: t('commands.openComposer', { defaultValue: 'Open Composer' }),
        group: t('commands.project'),
        keywords: ['composer', 'prompt', 'goal', 'plan'],
        run: () => openKind('work'),
      },
      {
        id: 'plan',
        label: t('commands.openFlowRuns', { defaultValue: 'Open Flow Runs history' }),
        group: t('commands.project'),
        keywords: ['flow', 'mission', 'history', 'runs'],
        run: () => openKind('missions'),
      },
      {
        id: 'resources',
        label: t('commands.open', { name: t('nav.resources') }),
        group: 'Nexus',
        keywords: ['quota', 'provider', 'accounts'],
        run: () => openKind('resources'),
      },
      {
        id: 'maestro-control',
        label: t('maestroControl.title'),
        group: 'Nexus',
        keywords: ['skills', 'gates', 'update', 'library'],
        run: () => setMaestroControlOpen(true),
      },
      {
        id: 'project-manager',
        label: t('projectManager.title'),
        group: t('commands.project'),
        keywords: ['switch', 'create', 'workspace'],
        run: () => openKind('projects'),
      },
      {
        id: 'sessions',
        label: t('commands.open', { name: t('nav.sessions') }),
        group: t('commands.project'),
        keywords: ['resume', 'continuity'],
        run: () => openKind('sessions'),
      },
      {
        id: 'settings',
        label: t('commands.open', { name: t('nav.settings') }),
        group: 'Nexus',
        keywords: ['theme', 'accessibility'],
        run: () => openKind('settings'),
      },
      {
        id: 'runtime',
        label: t('commands.open', { name: t('nav.runtimes') }),
        group: t('commands.advanced'),
        keywords: ['runtime', 'legacy'],
        run: () => openKind('legacy-runtimes'),
      },
      {
        id: 'providers',
        label: t('commands.open', { name: t('nav.providers') }),
        group: t('commands.advanced'),
        keywords: ['provider'],
        run: () => openKind('legacy-providers'),
      },
      ...flowRuns.map((run) => ({
        id: `flow-run-${run.id}`,
        label: `Flow Run · ${run.id.slice(-6)}`,
        group: t('commands.project'),
        keywords: ['flow', 'run', 'mission', run.state],
        run: () => open(flowRunSurface(run.id, `Flow Run · ${run.id.slice(-6)}`)),
      })),
      ...[
        {
          id: 'preset-automatic',
          label: t('layoutPreset.automatic', { defaultValue: 'Layout: Automatic (Smart)' }),
          group: t('layoutPreset.group', { defaultValue: 'Layout Presets' }),
          preset: 'automatic' as const,
        },
        {
          id: 'preset-terminal-focus',
          label: t('layoutPreset.terminalFocus', {
            defaultValue: 'Layout: Terminal Focus (68/32)',
          }),
          group: t('layoutPreset.group', { defaultValue: 'Layout Presets' }),
          preset: 'terminal-focus' as const,
        },
        {
          id: 'preset-two-columns',
          label: t('layoutPreset.twoColumns', { defaultValue: 'Layout: Two Columns (50/50)' }),
          group: t('layoutPreset.group', { defaultValue: 'Layout Presets' }),
          preset: 'two-columns' as const,
        },
        {
          id: 'preset-three-columns',
          label: t('layoutPreset.threeColumns', {
            defaultValue: 'Layout: Three Columns (33/33/33)',
          }),
          group: t('layoutPreset.group', { defaultValue: 'Layout Presets' }),
          preset: 'three-columns' as const,
        },
        {
          id: 'preset-terminal-chat',
          label: t('layoutPreset.terminalChat', { defaultValue: 'Layout: Terminal + Chat' }),
          group: t('layoutPreset.group', { defaultValue: 'Layout Presets' }),
          preset: 'terminal-chat' as const,
        },
        {
          id: 'preset-terminal-flow',
          label: t('layoutPreset.terminalFlow', { defaultValue: 'Layout: Terminal + Flow' }),
          group: t('layoutPreset.group', { defaultValue: 'Layout Presets' }),
          preset: 'terminal-flow' as const,
        },
        {
          id: 'preset-focus-mode',
          label: t('layoutPreset.focusMode', {
            defaultValue: 'Layout: Focus Mode (Active window)',
          }),
          group: t('layoutPreset.group', { defaultValue: 'Layout Presets' }),
          preset: 'focus-mode' as const,
        },
        {
          id: 'preset-restore-default',
          label: 'Layout: Restore Default',
          group: 'Layout Presets',
          preset: 'restore-default' as const,
        },
      ].map((p) => ({
        id: p.id,
        label: p.label,
        group: p.group,
        keywords: ['layout', 'preset', 'window', 'arrange', 'grid', 'columns', 'tile'],
        run: () => {
          presentation.setMode('MOSAIC');
          presentation.rearrangePreset(p.preset);
        },
      })),
      {
        id: 'welcome',
        label: t('welcome.title'),
        group: t('commands.help'),
        keywords: ['guide', 'help', 'onboarding'],
        run: () => setWelcomeOpen(true),
      },
      {
        id: 'tour',
        label: t('commands.tour'),
        group: t('commands.help'),
        keywords: ['help', 'onboarding'],
        run: () => setTour(true),
      },
    ],
    [
      data.agents,
      executionRoles,
      activeAgentRuntimeId,
      flowRuns,
      t,
      presentation,
      open,
      openKind,
      openNewAISession,
      terminal,
      config,
      shell,
    ],
  );

  const handleNexusVoiceTranscript = useCallback(
    async (transcript: string) => {
      const intent = interpretNexusVoiceCommand(transcript);
      if (intent.kind === 'voice-settings') {
        setSettingsInitialTab('accessibility');
        openKind('settings');
        return;
      }
      if (intent.kind === 'palette') {
        setPalette(true);
        return;
      }
      if (intent.kind === 'command') {
        const command = commands.find((entry) => entry.id === intent.commandId);
        if (!command) throw new Error(`Comando de voz indisponível: ${intent.commandId}`);
        await command.run();
        return;
      }
      throw new Error(`Comando de voz não reconhecido: "${transcript}"`);
    },
    [commands, openKind],
  );

  // Scoped Keyboard Shortcuts via KeyboardShortcutRegistry
  useEffect(() => {
    const registry = KeyboardShortcutRegistry.getInstance();
    const unregisterK = registry.register({
      id: 'open-palette',
      key: 'k',
      ctrlOrMeta: true,
      scope: 'global',
      description: 'Open Command Palette',
      preventDefault: true,
      action: () => setPalette((value) => !value),
    });
    const unregisterShiftP = registry.register({
      id: 'open-palette-shift',
      key: 'p',
      ctrlOrMeta: true,
      shift: true,
      scope: 'global',
      description: 'Open Command Palette',
      preventDefault: true,
      action: () => setPalette(true),
    });
    const unregisterP = registry.register({
      id: 'open-projects',
      key: 'p',
      ctrlOrMeta: true,
      shift: false,
      scope: 'global',
      description: 'Open Project Hub',
      preventDefault: true,
      action: () => openKind('projects'),
    });
    const unregisterZen = registry.register({
      id: 'toggle-zen',
      key: 'f',
      ctrlOrMeta: true,
      shift: true,
      scope: 'global',
      description: 'Toggle Focus / Zen Mode',
      preventDefault: true,
      action: () => presentation.toggleZenMode(),
    });
    const unregisterF11 = registry.register({
      id: 'toggle-zen-f11',
      key: 'F11',
      scope: 'global',
      description: 'Toggle Focus / Zen Mode',
      preventDefault: true,
      action: () => presentation.toggleZenMode(),
    });
    const unregisterRail = registry.register({
      id: 'toggle-rail',
      key: 'b',
      ctrlOrMeta: true,
      shift: false,
      scope: 'global',
      description: 'Toggle Project Rail',
      preventDefault: true,
      action: () => setRailOpen((prev) => !prev),
    });
    const unregisterNewTerm = registry.register({
      id: 'new-terminal-shortcut',
      key: 't',
      ctrlOrMeta: true,
      shift: true,
      scope: 'global',
      description: 'New Terminal',
      preventDefault: true,
      action: () => {
        void shell();
      },
    });
    const unregisterExecutionRole = registry.register({
      id: 'focus-execution-role',
      key: 'a',
      ctrlOrMeta: true,
      shift: true,
      scope: 'global',
      description: 'Focus Execution Role Selector',
      preventDefault: true,
      action: () => {
        if (!activeAgentRuntimeId) return;
        const selector = Array.from(
          document.querySelectorAll<HTMLSelectElement>('[data-execution-role-selector="true"]'),
        ).find((element) => element.dataset.runtimeId === activeAgentRuntimeId);
        selector?.focus();
      },
    });
    const unregisterAltTabs = [1, 2, 3, 4, 5, 6, 7, 8, 9].map((num) =>
      registry.register({
        id: `switch-pty-${num}`,
        key: String(num),
        alt: true,
        scope: 'global',
        description: `Switch to Terminal ${num}`,
        preventDefault: true,
        action: () => {
          const ptys = listSurfaces(workspace.model.root).filter(isPtySurface);
          const target = ptys[num - 1];
          if (target) {
            presentation.setActivePty(surfaceViewId(target));
            presentation.focus(surfaceViewId(target));
          }
        },
      }),
    );

    return () => {
      unregisterK();
      unregisterShiftP();
      unregisterP();
      unregisterZen();
      unregisterF11();
      unregisterRail();
      unregisterNewTerm();
      unregisterExecutionRole();
      unregisterAltTabs.forEach((unreg) => unreg());
    };
  }, [project.id, presentation, workspace.model.root, openKind, shell, activeAgentRuntimeId]);

  const handleProjectUpdated = (updated: Project) => {
    data.setProjects((cur) => cur.map((p) => (p.id === updated.id ? updated : p)));
    if (project.id === updated.id) {
      setProject(updated);
    }
  };

  const handleProjectDeleted = (deleted: Project) => {
    const remaining = data.projects.filter((entry) => entry.id !== deleted.id);
    data.setProjects(remaining);
    if (project.id !== deleted.id) return;
    const next = remaining[0];
    if (next) {
      setProject(next);
    } else {
      navigate('/projects');
    }
  };

  const renderer = (
    <WorkspaceRenderer
      renderSurface={(surface) => (
        <WorkspaceSurfaceHost
          surface={surface}
          project={project}
          projects={data.projects}
          agents={data.agents}
          workspaces={data.workspaces}
          runtimes={data.runtimes}
          flowRuns={allFlowRuns.filter((run) => !run.project_id || run.project_id === project.id)}
          providers={data.providers}
          profiles={data.profiles}
          events={data.events}
          refreshAgents={() => data.refreshAgents(project.id)}
          refreshGlobal={data.refreshGlobal}
          onSelectProject={setProject}
          onProjectCreated={(created) => {
            data.setProjects((current) => [created, ...current]);
            setProject(created);
          }}
          onProjectUpdated={handleProjectUpdated}
          openSurface={open}
          updateSurface={workspace.updateSurface}
          closeSurface={workspace.close}
          onTour={() => setWelcomeOpen(true)}
          settingsInitialTab={settingsInitialTab}
        />
      )}
      onRequestClose={requestCloseSurface}
      onActivateSurface={open}
      createActions={{
        onNewAgent: () => setNewAgentOpen(true),
        onNewAISession: openNewAISession,
        onProjectShell: () => {
          void shell();
        },
      }}
    />
  );

  const notifiedFingerprints = useRef<Set<string>>(new Set());
  const prefsRef = useRef(loadNotificationPrefs());

  useEffect(() => {
    const refreshPrefs = () => {
      prefsRef.current = loadNotificationPrefs();
    };
    window.addEventListener('nexus:notification-prefs', refreshPrefs);
    return () => window.removeEventListener('nexus:notification-prefs', refreshPrefs);
  }, []);

  // Keep agent name stable, append short status suffix, and sync attention markers.
  useEffect(() => {
    const allSurfaces = listSurfaces(workspace.model.root);
    const stacks = listStacks(workspace.model.root);
    const focusedIds = new Set(stacks.map((stack) => stack.activeId));
    const terminalsProductId = projectSurface(project.id, 'terminals').id;
    const activePtyViewId = presentation.state.activePtyViewId;

    data.agents.forEach((agent) => {
      const surfaceId = `agent:${agent.id}:terminal`;
      const runtime = currentRuntimeForAgent(data.runtimes, agent.id);
      const surface = allSurfaces.find((s) => isSurfaceMatch(s, surfaceId));
      if (!surface) return;

      const next = surfaceTitleFromAgent(agent.name, runtime);
      const focused = isPtyAttentionFocused({
        terminalsProductSurfaceId: terminalsProductId,
        agentViewId: surfaceViewId(surface),
        stackActiveIds: focusedIds,
        activePtyViewId,
      });
      const previousFingerprint = surface.data?.attentionFingerprint || '';
      const previousUnread = surface.data?.unreadAttention === 'true';
      let unread = previousUnread;
      if (focused) {
        unread = false;
      } else if (
        shouldMarkUnread({
          previousFingerprint,
          nextFingerprint: next.fingerprint,
          hasAttention: next.hasAttention,
          surfaceFocused: focused,
          attentionKind: next.attentionKind,
        })
      ) {
        unread = true;
      } else if (!next.hasAttention) {
        unread = false;
      }

      const providerLabel = runtime
        ? `${runtime.provider_id || runtime.provider || 'claude'}${
            runtime.profile_id && runtime.profile_id !== 'default' ? `:${runtime.profile_id}` : ''
          }`
        : '';

      const needsUpdate =
        surface.title !== next.title ||
        surface.data?.hasAttention !== (next.hasAttention ? 'true' : 'false') ||
        surface.data?.unreadAttention !== (unread ? 'true' : 'false') ||
        surface.data?.attentionKind !== next.attentionKind ||
        surface.data?.statusSuffix !== next.statusSuffix ||
        surface.data?.dynamicTitle !== next.dynamicTitle ||
        surface.data?.attentionFingerprint !== next.fingerprint ||
        surface.data?.providerLabel !== providerLabel ||
        surface.data?.agentName !== agent.name;

      if (needsUpdate) {
        workspace.updateSurface(surface.id, {
          title: next.title,
          data: {
            ...surface.data,
            agentName: agent.name,
            hasAttention: next.hasAttention ? 'true' : 'false',
            unreadAttention: unread ? 'true' : 'false',
            attentionKind: next.attentionKind,
            statusSuffix: next.statusSuffix,
            dynamicTitle: next.dynamicTitle,
            attentionFingerprint: next.fingerprint,
            providerLabel,
          },
        });
      }
    });

    document.title = buildDocumentTitle(project.name, data.runtimes);
  }, [
    data.runtimes,
    data.agents,
    project.name,
    project.id,
    workspace.model.root,
    presentation.state.activePtyViewId,
    workspace,
  ]);

  const handleFocusAttention = useCallback(
    (item: RadarRuntimeItem | { runtimeId: string; projectId?: string; agentId?: string }) => {
      const runtimeId = item.runtimeId;
      const runtime = data.runtimes.find((entry) => entry.runtime_id === runtimeId);
      const projectId =
        'projectId' in item && item.projectId ? item.projectId : runtime?.project_id;
      const agentId = 'agentId' in item && item.agentId ? item.agentId : runtime?.agent_id;
      const agentName =
        (agentId && data.agents.find((agent) => agent.id === agentId)?.name) ||
        runtime?.dynamic_title ||
        runtime?.title;

      const actions = planFocusAttention(
        { projectId, agentId, runtimeId },
        { currentProjectId: project.id, runtime, agentName },
      );
      const switchAction = actions.find((action) => action.type === 'switch-project');
      const openActions = actions.filter((action) => action.type !== 'switch-project');
      const runOpen = () => {
        for (const action of openActions) {
          if (action.type === 'open-agent-terminal') {
            open(agentTerminalSurface(action.agentId, action.title, '', action.runtimeId || ''));
            open(projectSurface(project.id, 'terminals'));
          } else if (action.type === 'open-project-shell') {
            open(projectShellSurface(action.projectId, action.runtimeId, action.title));
            open(projectSurface(action.projectId, 'terminals'));
          } else if (action.type === 'refresh-agents') {
            void data.refreshAgents(action.projectId).catch(() => undefined);
          }
        }
      };
      if (switchAction?.type === 'switch-project') {
        const next = data.projects.find((entry) => entry.id === switchAction.projectId);
        if (next) {
          setProject(next);
          window.setTimeout(runOpen, 0);
          return;
        }
      }
      runOpen();
    },
    [data, open, project.id, setProject],
  );

  // Attention watcher for the focused project only (radar remains global).
  useEffect(() => {
    const prefs = prefsRef.current;
    if (!prefs.notificationsEnabled && !prefs.soundEnabled) return;

    const stacks = listStacks(workspace.model.root);
    const focusedIds = new Set(stacks.map((stack) => stack.activeId));
    const allSurfaces = listSurfaces(workspace.model.root);
    const terminalsProductId = projectSurface(project.id, 'terminals').id;
    const activePtyViewId = presentation.state.activePtyViewId;

    for (const runtime of data.runtimes) {
      if (runtime.project_id && runtime.project_id !== project.id) continue;
      const provider = (runtime.provider_id || runtime.provider || '').toLowerCase();
      if (provider === 'shell') continue;

      const reason = runtime.attention_reason;
      if (
        reason !== 'QUESTION' &&
        reason !== 'APPROVAL' &&
        reason !== 'TASK_COMPLETED' &&
        reason !== 'ERROR'
      ) {
        continue;
      }
      if (reason === 'QUESTION' || reason === 'APPROVAL') {
        if (!isHonestNeedsUser(runtime)) continue;
      }
      const fingerprint = attentionFingerprintOf(runtime);
      if (!fingerprint || notifiedFingerprints.current.has(fingerprint)) continue;

      const agentSurface = allSurfaces.find(
        (surface) =>
          surface.type === 'terminal' &&
          surface.data?.agentId &&
          surface.data.agentId === runtime.agent_id,
      );
      const focused = agentSurface
        ? isPtyAttentionFocused({
            terminalsProductSurfaceId: terminalsProductId,
            agentViewId: surfaceViewId(agentSurface),
            stackActiveIds: focusedIds,
            activePtyViewId,
          })
        : false;
      if (focused) {
        notifiedFingerprints.current.add(fingerprint);
        continue;
      }

      notifiedFingerprints.current.add(fingerprint);

      const agentName =
        data.agents.find((agent) => agent.id === runtime.agent_id)?.name ||
        runtime.dynamic_title ||
        runtime.title ||
        undefined;
      const body = formatAttentionPushBody({
        reason,
        context: runtime.attention_context || runtime.last_task_summary || '',
        promptKind: runtime.prompt_kind,
        agentName,
        projectName: runtime.project_name || project.name,
        rich: false,
      });
      if (!body) continue;

      if (prefs.soundEnabled) playAttentionSound();

      if (prefs.notificationsEnabled) {
        pushNotifications.sendPush({
          runtimeId: runtime.runtime_id,
          projectName: runtime.project_name || project.name,
          agentName,
          reason,
          context: runtime.attention_context || runtime.last_task_summary || body,
          dynamicTitle: runtime.dynamic_title,
          fingerprint,
          promptKind: runtime.prompt_kind,
          onClick: () =>
            handleFocusAttention({
              runtimeId: runtime.runtime_id,
              projectId: runtime.project_id,
              agentId: runtime.agent_id,
            }),
        });
      }
    }
  }, [
    data.runtimes,
    data.agents,
    project.id,
    project.name,
    workspace.model.root,
    presentation.state.activePtyViewId,
    handleFocusAttention,
  ]);

  if (popout) return <div className="nx-popout-shell">{renderer}</div>;

  const rail = (
    <ProjectRail
      projects={data.projects}
      selected={project}
      open={railOpen}
      onClose={() => setRailOpen(false)}
      onSelect={setProject}
      onCreated={(created) => {
        data.setProjects((current) => [created, ...current]);
        setProject(created);
      }}
      onProjectUpdated={handleProjectUpdated}
      onProjectDeleted={handleProjectDeleted}
      onOpenAgent={(agent) => {
        const target = data.projects.find((entry) => entry.id === agent.project_id);
        if (target && target.id !== project.id) {
          setProject(target);
          window.setTimeout(() => terminal(agent), 0);
          return;
        }
        terminal(agent);
      }}
      onOpenAttention={(targetProject, item) => {
        navigate(buildProjectRoute(targetProject.id, 'missions', item.mission_id));
      }}
      onNewAgent={() => setNewAgentOpen(true)}
      onNewAISession={openNewAISession}
      onProjectShell={() => {
        void shell();
      }}
      onOpenGlobal={(kind) => {
        if (kind === 'overview') openKind('overview');
        else openKind(kind);
      }}
    />
  );

  const handleFocusRuntime = (runtimeId: string) => {
    handleFocusAttention({ runtimeId });
  };

  return (
    <>
      <NexusShell
        project={project}
        agents={data.agents}
        runtimes={data.runtimes}
        events={data.events}
        rail={rail}
        zenMode={presentation.state.zenMode}
        onToggleZenMode={presentation.toggleZenMode}
        onOpenRail={() => setRailOpen(true)}
        onOpenSurface={openKind}
        onCommand={() => setPalette(true)}
        onOpenWelcome={() => setWelcomeOpen(true)}
        onOpenProjectManager={() => openKind('projects')}
        onSettings={(tab) => {
          setSettingsInitialTab(tab || 'appearance');
          openKind('settings');
        }}
        onVoiceTranscript={handleNexusVoiceTranscript}
        onNewAgent={() => setNewAgentOpen(true)}
        onNewAISession={openNewAISession}
        onProjectShell={() => {
          void shell();
        }}
        onFocusRuntime={handleFocusRuntime}
        onFocusAttention={handleFocusAttention}
        onFocusAgent={(agentId) => {
          const agent = data.agents.find((a) => a.id === agentId);
          if (agent) terminal(agent);
          else {
            const rt = currentRuntimeForAgent(data.runtimes, agentId);
            if (rt) handleFocusRuntime(rt.runtime_id);
          }
        }}
      >
        {shellError && (
          <div className="nx-workspace-global-error" role="alert">
            {shellError}
          </div>
        )}
        <div className={shellStyles.workspaceContent} aria-busy={!layoutReady}>
          {renderer}
          {!layoutReady && !popout && (
            <div className={shellStyles.projectSwitchStatus} role="status" aria-live="polite">
              <LoaderCircle
                className={shellStyles.projectSwitchIcon}
                size={14}
                aria-hidden="true"
              />
              <span>{t('app.switchingProject', 'Carregando projeto…')}</span>
            </div>
          )}
        </div>
      </NexusShell>

      <NexusHud
        projectId={project.id}
        onOpenUsage={() => openKind('resources')}
        onOpenAgents={() => openKind('agents')}
        onOpenMissions={() => openKind('missions')}
        onOpenSettings={() => {
          setSettingsInitialTab('hud');
          openKind('settings');
        }}
      />

      {palette && (
        <React.Suspense fallback={null}>
          <CommandPalette open={palette} onClose={() => setPalette(false)} commands={commands} />
        </React.Suspense>
      )}

      {welcomeOpen && (
        <React.Suspense fallback={null}>
          <WelcomeModal
            open={welcomeOpen}
            onClose={() => setWelcomeOpen(false)}
            onStartTour={() => {
              setWelcomeOpen(false);
              setTour(true);
            }}
          />
        </React.Suspense>
      )}

      {maestroControlOpen && (
        <React.Suspense fallback={null}>
          <MaestroControlModal
            open={maestroControlOpen}
            onClose={() => setMaestroControlOpen(false)}
          />
        </React.Suspense>
      )}

      {newAgentOpen && (
        <React.Suspense fallback={null}>
          <NewAgentModal
            open={newAgentOpen}
            onClose={() => setNewAgentOpen(false)}
            project={project}
            onCreated={(created) => {
              data.setAgents((cur) => [created, ...cur]);
              terminal(created);
            }}
          />
        </React.Suspense>
      )}

      {directSession && (
        <React.Suspense fallback={null}>
          <DirectSessionLauncher
            open={!!directSession}
            project={project}
            request={directSession}
            onClose={() => setDirectSession(null)}
            refreshAgents={() => data.refreshAgents(project.id)}
            onStarted={(created, prompt) => {
              open(agentTerminalSurface(created.id, created.name, prompt));
              openTerminals();
            }}
          />
        </React.Suspense>
      )}

      {tour && (
        <React.Suspense fallback={null}>
          <ProductTour
            open={tour}
            onClose={() => {
              setTour(false);
              window.localStorage.setItem(tourKey, 'true');
            }}
          />
        </React.Suspense>
      )}

      {closeTarget && (
        <React.Suspense fallback={null}>
          <TerminalActionDialog
            close
            shell={closeTarget.type === 'project-shell'}
            onCancel={() => setCloseTarget(null)}
            onCloseTab={() => void closeConfirmed(false)}
            onStopRuntime={() => void closeConfirmed(true)}
          />
        </React.Suspense>
      )}
    </>
  );
};
