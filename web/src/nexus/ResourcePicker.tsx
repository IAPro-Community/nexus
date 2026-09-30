import React, { useEffect, useState } from 'react';
import { Activity, Check, Gauge, ShieldAlert } from 'lucide-react';
import { Badge, EmptyState, Progress, Spinner } from '../design-system';
import { nexus } from './api';
import { translateStatus } from '../i18n';
import { useTranslation } from 'react-i18next';
import styles from './ResourcePicker.module.scss';
import { isMeasuredQuotaStatus, quotaTruthState } from '../features/work/directSessionModel';
import {
  displayLabelKey,
  formatResetDescription,
  windowKindLabelKey,
} from '../features/usage/usageModel';
import { USAGE_POLL_MS } from '../features/usage/usageRefresh';

import type { ProviderAccount, SchedulerDecision } from '../types';

interface DisplayQuotaWindow {
  kind: string;
  label: string;
  remaining: number;
  reset_desc: string;
  status: string;
  bar: string;
}

interface DisplayQuotaModelGroup {
  name: string;
  windows: DisplayQuotaWindow[];
}

interface Props {
  agentId?: string;
  preferProvider?: string;
  onSelected?: (provider: string, profile: string) => void | Promise<void>;
}

const healthTone = (health: string) =>
  health === 'healthy'
    ? 'success'
    : health === 'degraded'
      ? 'warning'
      : health === 'unhealthy'
        ? 'danger'
        : 'default';

const QuotaBar: React.FC<{ w: DisplayQuotaWindow }> = ({ w }) => {
  const { t, i18n } = useTranslation();
  const windowKey = windowKindLabelKey(w.kind) || displayLabelKey(w.label);
  const reset = formatResetDescription(w.reset_desc, i18n.language, (key, options) =>
    t(key, options),
  );
  return (
    <span className="nx-resource-account__quota-row">
      <span className="nx-resource-account__quota-label">{windowKey ? t(windowKey) : w.label}</span>
      {w.kind !== 'unknown' && isMeasuredQuotaStatus(w.status) ? (
        <Progress value={w.remaining} label={`${Math.round(w.remaining)}%`} />
      ) : (
        <span className="nx-resource-account__quota-unknown">{t('resources.unknown')}</span>
      )}
      {reset ? <span className="nx-resource-account__quota-reset">{reset}</span> : null}
    </span>
  );
};

const groupAvailable = (group: DisplayQuotaModelGroup) =>
  group.windows.some(
    (w) => w.kind !== 'unknown' && isMeasuredQuotaStatus(w.status) && w.remaining > 0,
  );

function displayQuotaGroups(account: ProviderAccount): DisplayQuotaModelGroup[] {
  return (account.quota_view?.model_groups || []).map((group) => ({
    name: group.name || group.key || 'Quota',
    windows: (group.windows || []).map((window) => ({
      kind: window.kind || 'unknown',
      label: window.label || window.kind || 'Quota',
      remaining: typeof window.remaining === 'number' ? window.remaining : 0,
      reset_desc: window.reset_desc || '',
      status: window.status || 'UNKNOWN',
      bar: window.status || 'UNKNOWN',
    })),
  }));
}

