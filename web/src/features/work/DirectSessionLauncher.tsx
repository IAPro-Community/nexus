import React, { useEffect, useMemo, useRef, useState } from 'react';
import { Bot, Play, ShieldAlert, TerminalSquare } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import {
  Badge,
  Button,
  ChoiceGroup,
  ChoiceOption,
  Dialog,
  EmptyState,
  Input,
  InlineAlert,
  Spinner,
  Select,
} from '../../design-system';
import { nexus } from '../../nexus/api';
import type { Agent, Project, RecommendationResult } from '../../types';
import {
  buildDirectAgentName,
  directAccountTitle,
  directQuotaDisplay,
  eligibleDirectResources,
  quotaTruthState,
  recommendedDirectAccountId,
  directSchedulerContext,
} from './directSessionModel';
import resourceStyles from '../../nexus/ResourcePicker.module.scss';
import contextualStyles from './ContextualResourceSelection.module.scss';
import { DirectSessionSetupError, startDirectSession } from './directSessionLifecycle';

type DirectTaskKind = 'coding' | 'planning' | 'review' | 'security';
const taskRole: Record<DirectTaskKind, string> = {
  coding: 'implementer',
  planning: 'architect',
  review: 'reviewer',
  security: 'reviewer',
};
const modelIdValid = (value: string) =>
  !value || (value.length <= 128 && /^[\w./:+-]+$/.test(value));

interface DirectResource {
  id: string;
  provider: string;
  profile: string;
  display_name?: string;
  authenticated: boolean;
  available: boolean;
  health?: string;
  rate_limited?: boolean;
  quota_remaining?: number;
  avail_reasons?: { unknown_quota?: boolean; exhausted_windows?: string[]; rate_limited?: boolean };
  quota_view?: {
    status?: string;
    fetched_at?: string;
    model_groups?: Array<{
      key?: string;
      name?: string;
      windows?: Array<{ kind?: string; remaining?: number }>;
    }>;
  };
}

export interface DirectSessionRequest {
  mode: 'direct' | 'assisted';
  prompt: string;
}

