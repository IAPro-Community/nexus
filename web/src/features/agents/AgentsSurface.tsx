import React, { useMemo, useState } from 'react';
import {
  Bot,
  CheckCircle2,
  Clock,
  Code2,
  Compass,
  MessageSquareText,
  Play,
  Plus,
  RefreshCw,
  RotateCcw,
  Settings2,
  Sparkles,
  Square,
  TerminalSquare,
  Trash2,
} from 'lucide-react';
import { Badge, Button, Card, ConfirmDialog, Dialog, EmptyState, Input } from '../../design-system';
import { nexus } from '../../nexus/api';
import { ResourcePicker } from '../../nexus/ResourcePicker';
import type { Agent, Project } from '../../types';
import { translateStatus } from '../../i18n';
import { useTranslation } from 'react-i18next';
import { TaskPreparationDialog } from '../../components/TaskPreparationDialog';
import { askActionForStatus } from './askAgentModel';
import { formatOpenedAgo } from '../projects/projectRailRelativeTime';
import styles from './AgentsSurface.module.scss';

const tone = (status: string) =>
  status === 'WORKING'
    ? 'success'
    : status === 'FAILED' || status === 'STALE'
      ? 'danger'
      : status === 'RECOVERABLE' || status === 'WAITING' || status === 'RATE_LIMITED'
        ? 'warning'
        : 'default';

const ROLE_TEMPLATES = [
  { key: 'developer', labelKey: 'agents.roleTemplateDeveloper', icon: Code2 },
  { key: 'architect', labelKey: 'agents.roleTemplateArchitect', icon: Compass },
  { key: 'reviewer', labelKey: 'agents.roleTemplateReviewer', icon: CheckCircle2 },
  { key: 'tester', labelKey: 'agents.roleTemplateTester', icon: Sparkles },
];

