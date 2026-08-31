import React, { useEffect, useState } from 'react';
import {
  ArrowUpCircle,
  Bell,
  BellRing,
  BrainCircuit,
  ChevronDown,
  CircleHelp,
  Command,
  Menu,
  MoonStar,
  Network,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, IconButton } from '../design-system';
import { LanguagePicker } from './components/LanguagePicker';
import { WorkspaceTaskbar } from '../workspace/WorkspaceTaskbar';
import { nexus } from '../nexus/api';
import { AttentionIntermediationBanner } from '../components/AttentionIntermediationBanner';
import { pushNotifications } from '../notifications/PushNotificationManager';
import type { Agent, Project, RuntimeSession } from '../types';
import { WorkspaceLauncher } from '../workspace/WorkspaceLauncher';

export const NexusShell: React.FC<{
  project: Project;
  agents: Agent[];
  runtimes?: RuntimeSession[];
  rail: React.ReactNode;
  children: React.ReactNode;
  onOpenRail: () => void;
  onOpenSurface?: (kind: string) => void;
  onCommand: () => void;
  onOpenWelcome: () => void;
  onOpenProjectManager: () => void;
  onOpenMaestroControl: () => void;
  onSettings: () => void;
  onFocusRuntime?: (runtimeId: string) => void;
}> = ({
  project,
  agents,
  runtimes = [],
  rail,
  children,
  onOpenRail,
  onOpenSurface,
  onCommand,
  onOpenWelcome,
  onOpenProjectManager,
  onOpenMaestroControl,
  onSettings,
  onFocusRuntime,
}) => {
  const { t } = useTranslation();
  const working = agents.filter((agent) => agent.status === 'WORKING').length;
  const attention = agents.filter((agent) =>
    ['FAILED', 'STALE', 'RECOVERABLE', 'RATE_LIMITED'].includes(agent.status)
  ).length;
  const [sysInfo, setSysInfo] = useState<{
    nexus_version: string;
    maestro_version: string;
    maestro_available: boolean;
    update_available: boolean;
  } | null>(null);

  useEffect(() => {
    nexus.getSystemUpdates().then(setSysInfo).catch(() => undefined);
  }, []);

  return (
    <div className="nx-os-shell">
      <a href="#nexus-workspace" className="nx-skip-link">
        {t('shell.skip')}
      </a>
      {rail}

      <div className="nx-os-main">
        {/* Top OS Header */}
        <header className="nx-topbar">
          <div className="nx-topbar__context">
            <IconButton className="nx-mobile-menu" label={t('shell.openProjects')} onClick={onOpenRail}>
              <Menu size={16} />
            </IconButton>

            {/* Clickable Project Switcher */}
            <button
              type="button"
              className="nx-topbar__project-btn"
              onClick={onOpenProjectManager}
              title="Open Project Manager (Ctrl+P)"
            >
              <span className="nx-project-avatar nx-project-avatar--top">
                {(project.name || '').slice(0, 2).toUpperCase()}
              </span>
              <span className="nx-project-info">
                <strong>{project.name}</strong>
                <small>{project.default_branch || 'unknown'}</small>
              </span>
              <ChevronDown size={12} className="nx-project-chevron" />
            </button>

            {onOpenSurface && <WorkspaceLauncher onOpen={onOpenSurface} />}

            <span className="nx-topbar__path" title={project.canonical_path}>
              {project.canonical_path}
            </span>
          </div>

          {/* Right Status Controls */}
          <div className="nx-topbar__status" data-tour="status">
            {sysInfo?.update_available && (
              <button
                type="button"
                onClick={onSettings}
                className="nx-update-indicator"
                title={t('settings.updates')}
              >
                <ArrowUpCircle size={13} className="nx-spin-slow" />
                <span>{t('settings.updates')}</span>
              </button>
            )}

            {/* Working Agents Count */}
            <Badge tone={attention ? 'warning' : 'success'}>
              <Network size={10} />
              {t('shell.working', { count: working })}
            </Badge>

            {/* Interactive Maestro Badge */}
            <button
              type="button"
              className="nx-maestro-badge-btn"
              onClick={onOpenMaestroControl}
              title="Open Orquestrador Maestro Control"
            >
              <Badge tone={project.maestro_mode === 'OFF' ? 'default' : 'brand'}>
                <BrainCircuit size={10} />
                <span>Maestro: {project.maestro_mode || 'ASSIST'}</span>
              </Badge>
            </button>

            {/* System Versions Pill */}
            <div className="nx-topbar-version-pill" title="Nexus & Maestro Versions">
              <span className="nx-ver-nexus">Nexus v{sysInfo?.nexus_version || 'unknown'}</span>
            </div>

            {/* Language Switcher */}
            <LanguagePicker />

            {/* Command Palette Trigger */}
            <button
              type="button"
              className="nx-command-trigger"
              data-tour="command"
              onClick={onCommand}
              title="Search & Commands (Ctrl+K)"
            >
              <Command size={13} />
              <span>{t('shell.search')}</span>
              <kbd>Ctrl K</kbd>
            </button>

            {/* Push Notifications Toggle */}
            <IconButton
              label="Notificações Push"
              onClick={async () => {
                const perm = await pushNotifications.requestPermission();
                if (perm === 'granted') {
                  pushNotifications.sendPush({
                    runtimeId: 'test',
                    projectName: project.name,
                    reason: 'TASK_COMPLETED',
                    context: 'Notificações push ativadas com sucesso para o IAPro Nexus!',
                  });
                }
              }}
            >
              {pushNotifications.getPermission() === 'granted' ? (
                <BellRing size={15} className="nx-text-emerald-400" />
              ) : (
                <Bell size={15} />
              )}
            </IconButton>

            {/* Help & Welcome Guide */}
            <IconButton label={t('shell.tour')} onClick={onOpenWelcome}>
              <CircleHelp size={15} />
            </IconButton>

            {/* Theme & Settings */}
            <IconButton label={t('shell.appearance')} onClick={onSettings}>
              <MoonStar size={15} />
            </IconButton>
          </div>
        </header>

        {/* Attention & Intermediation Alert Banner */}
        <AttentionIntermediationBanner
          runtimes={runtimes}
          onFocusRuntime={(rtId) => {
            if (onFocusRuntime) onFocusRuntime(rtId);
          }}
        />

        {/* Workspace Canvas (Tabs inside stacks) */}
        <main id="nexus-workspace" className="nx-workspace-host">
          {children}
        </main>

        {/* OS Status Bar */}
        <WorkspaceTaskbar project={project} agents={agents} />
      </div>
    </div>
  );
};