export const ResourcePicker: React.FC<Props> = ({ agentId, preferProvider, onSelected }) => {
  const { t } = useTranslation();
  const [accounts, setAccounts] = useState<ProviderAccount[]>([]);
  const [decision, setDecision] = useState<SchedulerDecision | null>(null);
  const [loading, setLoading] = useState(true);
  const [selecting, setSelecting] = useState<string | null>(null);

  useEffect(() => {
    let mounted = true;
    const load = () =>
      nexus
        .listResources()
        .then((data) => {
          if (!mounted) return;
          setAccounts(data.accounts || []);
        })
        .catch(() => undefined)
        .finally(() => {
          if (mounted) setLoading(false);
        });

    void load();
    const poll = window.setInterval(() => {
      if (document.hidden) return;
      void load();
    }, USAGE_POLL_MS);
    const onVisible = () => {
      if (!document.hidden) void load();
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      mounted = false;
      window.clearInterval(poll);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [preferProvider]);

  const choose = async (account: ProviderAccount) => {
    if (!agentId) return;
    setSelecting(account.id);
    try {
      const result = await nexus.selectResource(agentId, account.provider, account.profile);
      setDecision(result.decision);
      await onSelected?.(account.provider, account.profile);
    } finally {
      setSelecting(null);
    }
  };

  if (loading) return <Spinner label={t('resources.loading')} />;
  if (accounts.length === 0)
    return (
      <EmptyState
        icon={<Gauge size={20} />}
        title={t('resources.empty')}
        hint={t('resources.emptyHint')}
      />
    );

  const providerGroups = accounts.reduce<Record<string, ProviderAccount[]>>((groups, account) => {
    const key = account.provider || 'unknown';
    (groups[key] ||= []).push(account);
    return groups;
  }, {});

  return (
    <div className="nx-resource-picker">
      <div className="nx-resource-account-list">
        {Object.entries(providerGroups).map(([provider, providerAccounts]) => (
          <section className={styles.providerGroup} key={provider}>
            <h3 className={styles.providerHeading}>
              <span>{provider}</span>
              <span className={styles.providerCount}>
                {t('resources.accountCount', { count: providerAccounts.length })}
              </span>
            </h3>
            {providerAccounts.map((account) => {
              const selected = decision?.selected?.id === account.id;
              const groups = displayQuotaGroups(account);
              const hasGroups = groups.length > 0;
              const multiGroups = groups.length > 1;
              const quotaState = quotaTruthState(account);

              return (
                <button
                  key={account.id}
                  type="button"
                  className={`${styles.account} nx-resource-account`}
                  data-quota-state={quotaState}
                  data-selected={selected}
                  onClick={() => void choose(account)}
                  disabled={
                    !agentId ||
                    !account.authenticated ||
                    !account.available ||
                    account.eligibility?.eligible === false ||
                    selecting === account.id
                  }
                  title={account.eligibility?.message || undefined}
                >
                  <span className="nx-resource-account__icon">
                    {account.rate_limited ? <ShieldAlert size={17} /> : <Activity size={17} />}
                  </span>
                  <span className="nx-resource-account__main">
                    <span className="nx-resource-account__title">
                      <strong>{account.display_name || account.provider}</strong>
                      <Badge>{account.profile}</Badge>
                      <Badge tone={healthTone(account.health)}>
                        {translateStatus(account.health)}
                      </Badge>
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
                      {account.is_default && <Badge>{t('common.default')}</Badge>}
                      {selected && (
                        <Badge>
                          <Check size={12} /> {t('common.selected')}
                        </Badge>
                      )}
                    </span>
                    <span className="nx-resource-account__quota">
                      {hasGroups ? (
                        groups.map((group, gi) => (
                          <span key={gi} className="nx-resource-account__group">
                            {multiGroups && group.name && (
                              <span className="nx-resource-account__group-heading">
                                <span className="nx-resource-account__group-name">
                                  {displayLabelKey(group.name)
                                    ? t(displayLabelKey(group.name) as string)
                                    : group.name}
                                </span>
                                <Badge tone={groupAvailable(group) ? 'success' : 'danger'}>
                                  {groupAvailable(group)
                                    ? t('usage.groupAvailable', 'Disponível')
                                    : t('usage.groupUnavailable', 'Indisponível')}
                                </Badge>
                              </span>
                            )}
                            {group.windows.map((w) => (
                              <QuotaBar key={w.kind} w={w} />
                            ))}
                          </span>
                        ))
                      ) : (
                        <span className="nx-resource-account__quota-unknown">
                          {t('resources.unknown')}
                        </span>
                      )}
                    </span>
                    <span className={styles.quotaState} data-state={quotaState}>
                      {t(`resources.state.${quotaState}`)}
                    </span>
                    {account.eligibility ? (
                      <span
                        className={styles.eligibility}
                        data-eligible={account.eligibility.eligible}
                        data-reason={account.eligibility.reason}
                      >
                        {t(`resources.eligibility.${account.eligibility.reason}`, {
                          defaultValue: account.eligibility.message || account.eligibility.reason,
                        })}
                      </span>
                    ) : null}
                  </span>
                  {selecting === account.id && <Spinner />}
                </button>
              );
            })}
          </section>
        ))}
      </div>
      {decision && (
        <div className="nx-resource-decision">
          <span className="nx-resource-decision__label">{t('resources.decision')}</span>
          <span className="nx-resource-decision__reason">{decision.reason}</span>
        </div>
      )}
    </div>
  );
};