export const AgentsSurface: React.FC<{
  project: Project;
  agents: Agent[];
  refresh: () => Promise<void>;
  onTerminal: (agent: Agent) => void;
  onConfigure: (agent: Agent) => void;
  onRemoved?: (agentId: string) => void;
}> = ({ project, agents, refresh, onTerminal, onConfigure, onRemoved }) => {
  const { t, i18n } = useTranslation();
  const [name, setName] = useState('');
  const [selectedRole, setSelectedRole] = useState('developer');
  const [busy, setBusy] = useState('');
  const [resourceAgent, setResourceAgent] = useState<Agent | null>(null);
  const [askAgent, setAskAgent] = useState<Agent | null>(null);
  const [agentToDelete, setAgentToDelete] = useState<Agent | null>(null);
  const [error, setError] = useState('');
  const [statusFilter, setStatusFilter] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const create = async () => {
    if (!name.trim()) return;
    setBusy('create');
    try {
      await nexus.createAgent(project.id, name.trim(), selectedRole);
      setName('');
      await refresh();
    } finally {
      setBusy('');
    }
  };

  const action = async (agent: Agent, kind: 'start' | 'stop' | 'recover') => {
    setBusy(agent.id);
    setError('');
    try {
      if (kind === 'start') await nexus.startAgent(agent.id);
      else if (kind === 'stop') await nexus.stopAgent(agent.id);
      else await nexus.recoverAgent(agent.id);
      await refresh();
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (kind === 'start' && message.includes('REQUIRED_RESOURCE_SELECTION'))
        setResourceAgent(agent);
      else setError(message);
    } finally {
      setBusy('');
    }
  };

  const remove = (agent: Agent) => {
    setAgentToDelete(agent);
  };

  const executeRemove = async () => {
    if (!agentToDelete) return;
    const agent = agentToDelete;
    setAgentToDelete(null);
    setBusy(agent.id);
    setError('');
    try {
      try {
        await nexus.stopAgent(agent.id);
      } catch {
        /* may already be dead */
      }
      await nexus.deleteAgent(agent.id);
      onRemoved?.(agent.id);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy('');
    }
  };

  const allocateAndStart = async () => {
    if (!resourceAgent) return;
    setBusy(resourceAgent.id);
    setError('');
    try {
      await nexus.startAgent(resourceAgent.id);
      await refresh();
      setResourceAgent(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy('');
    }
  };

  // Metrics calculation
  const totalCount = agents.length;
  const workingCount = agents.filter((a) => a.status === 'WORKING').length;
  const waitingCount = agents.filter((a) => ['WAITING', 'IDLE'].includes(a.status)).length;
  const degradedCount = agents.filter((a) =>
    ['FAILED', 'STALE', 'RECOVERABLE', 'RATE_LIMITED'].includes(a.status),
  ).length;
  const stoppedCount = agents.filter((a) => a.status === 'STOPPED').length;

  // Filtered list
  const filteredAgents = useMemo(() => {
    return agents.filter((agent) => {
      // Status filter
      if (statusFilter === 'working' && agent.status !== 'WORKING') return false;
      if (statusFilter === 'waiting' && !['WAITING', 'IDLE'].includes(agent.status)) return false;
      if (
        statusFilter === 'degraded' &&
        !['FAILED', 'STALE', 'RECOVERABLE', 'RATE_LIMITED'].includes(agent.status)
      )
        return false;
      if (statusFilter === 'stopped' && agent.status !== 'STOPPED') return false;

      // Text query
      if (!searchQuery.trim()) return true;
      const q = searchQuery.toLowerCase();
      return (
        agent.name.toLowerCase().includes(q) ||
        agent.id.toLowerCase().includes(q) ||
        (agent.role && agent.role.toLowerCase().includes(q))
      );
    });
  }, [agents, statusFilter, searchQuery]);

  return (
    <div className="nx-surface-scroll">
      <div className={styles.agentsContainer}>
        {/* Page Header */}
        <div className="nx-page-header">
          <div>
            <span className="nx-eyebrow">
              {t('agents.eyebrow', 'Workspace OS · Frota Autônoma')}
            </span>
            <h1>{t('agents.title', 'Agentes Persistentes')}</h1>
            <p>
              {t(
                'agents.intro',
                'Agentes persistentes especializados com continuidade nativa de sessão, PTY e binding de provedores.',
              )}
            </p>
          </div>
          <Button size="sm" tone="ghost" onClick={() => void refresh()}>
            <RefreshCw size={13} /> {t('common.refresh', 'Atualizar')}
          </Button>
        </div>

        {/* Top Summary Metrics Strip */}
        <div className={styles.summaryStrip}>
          <button
            type="button"
            className={`${styles.summaryCard} ${statusFilter === 'all' ? styles.activeFilter : ''}`}
            aria-pressed={statusFilter === 'all'}
            onClick={() => setStatusFilter('all')}
          >
            <div className={styles.summaryValue}>{totalCount}</div>
            <div className={styles.summaryLabel}>{t('agents.metricTotal')}</div>
          </button>

          <button
            type="button"
            className={`${styles.summaryCard} ${statusFilter === 'working' ? styles.activeFilter : ''}`}
            aria-pressed={statusFilter === 'working'}
            onClick={() => setStatusFilter(statusFilter === 'working' ? 'all' : 'working')}
          >
            <div className={`${styles.summaryValue} ${styles.summaryValueSuccess}`}>
              {workingCount}
            </div>
            <div className={styles.summaryLabel}>
              {workingCount > 0 && <span className={styles.pulseDot} />}
              {t('agents.metricWorking')}
            </div>
          </button>

          <button
            type="button"
            className={`${styles.summaryCard} ${statusFilter === 'waiting' ? styles.activeFilter : ''}`}
            aria-pressed={statusFilter === 'waiting'}
            onClick={() => setStatusFilter(statusFilter === 'waiting' ? 'all' : 'waiting')}
          >
            <div className={`${styles.summaryValue} ${styles.summaryValueAccent}`}>
              {waitingCount}
            </div>
            <div className={styles.summaryLabel}>{t('agents.metricWaiting')}</div>
          </button>

          <button
            type="button"
            className={`${styles.summaryCard} ${statusFilter === 'degraded' ? styles.activeFilter : ''}`}
            aria-pressed={statusFilter === 'degraded'}
            onClick={() => setStatusFilter(statusFilter === 'degraded' ? 'all' : 'degraded')}
          >
            <div
              className={`${styles.summaryValue} ${degradedCount > 0 ? styles.summaryValueWarning : ''}`}
            >
              {degradedCount}
            </div>
            <div className={styles.summaryLabel}>{t('agents.metricDegraded')}</div>
          </button>

          <button
            type="button"
            className={`${styles.summaryCard} ${statusFilter === 'stopped' ? styles.activeFilter : ''}`}
            aria-pressed={statusFilter === 'stopped'}
            onClick={() => setStatusFilter(statusFilter === 'stopped' ? 'all' : 'stopped')}
          >
            <div className={styles.summaryValue}>{stoppedCount}</div>
            <div className={styles.summaryLabel}>{t('agents.metricStopped')}</div>
          </button>
        </div>

        {/* Quick Creation Section */}
        <div className={styles.creationSection}>
          <div className={styles.createFormRow}>
            <div className={styles.createNameField}>
              <Input
                value={name}
                onChange={setName}
                onEnter={create}
                placeholder={t(
                  'agents.namePlaceholder',
                  'Nome do novo agente (ex: frontend-lead, reviewer-core)…',
                )}
              />
            </div>
            <Button tone="brand" disabled={!name.trim() || busy === 'create'} onClick={create}>
              <Plus size={14} /> {t('agents.create', 'Criar Agente')}
            </Button>
          </div>

          <div className={styles.roleTemplates}>
            <span className={styles.roleLabel}>{t('agents.selectRole')}</span>
            {ROLE_TEMPLATES.map((tpl) => {
              const Icon = tpl.icon;
              return (
                <button
                  key={tpl.key}
                  type="button"
                  className={`${styles.templatePill} ${selectedRole === tpl.key ? styles.activeRole : ''}`}
                  onClick={() => setSelectedRole(tpl.key)}
                >
                  <Icon size={12} />
                  {t(tpl.labelKey)}
                </button>
              );
            })}
          </div>
        </div>

        {/* Search Control Bar */}
        <div className={styles.controlBar}>
          <div className={styles.searchBox}>
            <Input
              value={searchQuery}
              onChange={setSearchQuery}
              placeholder={t('agents.searchPlaceholder', 'Buscar agentes por nome ou papel…')}
            />
          </div>
          <div className={styles.displayedCount}>
            {t('agents.displayedCount', {
              filtered: filteredAgents.length,
              total: agents.length,
            })}
          </div>
        </div>

        {error && <Card className="nx-inline-error">{error}</Card>}

        {/* Agent Cards Grid */}
        {filteredAgents.length === 0 ? (
          <EmptyState
            icon={<Bot size={32} />}
            title={t('agents.empty', 'Nenhum agente encontrado')}
            hint={t('agents.emptyHint', 'Crie um novo agente ou redefina os filtros de busca.')}
          />
        ) : (
          <div className={styles.agentGrid}>
            {filteredAgents.map((agent) => {
              const isWorking = agent.status === 'WORKING';
              const isRecoverable = agent.status === 'RECOVERABLE';

              return (
                <div key={agent.id} className={styles.agentCard}>
                  {/* Card Head */}
                  <div className={styles.cardHead}>
                    <div className={styles.agentAvatar}>
                      {(agent.name || 'AG').slice(0, 2).toUpperCase()}
                    </div>
                    <div className={styles.agentHeadInfo}>
                      <div className={styles.agentName} title={agent.name || agent.id}>
                        {agent.name || agent.id}
                      </div>
                      <div className={styles.agentIdSub}>{agent.id}</div>
                    </div>
                    <div className={styles.statusWrapper}>
                      {isWorking && <span className={styles.pulseDot} />}
                      <Badge tone={tone(agent.status)}>{translateStatus(agent.status)}</Badge>
                    </div>
                  </div>

                  {/* Meta Box */}
                  <div className={styles.metaSection}>
                    <div className={styles.metaRow}>
                      <span className={styles.metaLabel}>{t('agents.role', 'Papel')}</span>
                      <span className={styles.metaValue}>
                        <Badge tone="default">{agent.role || 'developer'}</Badge>
                      </span>
                    </div>

                    <div className={styles.metaRow}>
                      <span className={styles.metaLabel}>
                        {t('agents.continuity', 'Continuidade')}
                      </span>
                      <span className={styles.metaValue}>
                        <Badge
                          tone={agent.continuity_status === 'CONTINUOUS' ? 'success' : 'default'}
                        >
                          {agent.continuity_status || 'UNKNOWN'}
                        </Badge>
                      </span>
                    </div>

                    <div className={styles.metaRow}>
                      <span className={styles.metaLabel}>
                        {t('agents.lastStart', 'Última Execução')}
                      </span>
                      <span className={styles.metaValue} title={agent.last_started_at || ''}>
                        <Clock size={11} className={styles.metaClock} />
                        {formatOpenedAgo(agent.last_started_at, i18n.language, t('common.never'))}
                      </span>
                    </div>
                  </div>

                  {/* Actions Toolbar */}
                  <div className={styles.cardActions}>
                    <Button
                      size="sm"
                      tone="brand"
                      onClick={() => onTerminal(agent)}
                      title={t('agents.terminal', 'Terminal')}
                    >
                      <TerminalSquare size={13} /> {t('agents.terminal', 'Terminal')}
                    </Button>

                    <Button
                      size="sm"
                      onClick={() => setAskAgent(agent)}
                      title={t('agents.prepareTask', 'Preparar Tarefa')}
                    >
                      <MessageSquareText size={13} /> {t('agents.taskShort', 'Tarefa')}
                    </Button>

                    {/* Contextual lifecycle action */}
                    {isWorking ? (
                      <Button
                        size="sm"
                        tone="danger"
                        disabled={busy === agent.id}
                        onClick={() => void action(agent, 'stop')}
                      >
                        <Square size={11} /> {t('agents.stop', 'Parar')}
                      </Button>
                    ) : isRecoverable ? (
                      <Button
                        size="sm"
                        tone="warning"
                        disabled={busy === agent.id}
                        onClick={() => void action(agent, 'recover')}
                      >
                        <RotateCcw size={12} /> {t('agents.recover', 'Recuperar')}
                      </Button>
                    ) : (
                      <Button
                        size="sm"
                        tone="brand"
                        disabled={busy === agent.id}
                        onClick={() => void action(agent, 'start')}
                      >
                        <Play size={11} /> {t('agents.start', 'Iniciar')}
                      </Button>
                    )}

                    {/* Configure Button */}
                    <Button
                      size="sm"
                      tone="ghost"
                      onClick={() => onConfigure(agent)}
                      title={t('agents.configure', 'Configurar')}
                    >
                      <Settings2 size={13} />
                    </Button>

                    {/* Delete Action */}
                    <Button
                      size="sm"
                      tone="danger"
                      disabled={busy === agent.id}
                      onClick={() => remove(agent)}
                      title={t('agents.remove', 'Remover')}
                    >
                      <Trash2 size={13} />
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        )}

        {/* Dialogs */}
        <TaskPreparationDialog
          open={Boolean(askAgent)}
          agentId={askAgent?.id || ''}
          agentName={askAgent?.name}
          projectId={askAgent?.project_id}
          startIfNeeded={askActionForStatus(askAgent?.status || 'STOPPED').startIfNeeded}
          onClose={() => setAskAgent(null)}
          onSubmit={async (prompt, skills, startIfNeeded, contextFingerprintId) => {
            if (!askAgent) return;
            await nexus.askAgent(askAgent.id, prompt, startIfNeeded, skills, {
              projectId: askAgent.project_id,
              contextFingerprintId: contextFingerprintId || '',
            });
            await refresh();
            onTerminal(askAgent);
          }}
        />

        <Dialog
          open={!!resourceAgent}
          onClose={() => setResourceAgent(null)}
          title={
            resourceAgent ? `Selecionar recurso para ${resourceAgent.name}` : 'Selecionar recurso'
          }
          wide
        >
          <ResourcePicker agentId={resourceAgent?.id} onSelected={allocateAndStart} />
        </Dialog>

        <ConfirmDialog
          open={Boolean(agentToDelete)}
          title={t('agents.confirmRemoveTitle', 'Remover Agente')}
          description={t('agents.confirmRemove', {
            name: agentToDelete?.name || agentToDelete?.id,
          })}
          onConfirm={executeRemove}
          onCancel={() => setAgentToDelete(null)}
        />
      </div>
    </div>
  );
};
