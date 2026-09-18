import React, { useEffect, useMemo, useState } from 'react';
import { AlertTriangle, CheckCircle2, Save, SlidersHorizontal } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Card, EmptyState, Input, Select, Spinner } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, AgentConfig, ConfigImpact } from '../../types';

export const AgentConfigurationSurface: React.FC<{ agent?: Agent; onApplied?: () => void }> = ({
  agent,
  onApplied,
}) => {
  const { t } = useTranslation();

  const providers = useMemo(
    () =>
      ['', 'codex', 'claude', 'gemini', 'opencode', 'agy', 'cursor'].map((value) => ({
        value,
        label: value || t('agentConfig.automatic', 'Automatic'),
      })),
    [t],
  );

  const [config, setConfig] = useState<AgentConfig>({ provider: '', profile: 'default' });
  const [impact, setImpact] = useState<ConfigImpact | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!agent) {
      setLoading(false);
      return;
    }
    setLoading(true);
    nexus
      .getAgentConfig(agent.id)
      .then((data) => setConfig(data.config || { provider: '', profile: 'default' }))
      .catch((e) => setError(e instanceof Error ? e.message : String(e)))
      .finally(() => setLoading(false));
  }, [agent]);

  const update = <K extends keyof AgentConfig>(key: K, value: AgentConfig[K]) => {
    setConfig((current) => ({ ...current, [key]: value }));
    setImpact(null);
  };

  const preview = async () => {
    if (!agent) return;
    setBusy(true);
    try {
      const result = await nexus.previewAgentConfig(agent.id, config);
      setImpact(result.impact);
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const apply = async () => {
    if (!agent) return;
    setBusy(true);
    try {
      const result = await nexus.applyAgentConfig(agent.id, config);
      setImpact(result.impact);
      await onApplied?.();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  };

  const modeTone = useMemo(
    () =>
      impact?.mode === 'NEW_SESSION'
        ? 'warning'
        : impact?.mode === 'RESTART_RUNTIME'
          ? 'info'
          : 'success',
    [impact],
  );

  if (loading)
    return (
      <div className="nx-surface-center">
        <Spinner label={t('agentConfig.loading', 'Loading Agent configuration…')} />
      </div>
    );

  if (!agent)
    return (
      <div className="nx-surface-center">
        <EmptyState
          icon={<SlidersHorizontal size={22} />}
          title={t('agentConfig.unavailable', 'Agent unavailable')}
          hint={t('agentConfig.unavailableHint', 'The selected Agent no longer exists.')}
        />
      </div>
    );

  return (
    <div className="nx-surface-scroll">
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">{t('agentConfig.eyebrow', 'Agent configuration')}</span>
          <h1>{agent.name}</h1>
          <p>
            {t(
              'agentConfig.desc',
              'Configuration is versioned. Safe Apply previews continuity impact before the runtime changes.',
            )}
          </p>
        </div>
        <Badge>
          {agent.continuity_status || t('agentConfig.continuityUnknown', 'continuity unknown')}
        </Badge>
      </div>
      {error && (
        <Card className="nx-inline-error">
          <AlertTriangle size={15} />
          {error}
        </Card>
      )}
      <div className="nx-config-grid">
        <Card className="nx-config-card">
          <h3>{t('agentConfig.providerAndAccount', 'Provider & account')}</h3>
          <Select
            label={t('agentConfig.provider', 'Provider')}
            value={config.provider || ''}
            onChange={(value) => update('provider', value)}
            options={providers}
          />
          <label>
            {t('agentConfig.profile', 'Profile')}
            <Input
              value={config.profile || 'default'}
              onChange={(value) => update('profile', value)}
              mono
            />
          </label>
          <label>
            {t('agentConfig.model', 'Model')}
            <Input
              value={config.model || ''}
              onChange={(value) => update('model', value || undefined)}
              placeholder={t('agentConfig.providerDefault', 'Provider default')}
              mono
            />
          </label>
        </Card>
        <Card className="nx-config-card">
          <h3>{t('agentConfig.execution', 'Execution')}</h3>
          <label>
            {t('agentConfig.workspaceOverride', 'Workspace override')}
            <Input
              value={config.workspace || ''}
              onChange={(value) => update('workspace', value || undefined)}
              placeholder={t('agentConfig.inheritProjectPath', 'Inherit Project path')}
              mono
            />
          </label>
          <Select
            label={t('agentConfig.isolation', 'Isolation')}
            value={config.isolation || ''}
            onChange={(value) => update('isolation', value || undefined)}
            options={[
              { value: '', label: t('agentConfig.projectDefault', 'Project default') },
              { value: 'project', label: t('agentConfig.isolationProject', 'Project') },
              { value: 'worktree', label: t('agentConfig.isolationWorktree', 'Worktree') },
              { value: 'none', label: t('agentConfig.isolationNone', 'None') },
            ]}
          />
          <Select
            label={t('agentConfig.continuity', 'Continuity')}
            value={config.continuity_policy || ''}
            onChange={(value) => update('continuity_policy', value || undefined)}
            options={[
              { value: '', label: t('agentConfig.automatic', 'Automatic') },
              { value: 'native', label: t('agentConfig.continuityNative', 'Native resume only') },
              {
                value: 'new_session',
                label: t('agentConfig.continuityNewSession', 'Always new session'),
              },
            ]}
          />
        </Card>
        <Card className="nx-config-card">
          <h3>{t('agentConfig.maestroAndAllocation', 'Maestro & allocation')}</h3>
          <Select
            label={t('agentConfig.maestro', 'Maestro')}
            value={config.maestro_mode || ''}
            onChange={(value) => update('maestro_mode', value || undefined)}
            options={[
              { value: '', label: t('agentConfig.projectDefault', 'Project default') },
              { value: 'OFF', label: t('agentConfig.maestroOff', 'Off') },
              { value: 'ASSIST', label: t('agentConfig.maestroAssist', 'Assist') },
              { value: 'ORCHESTRATE', label: t('agentConfig.maestroOrchestrate', 'Orchestrate') },
            ]}
          />
          <Select
            label={t('agentConfig.preferProvider', 'Prefer provider')}
            value={config.allocation?.prefer_provider || ''}
            onChange={(value) =>
              update('allocation', { ...config.allocation, prefer_provider: value || undefined })
            }
            options={providers}
          />
          <label>
            {t('agentConfig.maxConcurrent', 'Max concurrent')}
            <Input
              value={String(config.allocation?.max_concurrent || '')}
              onChange={(value) =>
                update('allocation', {
                  ...config.allocation,
                  max_concurrent: value ? Number(value) : undefined,
                })
              }
            />
          </label>
        </Card>
      </div>
      {impact && (
        <Card className="nx-impact-card">
          <div>
            {modeTone === 'success' ? <CheckCircle2 size={16} /> : <AlertTriangle size={16} />}
            <Badge tone={modeTone}>{impact.mode}</Badge>
          </div>
          <strong>
            {impact.requires_new_session
              ? t('agentConfig.newSessionRequired', 'New provider session required')
              : impact.requires_restart
                ? t('agentConfig.restartRequired', 'Runtime restart required')
                : t('agentConfig.canApplyLive', 'Can be applied live')}
          </strong>
          <p>
            {(impact.changed_fields || []).length
              ? t('agentConfig.changed', {
                  fields: (impact.changed_fields || []).join(', '),
                  defaultValue: `Changed: ${(impact.changed_fields || []).join(', ')}`,
                })
              : t('agentConfig.noDifferences', 'No configuration differences detected.')}
          </p>
          {(impact.warnings || []).map((warning) => (
            <small key={warning}>{warning}</small>
          ))}
        </Card>
      )}
      <div className="nx-config-actions">
        <Button onClick={preview} disabled={busy}>
          {t('agentConfig.previewImpact', 'Preview impact')}
        </Button>
        <Button tone="brand" onClick={apply} disabled={busy}>
          <Save size={14} /> {t('agentConfig.safeApply', 'Safe Apply')}
        </Button>
      </div>
    </div>
  );
};
