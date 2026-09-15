import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Activity,
  AlertTriangle,
  ArrowRightLeft,
  Clock,
  GitBranch,
  History,
  Layers,
  Play,
  RefreshCw,
  Terminal,
} from 'lucide-react';
import { Badge, Card, EmptyState, Input, Spinner } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, AgentDetail, TelemetryHistoryEvent, TelemetryStats } from '../../types';
import { useTranslation } from 'react-i18next';
import styles from './SessionsSurface.module.scss';

function formatRelativeTime(iso: string): string {
  if (!iso) return '';
  const now = Date.now();
  const d = new Date(iso).getTime();
  if (isNaN(d)) return iso;
  const diffSec = Math.max(0, Math.floor((now - d) / 1000));
  if (diffSec < 60) return `${diffSec}s atrás`;
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return `${diffMin}m atrás`;
  const diffHours = Math.floor(diffMin / 60);
  if (diffHours < 24) return `${diffHours}h atrás`;
  const diffDays = Math.floor(diffHours / 24);
  return `${diffDays}d atrás`;
}

function formatDuration(ms?: number): string {
  if (!ms || ms <= 0) return '';
  if (ms < 1000) return `${ms}ms`;
  const sec = (ms / 1000).toFixed(1);
  return `${sec}s`;
}

function eventTone(type: string): 'start' | 'rate_limit' | 'fallback' | 'quota' | 'default' {
  switch (type) {
    case 'SESSION_STARTED':
      return 'start';
    case 'RATE_LIMIT_DETECTED':
      return 'rate_limit';
    case 'FALLBACK_TRIGGERED':
      return 'fallback';
    case 'QUOTA_REFRESHED':
      return 'quota';
    default:
      return 'default';
  }
}

function eventIcon(type: string) {
  switch (type) {
    case 'SESSION_STARTED':
      return <Play size={16} />;
    case 'RATE_LIMIT_DETECTED':
      return <AlertTriangle size={16} />;
    case 'FALLBACK_TRIGGERED':
      return <ArrowRightLeft size={16} />;
    case 'QUOTA_REFRESHED':
      return <RefreshCw size={16} />;
    default:
      return <Activity size={16} />;
  }
}

