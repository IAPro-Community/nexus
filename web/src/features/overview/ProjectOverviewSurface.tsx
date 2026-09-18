import React, { useEffect, useMemo, useState } from 'react';
import { ArrowUpCircle, Play, RotateCcw, Square, TerminalSquare } from 'lucide-react';
import { Badge, Button, Card, Dialog, EmptyState } from '../../design-system';
import { api } from '../../api';
import { nexus } from '../../nexus/api';
import { runtimeIdFromRecoverResult } from '../../nexus/agentTerminalModel';
import { ResourcePicker } from '../../nexus/ResourcePicker';
import { ProjectCreateActions } from '../projects/ProjectCreateActions';
import type { Agent, MissionRun, Project, RuntimeSession } from '../../types';
import { translateStatus } from '../../i18n';
import { useTranslation } from 'react-i18next';
import styles from './ProjectOverviewSurface.module.scss';
import { buildOverviewResume } from './overviewResumeModel';

const tone = (status: string) =>
  status === 'WORKING'
    ? 'success'
    : status === 'FAILED' || status === 'STALE'
      ? 'danger'
      : status === 'RECOVERABLE' || status === 'WAITING'
        ? 'warning'
        : 'default';

export const ProjectOverviewSurface: React.FC<{
  project: Project;
  agents: Agent[];
  runtimes?: RuntimeSession[];
  flowRuns?: MissionRun[];
  onOpenAgent: (agent: Agent, runtimeId?: string) => void;
  onNewAISession: () => void;
  onProjectShell: () => void;
  onOpenComposer?: () => void;
  onOpenFlow?: () => void;
  onNewAgent?: () => void;
  onConfigureAgent?: (agent: Agent) => void;
  refreshAgents?: () => Promise<void>;
  onStartAgent?: (agent: Agent) => Promise<{ runtime?: RuntimeSession } | void>;
  onRecoverAgent?: (agent: Agent) => Promise<{ runtime?: RuntimeSession } | void>;
  onOpenSettings?: () => void;
}> = ({
  project,
  agents,
  runtimes = [],
  flowRuns = [],
  onOpenAgent,
  onNewAISession,
  onProjectShell,
  onOpenComposer,
  onOpenFlow,
  onNewAgent,
  onConfigureAgent,
  refreshAgents,
  onStartAgent,
  onRecoverAgent,
  onOpenSettings,
}) => {
  const { t } = useTranslation();
  const [busy, setBusy] = useState<string>('');
  const [error, setError] = useState<string>('');
  const [resourceAgent, setResourceAgent] = useState<Agent | null>(null);
  const [updateInfo, setUpdateInfo] = useState<{
    nexus_version: string;
    maestro_version: string;
    maestro_latest_version?: string;
    maestro_available: boolean;
    update_available?: boolean;
  } | null>(null);

  useEffect(() => {
    let active = true;
    nexus
      .getSystemUpdates()
      .then((data) => {
        if (active) setUpdateInfo(data);
      })
      .catch(() => {});
    return () => {
      active = false;
    };
  }, []);

  const agentList = Array.isArray(agents) ? agents : [];
  const resumeItems = buildOverviewResume(project.id, runtimes, agentList);
  const needsYou = resumeItems.filter((item) => item.lane === 'needsYou');
  const continueItems = resumeItems.filter(
    (item) => item.lane === 'active' || item.lane === 'recoverable',
  );
  const recentItems = resumeItems.filter((item) => item.lane === 'recent').slice(0, 3);
  const recentRuns = (Array.isArray(flowRuns) ? flowRuns : []).slice(0, 3);
  const working = agentList.filter((agent) => agent.status === 'WORKING').length;
  const degraded = agentList.filter((agent) =>
    ['FAILED', 'STALE', 'RECOVERABLE', 'RATE_LIMITED'].includes(agent.status),
  ).length;

  const activeProjectRuntimes = useMemo(() => {
    const list = Array.isArray(runtimes) ? runtimes : [];
    return list.filter((r) => {
      if (r.state === 'STOPPED' || r.state === 'FAILED') return false;
      const ws = r.workspace || '';
      const pPath = project.canonical_path || '';
      const pId = project.id || '';
      return ws === pPath || (pPath && ws.startsWith(pPath)) || ws === pId || r.project_id === pId;
    });
  }, [runtimes, project]);

  const handleStopRuntime = async (runtimeId: string) => {
    try {
      await api.stopRuntime(runtimeId);
      await refreshAgents?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const handleRecover = async (agent: Agent) => {
    setBusy(agent.id);
    setError('');
    try {
      const res = await (onRecoverAgent ? onRecoverAgent(agent) : nexus.recoverAgent(agent.id));
      await refreshAgents?.();
      const runtimeId = runtimeIdFromRecoverResult(res);
      onOpenAgent(agent, runtimeId);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (message.includes('REQUIRED_RESOURCE_SELECTION')) {
        setResourceAgent(agent);
      } else {
        setError(message);
      }
    } finally {
      setBusy('');
    }
  };

  const handleStart = async (agent: Agent) => {
    setBusy(agent.id);
    setError('');
    try {
      const res = await (onStartAgent ? onStartAgent(agent) : nexus.startAgent(agent.id));
      await refreshAgents?.();
      const runtimeId =
        res && typeof res === 'object' && 'runtime' in res
          ? (res as any).runtime?.runtime_id
          : undefined;
      onOpenAgent(agent, runtimeId);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (message.includes('REQUIRED_RESOURCE_SELECTION')) {
        setResourceAgent(agent);
      } else {
        setError(message);
      }
    } finally {
      setBusy('');
    }
  };

  const allocateAndStart = async () => {
    if (!resourceAgent) return;
    setBusy(resourceAgent.id);
    setError('');
    try {
      const res = await nexus.startAgent(resourceAgent.id);
      await refreshAgents?.();
      setResourceAgent(null);
      onOpenAgent(resourceAgent, res?.runtime?.runtime_id);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy('');
    }
  };

  return (
    <div className="nx-surface-scroll">
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">Workspace OS · Terminais do Projeto</span>
          <h1>{project.name}</h1>
          <p>
            {project.canonical_path} ·{' '}
            <code className={styles.branchCode}>{project.default_branch || 'main'}</code>
          </p>
        </div>
        <div className="nx-page-header__actions">
          <ProjectCreateActions
            onNewAgent={onNewAgent}
            onNewAISession={onNewAISession}
            onProjectShell={onProjectShell}
            size="sm"
          />
          {onOpenComposer ? (
            <Button tone="ghost" onClick={onOpenComposer}>
              {t('overview.openComposer')}
            </Button>
          ) : null}
          {onOpenFlow ? (
            <Button tone="ghost" onClick={onOpenFlow}>
              {t('overview.openFlow', 'Flow')}
            </Button>
          ) : null}
        </div>
      </div>

      <section className={styles.resumePanel} aria-labelledby="overview-resume-title">
        <div className={styles.resumeHeader}>
          <div>
            <span className="nx-eyebrow">{t('overview.resumeEyebrow')}</span>
            <h2 id="overview-resume-title">{t('overview.continueWorking')}</h2>
          </div>
          <Badge tone={needsYou.length > 0 ? 'warning' : 'success'}>
            {needsYou.length > 0
              ? t('overview.needsAttention', { count: needsYou.length })
              : t('overview.noAttention')}
          </Badge>
        </div>
        <div className={styles.resumeGrid}>
          {(needsYou.length > 0 ? needsYou : continueItems).slice(0, 3).map((item) => {
            const target = item.agent;
            const title =
              target?.name ||
              item.runtime.dynamic_title ||
              item.runtime.title ||
              item.runtime.runtime_id;
            return (
              <Card key={item.runtime.runtime_id} className={styles.resumeCard}>
                <div className={styles.resumeCardBody}>
                  <strong>{title}</strong>
                  <span>
                    {item.lane === 'needsYou'
                      ? item.runtime.attention_context || t('overview.needsYou')
                      : item.runtime.last_task_summary || t('overview.activeWork')}
                  </span>
                </div>
                <div className={styles.resumeActions}>
                  {target ? (
                    <Button
                      size="sm"
                      tone="brand"
                      onClick={() => onOpenAgent(target, item.runtime.runtime_id)}
                    >
                      {item.lane === 'recoverable' ? t('overview.recover') : t('overview.continue')}
                    </Button>
                  ) : null}
                  <Button size="sm" tone="ghost" onClick={onProjectShell}>
                    {t('overview.openTerminal')}
                  </Button>
                </div>
              </Card>
            );
          })}
          {needsYou.length === 0 && continueItems.length === 0 ? (
            <EmptyState
              title={t('overview.noRecentWork')}
              hint={t('overview.noRecentWorkHint')}
              action={<Button onClick={onNewAISession}>{t('overview.newAISession')}</Button>}
            />
          ) : null}
        </div>
        {recentItems.length > 0 || recentRuns.length > 0 ? (
          <div className={styles.recentWork} aria-label={t('overview.recentWork')}>
            <strong>{t('overview.recentWork')}</strong>
            {recentItems.map((item) => (
              <span key={item.runtime.runtime_id}>
                {item.runtime.title || item.runtime.runtime_id}
              </span>
            ))}
            {recentRuns.map((run) => (
              <span key={run.id}>{t('overview.flowRun', { id: run.id.slice(-6) })}</span>
            ))}
          </div>
        ) : null}
      </section>

      {/* Live Supervised Runtimes Section */}
      <section className={styles.liveRuntimesPanel} aria-labelledby="overview-runtimes-title">
        <div className={styles.liveRuntimesHeader}>
          <div>
            <span className="nx-eyebrow">
              {t('overview.liveRuntimesEyebrow', 'Execução Ativa · PTY Host')}
            </span>
            <h2 id="overview-runtimes-title" style={{ margin: '2px 0 0' }}>
              {t('overview.liveRuntimesTitle', 'Runtimes Ativos no Projeto')} (
              {activeProjectRuntimes.length})
            </h2>
          </div>
          <Badge tone={activeProjectRuntimes.length > 0 ? 'success' : 'default'}>
            {activeProjectRuntimes.length > 0 ? (
              <>
                <span className={styles.pulseDot} />
                <span style={{ marginLeft: 6 }}>
                  {t('overview.runtimesRunning', {
                    count: activeProjectRuntimes.length,
                    defaultValue: `${activeProjectRuntimes.length} sessões ativas`,
                  })}
                </span>
              </>
            ) : (
              t('overview.noActiveRuntimes', 'Nenhum runtime ativo')
            )}
          </Badge>
        </div>

        {activeProjectRuntimes.length > 0 ? (
          <div className={styles.liveRuntimesGrid}>
            {activeProjectRuntimes.map((rt) => {
              const matchedAgent = agentList.find(
                (a) => a.id === rt.agent_id || a.name === rt.title,
              );
              return (
                <div key={rt.runtime_id} className={styles.runtimeCard}>
                  <div className={styles.runtimeHead}>
                    <span className={styles.pulseDot} />
                    <span className={styles.runtimeTitle}>
                      {rt.dynamic_title || rt.title || rt.last_task_summary || rt.runtime_id}
                    </span>
                    <Badge tone="default">{rt.state}</Badge>
                  </div>
                  <div className={styles.runtimeMeta}>
                    <span>
                      <strong>{rt.provider_id || rt.provider || 'CLI'}</strong>
                      {rt.profile_id || rt.profile ? `:${rt.profile_id || rt.profile}` : ''}
                    </span>
                    <span>·</span>
                    <span style={{ fontFamily: 'var(--nx-font-mono, monospace)' }}>
                      PID {rt.pid || rt.host_pid || '—'}
                    </span>
                  </div>
                  <div className={styles.runtimeActions}>
                    <Button
                      size="sm"
                      tone="brand"
                      onClick={() => {
                        if (matchedAgent) {
                          onOpenAgent(matchedAgent, rt.runtime_id);
                        } else {
                          onProjectShell();
                        }
                      }}
                    >
                      <TerminalSquare size={12} /> {t('overview.attachTerminal', 'Anexar Terminal')}
                    </Button>
                    <Button
                      size="sm"
                      tone="danger"
                      onClick={() => void handleStopRuntime(rt.runtime_id)}
                    >
                      <Square size={11} /> {t('common.stop', 'Parar')}
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        ) : (
          <p style={{ margin: 0, fontSize: '0.8125rem', color: 'var(--nx-text-muted)' }}>
            {t(
              'overview.noActiveRuntimesHint',
              'Nenhum processo supervisionado ativo para este workspace. Dispare uma sessão ou agente para iniciar.',
            )}
          </p>
        )}
      </section>

      {updateInfo?.update_available && (
        <Card className={`nx-inline-alert ${styles.updateAlert}`} data-tone="warning">
          <div className={styles.updateAlertContent}>
            <ArrowUpCircle size={18} className="nx-spin-slow" />
            <div>
              <strong>{t('settings.updates', 'Atualização disponível')}</strong>
              <p className={styles.updateAlertText}>
                Nexus v{updateInfo.nexus_version} · Maestro{' '}
                {updateInfo.maestro_latest_version
                  ? `v${updateInfo.maestro_latest_version}`
                  : 'nova versão disponível'}
              </p>
            </div>
          </div>
          <Button
            size="sm"
            tone="brand"
            onClick={() => {
              if (onOpenSettings) onOpenSettings();
              else window.dispatchEvent(new CustomEvent('nexus:open-settings'));
            }}
          >
            Ver Detalhes
          </Button>
        </Card>
      )}

      <div className={styles.fleetContainer}>
        <div className="nx-section-title">
          <div>
            <h2>
              {t('overview.fleet')} ({agentList.length})
            </h2>
            <p>{t('overview.fleetDescription')}</p>
          </div>
          <div className={styles.fleetStatusBadges}>
            <Badge tone={degraded > 0 ? 'warning' : 'success'}>
              {degraded > 0
                ? t('overview.degraded', {
                    count: degraded,
                    defaultValue: `${degraded} degradados`,
                  })
                : t('overview.healthy')}
            </Badge>
            {working > 0 && <Badge tone="success">{working} em execução</Badge>}
          </div>
        </div>

        {error && <Card className="nx-inline-error">{error}</Card>}

        {agentList.length === 0 ? (
          <EmptyState
            icon={<TerminalSquare size={36} />}
            title={t('overview.noAgents')}
            hint={t('overview.noAgentsHint')}
            action={
              <div className={`nx-project-create-actions ${styles.emptyActions}`} data-size="md">
                {onNewAgent && (
                  <Button tone="brand" onClick={onNewAgent}>
                    Novo Agente
                  </Button>
                )}
                <Button onClick={onNewAISession}>{t('overview.newAISession')}</Button>
                <Button onClick={onProjectShell}>{t('overview.projectShell')}</Button>
              </div>
            }
          />
        ) : (
          <div className="nx-agent-grid">
            {agentList.map((agent) => (
              <Card key={agent.id} className={`nx-agent-card ${styles.agentCardContent}`}>
                <div className="nx-agent-card__head">
                  <span className="nx-agent-avatar nx-agent-avatar--large">
                    {(agent.name || 'AG').slice(0, 2).toUpperCase()}
                  </span>
                  <div className={styles.agentInfo}>
                    <strong className={styles.agentName}>{agent.name || agent.id}</strong>
                    <small className={styles.agentSubtitle}>
                      {agent.role || t('overview.developmentAgent')} · {agent.id}
                    </small>
                  </div>
                  <Badge tone={tone(agent.status)}>
                    {agent.status === 'RECOVERABLE'
                      ? t('overview.runtimeStopped', 'Runtime parado')
                      : translateStatus(agent.status)}
                  </Badge>
                </div>

                <div className="nx-agent-card__meta">
                  <span>{t('agents.continuity')}</span>
                  <strong>{agent.continuity_status || t('common.unknown')}</strong>
                  <span>{t('agents.lastStart')}</span>
                  <strong>{agent.last_started_at || t('common.never')}</strong>
                </div>

                <div className={`nx-agent-card__actions ${styles.agentActions}`}>
                  <Button size="sm" tone="brand" onClick={() => onOpenAgent(agent)}>
                    <TerminalSquare size={13} /> {t('overview.openTerminal', 'Abrir Terminal')}
                  </Button>
                  {agent.status === 'RECOVERABLE' ? (
                    <Button
                      size="sm"
                      tone="warning"
                      disabled={busy === agent.id}
                      onClick={(e) => {
                        e.stopPropagation();
                        void handleRecover(agent);
                      }}
                    >
                      <RotateCcw size={12} />{' '}
                      {busy === agent.id
                        ? t('overview.recovering', 'Recuperando…')
                        : t('overview.recover', 'Recuperar')}
                    </Button>
                  ) : agent.status === 'STOPPED' ? (
                    <Button
                      size="sm"
                      tone="brand"
                      disabled={busy === agent.id}
                      onClick={(e) => {
                        e.stopPropagation();
                        void handleStart(agent);
                      }}
                    >
                      <Play size={12} />{' '}
                      {busy === agent.id
                        ? t('overview.starting', 'Iniciando…')
                        : t('overview.start', 'Iniciar')}
                    </Button>
                  ) : null}
                  {onConfigureAgent && (
                    <Button size="sm" tone="ghost" onClick={() => onConfigureAgent(agent)}>
                      {t('agents.configure', 'Configurar')}
                    </Button>
                  )}
                </div>
              </Card>
            ))}
          </div>
        )}
      </div>

      <Dialog
        open={!!resourceAgent}
        onClose={() => setResourceAgent(null)}
        title={resourceAgent ? `Select resource for ${resourceAgent.name}` : 'Select resource'}
        wide
      >
        <ResourcePicker agentId={resourceAgent?.id} onSelected={allocateAndStart} />
      </Dialog>
    </div>
  );
};