export const DirectSessionLauncher: React.FC<{
  open: boolean;
  project: Project;
  request: DirectSessionRequest | null;
  onClose: () => void;
  onStarted: (agent: Agent, prompt: string) => void | Promise<void>;
  refreshAgents: () => Promise<void>;
}> = ({ open, project, request, onClose, onStarted, refreshAgents }) => {
  const { t } = useTranslation();
  const [resources, setResources] = useState<DirectResource[]>([]);
  const [selectedID, setSelectedID] = useState('');
  const [name, setName] = useState('');
  const [taskKind, setTaskKind] = useState<DirectTaskKind>('coding');
  const [modelOverride, setModelOverride] = useState('');
  const [preferredProvider, setPreferredProvider] = useState(
    () => directSchedulerContext(project.resource_policy).preferProvider,
  );
  const [recommendation, setRecommendation] = useState<RecommendationResult | null>(null);
  const [recommendationError, setRecommendationError] = useState(false);
  const manualChoice = useRef(false);
  const submitInFlight = useRef(false);
  const [loading, setLoading] = useState(false);
  const [starting, setStarting] = useState(false);
  const [error, setError] = useState('');
  const [needsAgentRecovery, setNeedsAgentRecovery] = useState(false);

  useEffect(() => {
    if (!open) return;
    manualChoice.current = false;
    setSelectedID('');
    setNeedsAgentRecovery(false);
    setModelOverride('');
    setPreferredProvider(directSchedulerContext(project.resource_policy).preferProvider);
    setName('');
    setError('');
  }, [open, project.id, project.resource_policy]);

  useEffect(() => {
    if (!open) return;
    let mounted = true;
    setLoading(true);
    // Preserve a recoverable start error: changing filters must not hide the
    // identity of an Agent whose runtime might already be alive.
    setRecommendation(null);
    setRecommendationError(false);
    const context = directSchedulerContext(project.resource_policy, preferredProvider);
    void Promise.allSettled([
      nexus.listResources(),
      nexus.recommendResources(
        {
          task_kind: taskKind,
          role: taskRole[taskKind],
          ...(taskKind === 'security' ? { domains: ['security'] } : {}),
          ...(context.preferProvider ? { prefer_provider: context.preferProvider } : {}),
          source: 'direct_session_launcher',
        },
        context.policy,
      ),
    ]).then(([resourcesResult, recommendationResult]) => {
      if (!mounted) return;
      if (resourcesResult.status !== 'fulfilled') {
        setRecommendationError(true);
        setError(String(resourcesResult.reason));
        setResources([]);
        setSelectedID('');
      } else {
        const eligible = eligibleDirectResources(
          (resourcesResult.value.accounts || []) as DirectResource[],
        );
        setResources(eligible);
        const ranked =
          recommendationResult.status === 'fulfilled' ? recommendationResult.value : null;
        const suggested = recommendedDirectAccountId(eligible, ranked);
        setRecommendation(ranked);
        setRecommendationError(recommendationResult.status !== 'fulfilled');
        setSelectedID((previous) =>
          manualChoice.current && eligible.some((account) => account.id === previous)
            ? previous
            : suggested,
        );
        if (suggested && !manualChoice.current) {
          const preferred = eligible.find((account) => account.id === suggested);
          if (preferred)
            setName(
              (previous) =>
                previous.trim() || buildDirectAgentName(request?.prompt || '', preferred.provider),
            );
        }
      }
      setLoading(false);
    });
    return () => {
      mounted = false;
    };
  }, [open, request?.prompt, taskKind, preferredProvider, project.resource_policy]);

  const selected = useMemo(
    () => resources.find((resource) => resource.id === selectedID) || null,
    [resources, selectedID],
  );

  useEffect(() => {
    if (!selected || name.trim()) return;
    setName(buildDirectAgentName(request?.prompt || '', selected.provider));
  }, [selected, request?.prompt, name]);

  const choose = (resource: DirectResource) => {
    manualChoice.current = true;
    setSelectedID(resource.id);
    setName(buildDirectAgentName(request?.prompt || '', resource.provider));
  };

  const start = async () => {
    if (
      submitInFlight.current ||
      needsAgentRecovery ||
      loading ||
      !selected ||
      !request ||
      !modelIdValid(modelOverride.trim())
    )
      return;
    submitInFlight.current = true;
    setStarting(true);
    setError('');
    let started: Agent | null = null;
    try {
      started = await startDirectSession(nexus, {
        projectId: project.id,
        name: name.trim() || buildDirectAgentName(request.prompt, selected.provider),
        provider: selected.provider,
        profile: selected.profile,
        model: modelOverride.trim() || undefined,
        workspace: project.canonical_path || undefined,
      });
      await refreshAgents();
      await onStarted(started, request.prompt);
      onClose();
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (err instanceof DirectSessionSetupError && err.recovery !== 'deleted') {
        setNeedsAgentRecovery(true);
        setError(t('directSession.recoverableStartError', { message, agentId: err.agentId }));
      } else if (started) {
        // The runtime is running; never offer to re-create it as a fresh session.
        setNeedsAgentRecovery(true);
        setError(t('directSession.startedButNotOpened', { message, agentId: started.id }));
      } else {
        setError(message);
      }
      await refreshAgents().catch(() => undefined);
    } finally {
      submitInFlight.current = false;
      setStarting(false);
    }
  };

  const healthBadge = (health?: string) => {
    if (health === 'healthy') return <Badge tone="success">{t('overview.healthy')}</Badge>;
    if (health === 'degraded') return <Badge tone="warning">{t('status.DEGRADED')}</Badge>;
    return null;
  };

  return (
    <Dialog
      open={open}
      onClose={() => {
        if (!starting) onClose();
      }}
      title={
        request?.mode === 'assisted' ? t('directSession.assistedTitle') : t('directSession.title')
      }
      wide
    >
      <div className="nx-form-stack">
        <Input
          value={name}
          onChange={setName}
          placeholder={t('directSession.namePlaceholder')}
          aria-label={t('directSession.name')}
        />

        <div className={contextualStyles.controls}>
          <label className={contextualStyles.field}>
            <span>{t('directSession.preferProvider')}</span>
            <Select
              value={preferredProvider}
              onChange={(provider) => {
                manualChoice.current = false;
                setSelectedID('');
                setPreferredProvider(provider);
              }}
              options={[
                { value: '', label: t('directSession.providerAutomatic') },
                ...Array.from(new Set(resources.map((item) => item.provider)))
                  .sort()
                  .map((provider) => ({
                    value: provider,
                    label: provider,
                  })),
              ]}
            />
          </label>
          <label className={contextualStyles.field}>
            <span>{t('directSession.taskKind')}</span>
            <Select
              value={taskKind}
              onChange={(value) => {
                manualChoice.current = false;
                setSelectedID('');
                setRecommendation(null);
                setTaskKind(value as DirectTaskKind);
              }}
              options={(['coding', 'planning', 'review', 'security'] as const).map((kind) => ({
                value: kind,
                label: t(`directSession.task.${kind}`),
              }))}
            />
          </label>
          <label className={contextualStyles.field}>
            <span>{t('directSession.modelOverride')}</span>
            <Input
              value={modelOverride}
              onChange={setModelOverride}
              placeholder={t('directSession.providerDefaultModel')}
              aria-label={t('directSession.modelOverride')}
            />
          </label>
        </div>
        <p className={contextualStyles.hint}>{t('directSession.modelHint')}</p>
        {recommendationError && (
          <InlineAlert tone="warning">{t('directSession.recommendationUnavailable')}</InlineAlert>
        )}
        {recommendation && (
          <p className={contextualStyles.explanation}>
            {recommendation.recommended
              ? t('directSession.schedulerExplained', {
                  policy: recommendation.policy,
                  provider:
                    recommendation.recommended.account.display_name ||
                    recommendation.recommended.account.provider,
                })
              : t('directSession.schedulerNoCandidate')}
          </p>
        )}
        {!modelIdValid(modelOverride.trim()) && (
          <InlineAlert tone="warning">{t('directSession.invalidModel')}</InlineAlert>
        )}
        {loading ? (
          <Spinner label={t('directSession.loading')} />
        ) : resources.length === 0 ? (
          <EmptyState
            icon={<Bot size={22} />}
            title={t('directSession.empty')}
            hint={t('directSession.emptyHint')}
          />
        ) : (
          <ChoiceGroup
            className="nx-direct-resource-list"
            ariaLabel={t('directSession.chooseAccount')}
          >
            {Object.entries(
              resources.reduce<Record<string, DirectResource[]>>((groups, resource) => {
                (groups[resource.provider] ||= []).push(resource);
                return groups;
              }, {}),
            ).map(([provider, providerResources]) => (
              <section className={resourceStyles.providerGroup} key={provider}>
                <h3 className={resourceStyles.providerHeading}>
                  <span>{provider}</span>
                  <span className={resourceStyles.providerCount}>
                    {t('resources.accountCount', { count: providerResources.length })}
                  </span>
                </h3>
                {providerResources.map((resource) => {
                  const checked = resource.id === selectedID;
                  const quota = directQuotaDisplay(resource);
                  const quotaState = quotaTruthState(resource);
                  const candidate = recommendation?.candidates?.find(
                    (item) => item.account.id === resource.id,
                  );
                  const recommended = recommendation?.recommended?.account.id === resource.id;
                  return (
                    <ChoiceOption
                      selected={checked}
                      onSelect={() => choose(resource)}
                      data-quota-state={quotaState}
                      className={`${resourceStyles.account} nx-direct-resource`}
                      key={resource.id}
                    >
                      <span className="nx-direct-resource__icon">
                        {resource.rate_limited ? (
                          <ShieldAlert size={17} />
                        ) : (
                          <TerminalSquare size={17} />
                        )}
                      </span>
                      <span className="nx-direct-resource__body">
                        <span className="nx-direct-resource__title">
                          <strong>{directAccountTitle(resource)}</strong>
                          {healthBadge(resource.health)}
                          {recommended && (
                            <Badge tone="success">{t('directSession.recommended')}</Badge>
                          )}
                          {candidate && (
                            <Badge>
                              {t('directSession.schedulerRank', { rank: candidate.rank })}
                            </Badge>
                          )}
                          <Badge
                            tone={
                              quotaState === 'confirmed'
                                ? 'success'
                                : quotaState === 'stale'
                                  ? 'warning'
                                  : 'danger'
                            }
                          >
                            {quotaState === 'confirmed'
                              ? t('resources.confirmed')
                              : quotaState === 'exhausted'
                                ? t('resources.exhausted')
                                : quotaState === 'blocked'
                                  ? t('resources.rateLimited')
                                  : quotaState === 'unauthenticated'
                                    ? t('resources.reauthRequired')
                                    : quotaState === 'stale'
                                      ? t('resources.stale')
                                      : t('resources.unknown')}
                          </Badge>
                        </span>
                        <small>
                          {quota === null
                            ? t('directSession.quotaUnknown')
                            : t('directSession.remainingLabel', { value: quota })}
                        </small>
                      </span>
                    </ChoiceOption>
                  );
                })}
              </section>
            ))}
          </ChoiceGroup>
        )}

        {request?.prompt && (
          <div className="nx-direct-prompt-preview">
            <strong>{t('directSession.prompt')}</strong>
            <pre>{request.prompt}</pre>
          </div>
        )}
        {error && (
          <InlineAlert tone="danger" title={t('directSession.startError')}>
            {error}
          </InlineAlert>
        )}
        <div className="nx-dialog-actions">
          <Button disabled={starting} onClick={onClose}>
            {t('directSession.cancel')}
          </Button>
          <Button
            tone="brand"
            disabled={
              !selected ||
              needsAgentRecovery ||
              starting ||
              loading ||
              !name.trim() ||
              !modelIdValid(modelOverride.trim())
            }
            onClick={() => void start()}
          >
            <Play size={14} /> {starting ? t('directSession.starting') : t('directSession.start')}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