export const SessionsSurface: React.FC<{ agents: Agent[] }> = ({ agents }) => {
  const { t } = useTranslation();
  const [activeTab, setActiveTab] = useState<'timeline' | 'generations'>('timeline');
  const [stats, setStats] = useState<TelemetryStats | null>(null);
  const [history, setHistory] = useState<TelemetryHistoryEvent[]>([]);
  const [details, setDetails] = useState<AgentDetail[]>([]);
  const [loading, setLoading] = useState(true);
  const [selectedProvider, setSelectedProvider] = useState<string>('all');
  const [searchQuery, setSearchQuery] = useState('');

  const loadData = useCallback(async () => {
    setLoading(true);
    try {
      const [statsData, historyData, agentDetails] = await Promise.all([
        nexus.getTelemetryStats(7).catch(() => null),
        nexus.getTelemetryHistory(100).catch(() => []),
        Promise.all(agents.map((agent) => nexus.getAgent(agent.id).catch(() => null))),
      ]);
      setStats(statsData);
      setHistory(Array.isArray(historyData) ? historyData : []);
      setDetails(agentDetails.filter(Boolean) as AgentDetail[]);
    } finally {
      setLoading(false);
    }
  }, [agents]);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  // Providers present in events for quick filter
  const availableProviders = useMemo(() => {
    const set = new Set<string>();
    history.forEach((e) => {
      if (e.provider_id) set.add(e.provider_id);
    });
    if (stats?.by_provider) {
      Object.keys(stats.by_provider).forEach((p) => set.add(p));
    }
    return Array.from(set).sort();
  }, [history, stats]);

  // Filtered timeline events
  const filteredEvents = useMemo(() => {
    return history.filter((ev) => {
      if (selectedProvider !== 'all' && ev.provider_id !== selectedProvider) {
        return false;
      }
      if (!searchQuery.trim()) return true;
      const q = searchQuery.toLowerCase();
      return (
        ev.type.toLowerCase().includes(q) ||
        ev.provider_id.toLowerCase().includes(q) ||
        ev.profile_id.toLowerCase().includes(q) ||
        (ev.workspace && ev.workspace.toLowerCase().includes(q)) ||
        (ev.reason && ev.reason.toLowerCase().includes(q)) ||
        (ev.failure_kind && ev.failure_kind.toLowerCase().includes(q))
      );
    });
  }, [history, selectedProvider, searchQuery]);

  // Filtered agent generations
  const filteredAgentDetails = useMemo(() => {
    return details.filter((detail) => {
      if (!searchQuery.trim()) return true;
      const q = searchQuery.toLowerCase();
      return (
        detail.agent.name.toLowerCase().includes(q) ||
        detail.agent.id.toLowerCase().includes(q) ||
        detail.generations?.some(
          (g) =>
            g.provider.toLowerCase().includes(q) ||
            g.profile.toLowerCase().includes(q) ||
            g.runtime_id.toLowerCase().includes(q),
        )
      );
    });
  }, [details, searchQuery]);

  if (loading && !stats && history.length === 0) {
    return (
      <div className="nx-surface-center">
        <Spinner label={t('sessions.loading', 'Carregando histórico e telemetria…')} />
      </div>
    );
  }

  return (
    <div className="nx-surface-scroll">
      <div className={styles.sessionsContainer}>
        {/* Page Header */}
        <div className="nx-page-header">
          <div>
            <span className="nx-eyebrow">
              {t('sessions.eyebrow', 'Workspace OS · Auditoria e Telemetria')}
            </span>
            <h1>{t('sessions.title', 'Sessões e Histórico')}</h1>
            <p>
              {t(
                'sessions.intro',
                'Registro unificado de execuções de IA, fallbacks, limites de cota e continuidade de agentes.',
              )}
            </p>
          </div>
          <button
            type="button"
            className={styles.tabButton}
            onClick={() => void loadData()}
            title={t('common.refresh', 'Atualizar')}
          >
            <RefreshCw size={14} />
            {t('common.refresh', 'Atualizar')}
          </button>
        </div>

        {/* Top Metrics Grid */}
        <div className={styles.metricsGrid}>
          <div className={styles.metricCard}>
            <div className={styles.metricValue}>{stats?.total_sessions ?? history.length}</div>
            <div className={styles.metricLabel}>
              {t('sessions.metricTotalSessions', 'Sessões Totais (7d)')}
            </div>
            <Terminal size={40} className={styles.metricIcon} />
          </div>

          <div className={styles.metricCard}>
            <div className={styles.metricValue} style={{ color: '#f59e0b' }}>
              {stats?.total_fallbacks ?? 0}
            </div>
            <div className={styles.metricLabel}>
              {t('sessions.metricFallbacks', 'Fallbacks Automáticos')}
            </div>
            <ArrowRightLeft size={40} className={styles.metricIcon} />
          </div>

          <div className={styles.metricCard}>
            <div
              className={styles.metricValue}
              style={{ color: (stats?.total_rate_limits ?? 0) > 0 ? '#ef4444' : 'inherit' }}
            >
              {stats?.total_rate_limits ?? 0}
            </div>
            <div className={styles.metricLabel}>
              {t('sessions.metricRateLimits', 'Rate Limits Registrados')}
            </div>
            <AlertTriangle size={40} className={styles.metricIcon} />
          </div>

          <div className={styles.metricCard}>
            <div className={styles.metricValue}>{details.length}</div>
            <div className={styles.metricLabel}>
              {t('sessions.metricActiveAgents', 'Agentes Persistentes')}
            </div>
            <Layers size={40} className={styles.metricIcon} />
          </div>
        </div>

        {/* Control Bar: Tabs, Search and Provider Filter */}
        <div className={styles.controlBar}>
          <div className={styles.tabGroup}>
            <button
              type="button"
              className={`${styles.tabButton} ${activeTab === 'timeline' ? styles.activeTab : ''}`}
              onClick={() => setActiveTab('timeline')}
            >
              <History size={15} />
              {t('sessions.tabTimeline', 'Histórico de Execuções')}
              <Badge tone="default">{filteredEvents.length}</Badge>
            </button>
            <button
              type="button"
              className={`${styles.tabButton} ${activeTab === 'generations' ? styles.activeTab : ''}`}
              onClick={() => setActiveTab('generations')}
            >
              <GitBranch size={15} />
              {t('sessions.tabGenerations', 'Gerações dos Agentes')}
              <Badge tone="default">{details.length}</Badge>
            </button>
          </div>

          <div className={styles.searchAndFilters}>
            <Input
              value={searchQuery}
              onChange={setSearchQuery}
              placeholder={t('sessions.searchPlaceholder', 'Buscar por tipo, provedor, perfil…')}
            />
          </div>
        </div>

        {/* Provider Quick Pills (active in timeline view) */}
        {activeTab === 'timeline' && availableProviders.length > 0 && (
          <div className={styles.providerPills}>
            <button
              type="button"
              className={`${styles.providerPill} ${selectedProvider === 'all' ? styles.activePill : ''}`}
              onClick={() => setSelectedProvider('all')}
            >
              {t('common.all', 'Todos')} ({history.length})
            </button>
            {availableProviders.map((prov) => {
              const count = history.filter((e) => e.provider_id === prov).length;
              return (
                <button
                  key={prov}
                  type="button"
                  className={`${styles.providerPill} ${selectedProvider === prov ? styles.activePill : ''}`}
                  onClick={() => setSelectedProvider(prov)}
                >
                  {prov.toUpperCase()} ({count})
                </button>
              );
            })}
          </div>
        )}

        {/* Tab 1: Execution History Timeline */}
        {activeTab === 'timeline' && (
          <>
            {filteredEvents.length === 0 ? (
              <EmptyState
                icon={<History size={24} />}
                title={t('sessions.emptyTimeline', 'Nenhum evento registrado')}
                hint={t(
                  'sessions.emptyTimelineHint',
                  'Sessões de CLI e intervenções aparecerão aqui conforme forem executadas.',
                )}
              />
            ) : (
              <div className={styles.timelineList}>
                {filteredEvents.map((ev, index) => {
                  const tone = eventTone(ev.type);
                  const icon = eventIcon(ev.type);
                  return (
                    <div key={`${ev.timestamp}-${index}`} className={styles.timelineItem}>
                      <div className={styles.eventIconWrapper} data-tone={tone}>
                        {icon}
                      </div>
                      <div className={styles.eventContent}>
                        <div className={styles.eventHeader}>
                          <span className={styles.eventTitle}>{ev.type}</span>
                          <span className={styles.eventTarget}>
                            {ev.provider_id}
                            {ev.profile_id ? `:${ev.profile_id}` : ''}
                          </span>
                          {ev.duration_ms ? (
                            <Badge tone="default">
                              <Clock size={11} style={{ marginRight: 3 }} />
                              {formatDuration(ev.duration_ms)}
                            </Badge>
                          ) : null}
                          {ev.failure_kind && <Badge tone="danger">{ev.failure_kind}</Badge>}
                          {ev.fallback_profile && (
                            <Badge tone="warning">➔ {ev.fallback_profile}</Badge>
                          )}
                          <span className={styles.eventTime} title={ev.timestamp}>
                            {formatRelativeTime(ev.timestamp)}
                          </span>
                        </div>

                        <div className={styles.eventMeta}>
                          {ev.workspace && (
                            <span className={styles.eventWorkspace} title={ev.workspace}>
                              {ev.workspace}
                            </span>
                          )}
                          {ev.reason && <span>{ev.reason}</span>}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </>
        )}

        {/* Tab 2: Agent Generations */}
        {activeTab === 'generations' && (
          <>
            {filteredAgentDetails.length === 0 ? (
              <EmptyState
                icon={<GitBranch size={24} />}
                title={t('sessions.empty', 'Nenhuma geração de agente encontrada')}
                hint={t('sessions.emptyHint', 'Crie ou execute agentes para visualizar linhagens.')}
              />
            ) : (
              <div className={styles.timelineList}>
                {filteredAgentDetails.map((detail) => {
                  const generations = Array.isArray(detail.generations) ? detail.generations : [];
                  return (
                    <Card key={detail.agent.id} className={styles.timelineItem}>
                      <div className={styles.eventContent}>
                        <div className={styles.agentCardHead}>
                          <div>
                            <strong style={{ fontSize: '0.95rem' }}>{detail.agent.name}</strong>
                            <small
                              style={{
                                color: 'var(--nx-text-muted)',
                                marginLeft: '0.5rem',
                                fontFamily: 'var(--nx-font-mono, monospace)',
                              }}
                            >
                              {detail.agent.id}
                            </small>
                          </div>
                          <Badge
                            tone={
                              detail.agent.continuity_status === 'CONTINUOUS'
                                ? 'success'
                                : 'default'
                            }
                          >
                            {detail.agent.continuity_status || 'UNKNOWN'}
                          </Badge>
                        </div>

                        <div className={styles.generationList}>
                          {generations.length === 0 ? (
                            <span style={{ fontSize: '0.8125rem', color: 'var(--nx-text-muted)' }}>
                              {t(
                                'sessions.noGenerations',
                                'Nenhuma geração registrada para este agente.',
                              )}
                            </span>
                          ) : (
                            generations.map((gen) => (
                              <div key={gen.id} className={styles.generationRow}>
                                <GitBranch size={14} style={{ color: 'var(--nx-accent-text)' }} />
                                <div style={{ flex: 1, minWidth: 0 }}>
                                  <strong>
                                    {gen.provider}/{gen.profile}
                                  </strong>
                                  <div
                                    style={{
                                      fontSize: '0.75rem',
                                      color: 'var(--nx-text-muted)',
                                      fontFamily: 'var(--nx-font-mono, monospace)',
                                    }}
                                  >
                                    {gen.runtime_id} ·{' '}
                                    {gen.provider_session ||
                                      t('sessions.unknown', 'Sessão desconhecida')}
                                  </div>
                                </div>
                                <Badge tone="default">{gen.state}</Badge>
                                <Badge
                                  tone={
                                    gen.continuity === 'NATIVE_RESUME_VERIFIED'
                                      ? 'success'
                                      : 'default'
                                  }
                                >
                                  {gen.continuity}
                                </Badge>
                              </div>
                            ))
                          )}
                        </div>
                      </div>
                    </Card>
                  );
                })}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
};
