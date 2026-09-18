import React, { useEffect, useState } from 'react';
import {
  ArrowUpCircle,
  Bell,
  BellRing,
  ChevronDown,
  CircleHelp,
  Command,
  Globe,
  Maximize2,
  Menu,
  Minimize2,
  MoonStar,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { IconButton } from '../design-system';
import { LanguagePicker } from './components/LanguagePicker';
import { FontScalePicker } from './components/FontScalePicker';
import { ChromeOverflowMenu } from './components/ChromeOverflowMenu';
import { WorkspaceTaskbar } from '../workspace/WorkspaceTaskbar';
import { nexus } from '../nexus/api';
import { AttentionIntermediationBanner } from '../components/AttentionIntermediationBanner';
import { pushNotifications } from '../notifications/PushNotificationManager';
import { InAppNotificationCenter } from '../notifications/InAppNotificationCenter';
import { ProjectCreateMenu } from '../features/projects/ProjectCreateMenu';
import type { Agent, EventRecord, Project, RuntimeSession } from '../types';
import type { RadarRuntimeItem } from './attentionRadarModel';
import styles from './NexusShell.module.scss';

export const NexusShell: React.FC<{
  project: Project;
  agents: Agent[];
  runtimes?: RuntimeSession[];
  events?: EventRecord[];
  rail: React.ReactNode;
  children: React.ReactNode;
  zenMode?: boolean;
  onToggleZenMode?: () => void;
  onOpenRail: () => void;
  onOpenSurface?: (kind: string) => void;
  onCommand: () => void;
  onOpenWelcome: () => void;
  onOpenProjectManager: () => void;
  onSettings: (tab?: 'appearance' | 'updates' | 'remote') => void;
  onNewAgent?: () => void;
  onNewAISession?: () => void;
  onProjectShell?: () => void;
  onFocusRuntime?: (runtimeId: string) => void;
  onFocusAttention?: (item: RadarRuntimeItem) => void;
  onFocusAgent?: (agentId: string) => void;
}> = ({
  project,
  agents,
  runtimes = [],
  events,
  rail,
  children,
  zenMode = false,
  onToggleZenMode,
  onOpenRail,
  onOpenSurface: _onOpenSurface,
  onCommand,
  onOpenWelcome,
  onOpenProjectManager,
  onSettings,
  onNewAgent,
  onNewAISession,
  onProjectShell,
  onFocusRuntime,
  onFocusAttention,
  onFocusAgent,
}) => {
  const { t } = useTranslation();
  const [notificationDrawerOpen, setNotificationDrawerOpen] = useState(false);
  const [sysInfo, setSysInfo] = useState<{
    nexus_version: string;
    maestro_version: string;
    maestro_available: boolean;
    update_available: boolean;
  } | null>(null);

  useEffect(() => {
    const load = () => {
      nexus
        .getSystemUpdates()
        .then(setSysInfo)
        .catch(() => undefined);
    };
    load();
    window.addEventListener('nexus:system-updates', load);
    return () => window.removeEventListener('nexus:system-updates', load);
  }, []);

  const hasAttentionAlerts = runtimes.some(
    (rt) =>
      rt.attention_kind === 'needs_user' ||
      rt.attention_reason === 'QUESTION' ||
      rt.attention_reason === 'APPROVAL' ||
      rt.attention_reason === 'ERROR' ||
      rt.state === 'FAILED',
  );

  return (
    <div
      className={
        zenMode ? `nx-os-shell nx-os-shell--zen ${styles.shell}` : `nx-os-shell ${styles.shell}`
      }
    >
      <a href="#nexus-workspace" className="nx-skip-link">
        {t('shell.skip')}
      </a>
      {rail}

      <div className={`nx-os-main ${styles.main}`}>
        <div className="nx-shell-chrome">
          {/* Top OS Header */}
          <header className={`nx-topbar ${styles.topbar}`}>
            <div className={`nx-topbar__context ${styles.context}`}>
              <IconButton
                className="nx-mobile-menu"
                label={t('shell.openProjects')}
                onClick={onOpenRail}
              >
                <Menu size={16} />
              </IconButton>

              {/* Clickable Project Switcher */}
              <button
                type="button"
                className={`nx-topbar__project-btn ${styles.projectButton}`}
                onClick={onOpenProjectManager}
                title={t('shell.projectManagerShortcut', 'Open Project Manager (Ctrl+P)')}
              >
                <span className="nx-project-avatar nx-project-avatar--top">
                  {(project.name || 'PR').slice(0, 2).toUpperCase()}
                </span>
                <span className="nx-project-info">
                  <strong>{project.name}</strong>
                  <small>{project.default_branch || 'unknown'}</small>
                </span>
                <ChevronDown size={12} className="nx-project-chevron" />
              </button>

              <ProjectCreateMenu
                onNewAgent={onNewAgent}
                onNewAISession={onNewAISession}
                onProjectShell={onProjectShell}
                size="sm"
                variant="topbar"
              />
            </div>

            {/* Right Status Controls */}
            <div className={`nx-topbar__status ${styles.status}`} data-tour="status">
              <div className="nx-chrome-wide">
                <div
                  className="nx-topbar-version-pill"
                  title={t('settings.nexusVersion', 'Nexus version')}
                >
                  <span className="nx-ver-nexus">Nexus v{sysInfo?.nexus_version || 'unknown'}</span>
                </div>

                {sysInfo?.update_available && (
                  <button
                    type="button"
                    onClick={() => onSettings('updates')}
                    className="nx-update-indicator"
                    title={t('settings.updates')}
                  >
                    <ArrowUpCircle size={13} className="nx-spin-slow" />
                    <span className="nx-update-badge">{t('settings.updates')}</span>
                  </button>
                )}

                <LanguagePicker />
                <FontScalePicker />
                <IconButton
                  label={t('settings.remote.title', 'Acesso remoto')}
                  onClick={() => onSettings('remote')}
                  title={t(
                    'settings.remote.description',
                    'Acesso Nexus de outro dispositivo via túnel Cloudflare',
                  )}
                >
                  <Globe size={15} aria-hidden="true" />
                </IconButton>

                <button
                  type="button"
                  className={`nx-command-trigger ${styles.commandButton}`}
                  data-tour="command"
                  onClick={onCommand}
                  title={t('shell.searchShortcut', 'Search & Commands (Ctrl+K)')}
                >
                  <Command size={13} />
                  <span>{t('shell.search')}</span>
                  <kbd>Ctrl K</kbd>
                </button>
              </div>

              <div className={styles.notificationWrap}>
                <IconButton
                  label={t('shell.notificationsAndRadar', 'Central de Notificações e Radar')}
                  onClick={() => setNotificationDrawerOpen((prev) => !prev)}
                  aria-expanded={notificationDrawerOpen}
                >
                  {pushNotifications.getPermission() === 'granted' ? (
                    <BellRing size={15} className="nx-text-emerald-400" />
                  ) : (
                    <Bell size={15} />
                  )}
                </IconButton>
                {hasAttentionAlerts && <span className={styles.attentionIndicator} />}
              </div>

              <div className="nx-chrome-wide">
                <IconButton
                  className="nx-topbar-tour-btn"
                  label={t('shell.tour')}
                  onClick={onOpenWelcome}
                >
                  <CircleHelp size={15} />
                </IconButton>

                {onToggleZenMode && (
                  <button
                    type="button"
                    className={styles.focusToggleBtn}
                    data-active={zenMode ? 'true' : 'false'}
                    onClick={onToggleZenMode}
                    title={zenMode ? t('workspace.exitFocusMode') : t('workspace.focusMode')}
                    aria-label={zenMode ? t('workspace.exitFocusMode') : t('workspace.focusMode')}
                  >
                    {zenMode ? <Minimize2 size={13} /> : <Maximize2 size={13} />}
                    <span className={styles.focusBtnText}>
                      {zenMode
                        ? t('workspace.exitFocus')
                        : t('workspace.focusModeShort', 'Modo Foco')}
                    </span>
                    <kbd>Ctrl+Shift+F</kbd>
                  </button>
                )}

                <IconButton label={t('shell.appearance')} onClick={() => onSettings('appearance')}>
                  <MoonStar size={15} />
                </IconButton>
              </div>

              <ChromeOverflowMenu
                zenMode={zenMode}
                updateAvailable={Boolean(sysInfo?.update_available)}
                onCommand={onCommand}
                onOpenWelcome={onOpenWelcome}
                onSettings={onSettings}
                onToggleZenMode={onToggleZenMode}
              />
            </div>
          </header>

          {/* Attention & Intermediation Alert Banner */}
          <AttentionIntermediationBanner
            runtimes={runtimes}
            focusedProjectId={project.id}
            onFocusRuntime={(rtId) => {
              if (onFocusRuntime) onFocusRuntime(rtId);
            }}
          />
        </div>

        {/* Workspace Canvas (Tabs inside stacks) */}
        <main id="nexus-workspace" className={`nx-workspace-host ${styles.workspace}`}>
          {children}
        </main>

        {/* OS Status Bar with interactive direct agent focus */}
        <WorkspaceTaskbar project={project} agents={agents} onFocusAgent={onFocusAgent} />
      </div>

      <InAppNotificationCenter
        runtimes={runtimes}
        events={events}
        focusedProjectId={project.id}
        drawerOpen={notificationDrawerOpen}
        onCloseDrawer={() => setNotificationDrawerOpen(false)}
        onFocusRuntime={(runtimeId) => onFocusRuntime?.(runtimeId)}
        onFocusAttention={(item) => onFocusAttention?.(item)}
      />
    </div>
  );
};
