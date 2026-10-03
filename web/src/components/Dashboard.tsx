import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { RuntimeSession, ProviderInfo, Workspace } from '../types';
import { asArray } from '../lib/safeArray';
import {
  Cpu,
  Play,
  Square,
  ArrowRightLeft,
  FastForward,
  Trash2,
  Folder,
  Layers,
  History,
} from 'lucide-react';

interface DashboardProps {
  runtimes: RuntimeSession[];
  providers: ProviderInfo[];
  workspaces: Workspace[];
  activeWorkspace?: string;
  onOpenTerminal: (id: string) => void;
  onOpenStartModal: () => void;
  onOpenHandoffModal: (runtime: RuntimeSession) => void;
  onOpenContinueModal: (runtime: RuntimeSession) => void;
  onStopRuntime: (id: string) => void;
  onDeleteRuntime?: (id: string) => void;
  onCleanInactive?: () => void;
}

export const Dashboard: React.FC<DashboardProps> = ({
  runtimes,
  providers,
  workspaces,
  activeWorkspace,
  onOpenTerminal,
  onOpenStartModal,
  onOpenHandoffModal,
  onOpenContinueModal,
  onStopRuntime,
  onDeleteRuntime,
  onCleanInactive,
}) => {
  const { t } = useTranslation();
  const [showAllWorkspaces, setShowAllWorkspaces] = useState<boolean>(false);

  const safeRuntimes = asArray<RuntimeSession>(runtimes);
  const safeProviders = asArray<ProviderInfo>(providers);
  const safeWorkspaces = asArray<Workspace>(workspaces);

  const normalize = (p?: string) =>
    (p || '')
      .trim()
      .replace(/[\\/]+$/, '')
      .toLowerCase();

  // Filter runtimes by active workspace unless "All Projects" is toggled
  const projectRuntimes = safeRuntimes.filter((r) => {
    if (showAllWorkspaces || !activeWorkspace) return true;
    const normActive = normalize(activeWorkspace);
    const normR = normalize(r.workspace);
    return normR === normActive || normR.startsWith(normActive + '/');
  });

  // Only truly running or starting agents are "Live"
  const liveRuntimes = projectRuntimes.filter(
    (r) => r.state === 'RUNNING' || r.state === 'STARTING' || r.state === 'HANDOFF',
  );

  // Past/offline sessions go to Session History
  const pastRuntimes = projectRuntimes.filter(
    (r) => r.state !== 'RUNNING' && r.state !== 'STARTING' && r.state !== 'HANDOFF',
  );

  const activeTotalCount = safeRuntimes.filter(
    (r) => r.state === 'RUNNING' || r.state === 'STARTING',
  ).length;
  const installedCount = safeProviders.filter((p) => p.installed).length;

  return (
    <div className="space-y-6">
      {/* Top Banner & Stats */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div
          className="rounded-lg p-4"
          style={{
            background: 'var(--nx-surface)',
            border: '1px solid var(--nx-border)',
          }}
        >
          <div className="text-xs font-mono" style={{ color: 'var(--nx-text-soft)' }}>
            {t('legacy.supervisedRuntimes', 'Supervised Runtimes')}
          </div>
          <div className="mt-2 text-2xl font-bold font-mono" style={{ color: 'var(--nx-success)' }}>
            {liveRuntimes.length}
            {activeWorkspace && !showAllWorkspaces && (
              <span className="text-xs font-normal ml-2" style={{ color: 'var(--nx-muted)' }}>
                / {activeTotalCount} total
              </span>
            )}
          </div>
          <div className="mt-1 text-[11px]" style={{ color: 'var(--nx-muted)' }}>
            {activeWorkspace && !showAllWorkspaces
              ? t('legacy.activeCurrentProject', 'Active in current project')
              : t('legacy.activeAllProjects', 'Active across all projects')}
          </div>
        </div>

        <div
          className="rounded-lg p-4"
          style={{
            background: 'var(--nx-surface)',
            border: '1px solid var(--nx-border)',
          }}
        >
          <div className="text-xs font-mono" style={{ color: 'var(--nx-text-soft)' }}>
            {t('legacy.availableProviders', 'Available Providers')}
          </div>
          <div
            className="mt-2 text-2xl font-bold font-mono"
            style={{ color: 'var(--nx-accent-text)' }}
          >
            {installedCount} / {safeProviders.length}
          </div>
          <div className="mt-1 text-[11px]" style={{ color: 'var(--nx-muted)' }}>
            {t('legacy.installedLocalPath', 'Installed in local PATH')}
          </div>
        </div>

        <div
          className="rounded-lg p-4"
          style={{
            background: 'var(--nx-surface)',
            border: '1px solid var(--nx-border)',
          }}
        >
          <div className="text-xs font-mono" style={{ color: 'var(--nx-text-soft)' }}>
            {t('legacy.registeredProjects', 'Registered Projects')}
          </div>
          <div className="mt-2 text-2xl font-bold font-mono" style={{ color: 'var(--nx-text)' }}>
            {safeWorkspaces.length}
          </div>
          <div className="mt-1 text-[11px]" style={{ color: 'var(--nx-muted)' }}>
            {t('legacy.workspacesInConfig', 'Workspaces in config')}
          </div>
        </div>

        <div
          className="rounded-lg p-4 flex flex-col justify-between"
          style={{
            background: 'var(--nx-surface)',
            border: '1px solid var(--nx-border)',
          }}
        >
          <div className="text-xs font-mono" style={{ color: 'var(--nx-text-soft)' }}>
            {t('legacy.quickLaunch', 'Quick Launch')}
          </div>
          <button
            onClick={onOpenStartModal}
            className="mt-2 w-full flex items-center justify-center space-x-2 py-2 px-3 hover:opacity-95 text-[var(--nx-accent-contrast)] rounded text-xs font-semibold shadow-md transition"
            style={{ background: 'var(--nx-accent)' }}
          >
            <Play className="w-3.5 h-3.5 fill-current" />
            <span>{t('legacy.launchAgent', 'Launch Agent')}</span>
          </button>
        </div>
      </div>

      {/* Scope Filter Switcher */}
      {activeWorkspace && (
        <div
          className="flex items-center justify-between px-3 py-2 rounded-lg text-xs font-mono"
          style={{
            background: 'var(--nx-surface)',
            border: '1px solid var(--nx-border)',
          }}
        >
          <div className="flex items-center space-x-2" style={{ color: 'var(--nx-text-soft)' }}>
            <Folder className="w-4 h-4" style={{ color: 'var(--nx-accent-text)' }} />
            <span>{t('legacy.scope', 'Scope:')}</span>
            <span className="font-semibold truncate max-w-sm" style={{ color: 'var(--nx-text)' }}>
              {activeWorkspace}
            </span>
          </div>

          <div
            className="flex items-center space-x-1.5 p-1 rounded-md"
            style={{
              background: 'var(--nx-bg)',
              border: '1px solid var(--nx-border)',
            }}
          >
            <button
              onClick={() => setShowAllWorkspaces(false)}
              className="px-2.5 py-1 rounded text-[11px] font-medium transition flex items-center space-x-1"
              style={{
                background: !showAllWorkspaces ? 'var(--nx-accent)' : 'transparent',
                color: !showAllWorkspaces ? 'var(--nx-accent-contrast)' : 'var(--nx-text-soft)',
              }}
            >
              <span>{t('legacy.projectOnly', 'Project Only')}</span>
              <span className="ml-1 px-1.5 py-0.2 rounded-full text-[10px] opacity-80">
                {liveRuntimes.length}
              </span>
            </button>
            <button
              onClick={() => setShowAllWorkspaces(true)}
              className="px-2.5 py-1 rounded text-[11px] font-medium transition flex items-center space-x-1"
              style={{
                background: showAllWorkspaces ? 'var(--nx-accent)' : 'transparent',
                color: showAllWorkspaces ? 'var(--nx-accent-contrast)' : 'var(--nx-text-soft)',
              }}
            >
              <Layers className="w-3 h-3" />
              <span>{t('legacy.allProjects', 'All Projects')}</span>
              <span className="ml-1 px-1.5 py-0.2 rounded-full text-[10px] opacity-80">
                {safeRuntimes.length}
              </span>
            </button>
          </div>
        </div>
      )}

      {/* LIVE RUNTIMES CARD (Active Only) */}
      <div
        className="rounded-lg overflow-hidden shadow-sm"
        style={{
          background: 'var(--nx-surface)',
          border: '1px solid var(--nx-border)',
        }}
      >
        <div
          className="px-4 py-3 flex items-center justify-between"
          style={{ borderBottom: '1px solid var(--nx-border)' }}
        >
          <div className="flex items-center space-x-2.5">
            <span
              className="w-2.5 h-2.5 rounded-full animate-pulse"
              style={{ background: 'var(--nx-success)' }}
            />
            <h2
              className="text-xs font-bold font-mono uppercase tracking-wider"
              style={{ color: 'var(--nx-text)' }}
            >
              {t('legacy.liveRuntimesTitle', {
                count: liveRuntimes.length,
                defaultValue: `Live Runtimes (${liveRuntimes.length})`,
              })}
            </h2>
          </div>
          <span className="text-[11px] font-mono" style={{ color: 'var(--nx-muted)' }}>
            {t('legacy.activeProcesses', 'Active coding processes')}
          </span>
        </div>

        {liveRuntimes.length === 0 ? (
          <div
            className="p-8 text-center text-xs font-mono space-y-2"
            style={{ color: 'var(--nx-muted)' }}
          >
            <div>{t('legacy.noLiveAgents', 'No live agent runtimes active in this project.')}</div>
            <div className="text-[11px]">
              {t(
                'legacy.launchHint',
                'Click "Launch Agent" above or run nexus start <provider> to start one.',
              )}
            </div>
          </div>
        ) : (
          <div>
            {liveRuntimes.map((r) => {
              const prov = r.provider_id || r.provider || 'AI';
              const prof = r.profile_id || r.profile || 'default';
              const title = r.title || `${prov.toUpperCase()} (${prof})`;
              return (
                <div
                  key={r.runtime_id}
                  className="p-4 flex items-center justify-between transition hover:bg-[var(--nx-surface-2)]"
                  style={{ borderBottom: '1px solid var(--nx-border)' }}
                >
                  <div className="flex items-center space-x-3">
                    <div
                      className="w-3 h-3 rounded-full animate-pulse"
                      style={{ background: 'var(--nx-success)' }}
                    />
                    <div>
                      <div className="flex items-center space-x-2">
                        <span
                          className="font-bold text-sm font-sans"
                          style={{ color: 'var(--nx-text)' }}
                        >
                          {title}
                        </span>
                        <span
                          className="text-[10px] px-1.5 py-0.5 rounded font-mono uppercase"
                          style={{
                            background: 'var(--nx-surface-2)',
                            color: 'var(--nx-accent-text)',
                            border: '1px solid var(--nx-border)',
                          }}
                        >
                          {prov}
                        </span>
                        <span
                          className="text-xs px-2 py-0.5 rounded font-mono"
                          style={{
                            background: 'var(--nx-bg-elevated)',
                            color: 'var(--nx-text-soft)',
                            border: '1px solid var(--nx-border)',
                          }}
                        >
                          {prof}
                        </span>
                        <span className="text-xs font-mono" style={{ color: 'var(--nx-muted)' }}>
                          ID: {r.runtime_id}
                        </span>
                        {r.handoff_type && (
                          <span
                            className="text-[10px] px-1.5 py-0.5 rounded font-mono"
                            style={{
                              background: 'var(--nx-surface-2)',
                              color: 'var(--nx-accent-text)',
                              border: '1px solid var(--nx-border)',
                            }}
                          >
                            {r.handoff_type}
                          </span>
                        )}
                      </div>
                      <div
                        className="mt-1 text-xs font-mono truncate max-w-md"
                        style={{ color: 'var(--nx-text-soft)' }}
                      >
                        {r.workspace}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => onOpenTerminal(r.runtime_id)}
                      className="px-3 py-1.5 rounded text-xs font-medium transition flex items-center space-x-1.5 shadow-sm hover:bg-[var(--nx-surface-2)]"
                      style={{
                        background: 'var(--nx-bg-elevated)',
                        border: '1px solid var(--nx-border)',
                        color: 'var(--nx-text)',
                      }}
                    >
                      <Cpu className="w-3.5 h-3.5" />
                      <span>{t('legacy.terminal', 'Terminal')}</span>
                    </button>

                    <button
                      onClick={() => onOpenHandoffModal(r)}
                      className="px-2.5 py-1.5 rounded text-xs font-medium transition flex items-center space-x-1 hover:bg-[var(--nx-surface-2)]"
                      style={{
                        background: 'var(--nx-bg-elevated)',
                        border: '1px solid var(--nx-border)',
                        color: 'var(--nx-text-soft)',
                      }}
                      title={t('legacy.handoff', 'Handoff')}
                    >
                      <ArrowRightLeft className="w-3.5 h-3.5" />
                      <span className="hidden sm:inline">{t('legacy.handoff', 'Handoff')}</span>
                    </button>

                    <button
                      onClick={() => onOpenContinueModal(r)}
                      className="px-2.5 py-1.5 rounded text-xs font-medium transition flex items-center space-x-1 hover:bg-[var(--nx-surface-2)]"
                      style={{
                        background: 'var(--nx-bg-elevated)',
                        border: '1px solid var(--nx-border)',
                        color: 'var(--nx-text-soft)',
                      }}
                      title={t('legacy.continueAI', 'Continue')}
                    >
                      <FastForward className="w-3.5 h-3.5" />
                      <span className="hidden sm:inline">{t('legacy.continueAI', 'Continue')}</span>
                    </button>

                    <button
                      onClick={() => onStopRuntime(r.runtime_id)}
                      className="px-2.5 py-1.5 rounded text-xs font-medium transition hover:bg-[var(--nx-danger-soft)] text-[var(--nx-danger)]"
                      style={{
                        background: 'var(--nx-bg-elevated)',
                        border: '1px solid var(--nx-border)',
                      }}
                      title={t('legacy.stopProcess', 'Stop Process')}
                      aria-label={t('legacy.stopProcess', 'Stop Process')}
                    >
                      <Square className="w-3.5 h-3.5 fill-current" />
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* SESSION HISTORY CARD (Past / Offline Sessions) */}
      {pastRuntimes.length > 0 && (
        <div
          className="rounded-lg overflow-hidden"
          style={{
            background: 'var(--nx-surface)',
            border: '1px solid var(--nx-border)',
          }}
        >
          <div
            className="px-4 py-3 flex items-center justify-between"
            style={{
              background: 'var(--nx-bg-elevated)',
              borderBottom: '1px solid var(--nx-border)',
            }}
          >
            <div className="flex items-center space-x-2.5">
              <History className="w-4 h-4" style={{ color: 'var(--nx-muted)' }} />
              <h2
                className="text-xs font-bold font-mono uppercase tracking-wider"
                style={{ color: 'var(--nx-text-soft)' }}
              >
                {t('legacy.sessionHistoryTitle', {
                  count: pastRuntimes.length,
                  defaultValue: `Session History (${pastRuntimes.length})`,
                })}
              </h2>
            </div>
            {onCleanInactive && (
              <button
                onClick={onCleanInactive}
                className="px-2.5 py-1 rounded text-[10px] font-mono transition hover:bg-[var(--nx-surface-2)]"
                style={{
                  background: 'var(--nx-surface)',
                  border: '1px solid var(--nx-border)',
                  color: 'var(--nx-text-soft)',
                }}
                title={t('legacy.clearInactiveTitle', 'Clear all inactive/stale sessions')}
              >
                {t('legacy.clearInactive', 'Clear Inactive')}
              </button>
            )}
          </div>

          <div>
            {pastRuntimes.map((r) => {
              const prov = r.provider_id || r.provider || 'AI';
              const prof = r.profile_id || r.profile || 'default';
              const title = r.title || `${prov.toUpperCase()} (${prof})`;
              return (
                <div
                  key={r.runtime_id}
                  className="p-3.5 flex items-center justify-between transition hover:bg-[var(--nx-surface-2)]"
                  style={{ borderBottom: '1px solid var(--nx-border)' }}
                >
                  <div className="flex items-center space-x-3">
                    <div
                      className="w-2.5 h-2.5 rounded-full"
                      style={{ background: 'var(--nx-muted)' }}
                    />
                    <div>
                      <div className="flex items-center space-x-2">
                        <span
                          className="font-semibold text-sm font-sans"
                          style={{ color: 'var(--nx-text)' }}
                        >
                          {title}
                        </span>
                        <span
                          className="text-[10px] px-1.5 py-0.5 rounded font-mono uppercase"
                          style={{
                            background: 'var(--nx-surface-2)',
                            color: 'var(--nx-text-soft)',
                            border: '1px solid var(--nx-border)',
                          }}
                        >
                          {prov}
                        </span>
                        <span
                          className="text-xs px-2 py-0.5 rounded font-mono"
                          style={{
                            background: 'var(--nx-bg)',
                            color: 'var(--nx-text-soft)',
                            border: '1px solid var(--nx-border)',
                          }}
                        >
                          {prof}
                        </span>
                        <span className="text-xs font-mono" style={{ color: 'var(--nx-muted)' }}>
                          ID: {r.runtime_id}
                        </span>
                        <span
                          className="text-[10px] px-1.5 py-0.5 rounded font-mono uppercase"
                          style={{
                            background: 'var(--nx-surface-2)',
                            color:
                              r.state === 'STOPPED'
                                ? 'var(--nx-text-soft)'
                                : r.state === 'FAILED'
                                  ? 'var(--nx-danger)'
                                  : 'var(--nx-warning)',
                            border: '1px solid var(--nx-border)',
                          }}
                        >
                          {r.state}
                        </span>
                      </div>
                      <div
                        className="mt-0.5 text-xs font-mono truncate max-w-md"
                        style={{ color: 'var(--nx-muted)' }}
                      >
                        {r.workspace}
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center space-x-2">
                    <button
                      onClick={() => onOpenContinueModal(r)}
                      className="px-2.5 py-1 rounded text-xs font-medium transition flex items-center space-x-1 hover:bg-[var(--nx-surface-2)]"
                      style={{
                        background: 'var(--nx-bg-elevated)',
                        border: '1px solid var(--nx-border)',
                        color: 'var(--nx-text-soft)',
                      }}
                      title={t('legacy.resumeSession', 'Resume conversation with new session')}
                    >
                      <FastForward className="w-3.5 h-3.5" />
                      <span>{t('legacy.resume', 'Resume')}</span>
                    </button>

                    {onDeleteRuntime && (
                      <button
                        onClick={() => onDeleteRuntime(r.runtime_id)}
                        className="p-1 rounded transition hover:bg-[var(--nx-surface-2)] text-[var(--nx-danger)]"
                        title={t('legacy.deleteSessionRecord', 'Delete Session Record')}
                        aria-label={t('legacy.deleteSessionRecord', 'Delete Session Record')}
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
};
