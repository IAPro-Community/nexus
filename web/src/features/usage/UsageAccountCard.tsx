import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Activity,
  CheckCircle2,
  Clock,
  HelpCircle,
  KeyRound,
  LogOut,
  MoreVertical,
  Plus,
  RefreshCw,
  ShieldAlert,
  Star,
  Terminal,
  Trash2,
} from 'lucide-react';
import { toast } from 'sonner';
import {
  Badge,
  Button,
  ConfirmDialog,
  DropdownMenu,
  IconButton,
  Progress,
} from '../../design-system';
import { translateStatus } from '../../i18n';
import type { CapacityResourceStatus, ProviderAccount } from '../../types';
import { asArray } from '../../lib/safeArray';
import { nexus } from '../../nexus/api';
import {
  accountQuotaState,
  capabilityLabelKey,
  displayLabelKey,
  formatResetDescription,
  getQuotaProgressTone,
  groupHasCapacity,
  isAccountAuthRequired,
  relativeAge,
  windowKindLabelKey,
} from './usageModel';
import styles from './UsageAccountCard.module.scss';

const healthTone = (health: string) =>
  health === 'healthy'
    ? 'success'
    : health === 'degraded'
      ? 'warning'
      : health === 'unhealthy'
        ? 'danger'
        : 'default';

const quotaTone = (state: string) =>
  state === 'confirmed' ? 'success' : state === 'stale' ? 'warning' : 'danger';

const compactNumber = (value?: number) => {
  if (typeof value !== 'number' || !Number.isFinite(value)) return null;
  return new Intl.NumberFormat(undefined, {
    notation: value >= 10000 ? 'compact' : 'standard',
    maximumFractionDigits: value >= 1000 ? 1 : 0,
  }).format(value);
};

const money = (currency?: string, value?: number) => {
  if (typeof value !== 'number' || !Number.isFinite(value)) return null;
  const code = (currency || 'USD').toUpperCase();
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: code,
      maximumFractionDigits: 2,
    }).format(value);
  } catch {
    return `${code} ${value.toFixed(2)}`;
  }
};

export interface UsageAccountCardProps {
  account: ProviderAccount;
  now?: number;
  isRecommended?: boolean;
  monitorState?: CapacityResourceStatus | null;
  onRefreshNeeded: () => Promise<void>;
  onCapacityChange?: () => void;
  onRequestLogin: (account: ProviderAccount) => void;
  onRequestExplain: () => void;
  onOpenTerminal?: (provider: string, profile: string) => void;
  onAddProfile?: (provider: string) => void;
}

const monitorTone = (state: string) =>
  state === 'HEALTHY' || state === 'RECOVERED'
    ? 'success'
    : state === 'WARNING'
      ? 'warning'
      : state === 'CRITICAL' || state === 'EXHAUSTED'
        ? 'danger'
        : 'default';

export const UsageAccountCard: React.FC<UsageAccountCardProps> = ({
  account,
  now,
  isRecommended,
  monitorState,
  onRefreshNeeded,
  onCapacityChange,
  onRequestLogin,
  onRequestExplain,
  onOpenTerminal,
  onAddProfile,
}) => {
  const { t, i18n } = useTranslation();
  const [refreshing, setRefreshing] = useState(false);
  const [confirmRemove, setConfirmRemove] = useState(false);
  const [acking, setAcking] = useState(false);

  const qv = account.quota_view;
  const groups = asArray<
    NonNullable<NonNullable<ProviderAccount['quota_view']>['model_groups']>[number]
  >(qv?.model_groups);
  const multi = groups.length > 1;
  const quotaState = accountQuotaState(account);
  const authRequired = isAccountAuthRequired(account);
  const capabilities = Object.entries(account.capabilities || {})
    .filter(([, value]) => value && value !== 'false')
    .slice(0, 5);
  const apiChannel = asArray<NonNullable<ProviderAccount['resource_channels']>[number]>(
    account.resource_channels,
  ).find((channel) => channel.channel === 'API');

  const planKey = displayLabelKey(qv?.plan);
  const age = relativeAge(qv?.fetched_at, now);
  const statusLabel = translateStatus(qv?.status || 'UNKNOWN');
  const freshness = age
    ? t(`usage.freshness.${age.unit}`, { status: statusLabel, count: age.count })
    : statusLabel;

  const handleRefreshAccount = async () => {
    setRefreshing(true);
    try {
      await nexus.refreshResources(account.provider, account.profile);
      await onRefreshNeeded();
      toast.success(t('usage.refreshSuccess', 'Cota atualizada com sucesso'));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setRefreshing(false);
    }
  };

  const handleAckMonitor = async () => {
    if (!monitorState?.fingerprint) return;
    setAcking(true);
    try {
      await nexus.ackCapacityAlert(monitorState.fingerprint);
      toast.success(t('settings.capacityAcked', 'Acknowledged'));
      onCapacityChange?.();
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setAcking(false);
    }
  };

  const handleMakeDefault = async () => {
    try {
      await nexus.setDefaultProfile(account.provider, account.profile);
      await onRefreshNeeded();
      toast.success(
        t('usage.makeDefaultSuccess', { name: account.profile, provider: account.provider }),
      );
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  };

  const handleLogout = async () => {
    try {
      await nexus.logoutProfile(account.provider, account.profile);
      await onRefreshNeeded();
      toast.success(t('usage.logoutSuccess'));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    }
  };

  const handleRemoveProfile = async () => {
    try {
      await nexus.deleteProfile(account.provider, account.profile);
      await onRefreshNeeded();
      toast.success(t('common.deleted', 'Perfil removido com sucesso'));
    } catch (err) {
      toast.error(err instanceof Error ? err.message : String(err));
    } finally {
      setConfirmRemove(false);
    }
  };

  const handleTerminalLaunch = () => {
    if (onOpenTerminal) {
      onOpenTerminal(account.provider, account.profile);
    } else {
      window.dispatchEvent(
        new CustomEvent('nexus:terminal-launch', {
          detail: {
            provider: account.provider,
            profile: account.profile,
            args: [],
            title: `${account.provider} · ${account.profile}`,
          },
        }),
      );
    }
  };

  return (
    <>
      <article
        className={styles.card}
        data-quota-state={quotaState}
        data-available={account.available}
        data-recommended={isRecommended ? 'true' : 'false'}
      >
        <header className={styles.header}>
          <div className={styles.headerLeft}>
            <span className={styles.icon} aria-hidden="true">
              {account.rate_limited ? <ShieldAlert size={18} /> : <Activity size={18} />}
            </span>
            <div className={styles.titleBlock}>
              <div className={styles.titleRow}>
                <span className={styles.accountName}>
                  {account.display_name || `${account.provider} (${account.profile})`}
                </span>
                {account.is_default ? (
                  <Badge tone="brand">
                    <Star size={10} style={{ marginRight: 3 }} />
                    {t('common.default', 'padrão')}
                  </Badge>
                ) : null}
                {isRecommended ? (
                  <Badge tone="brand">{t('usage.bestChoice', 'Melhor escolha')} #1</Badge>
                ) : null}
                <Badge tone={healthTone(account.health)}>{translateStatus(account.health)}</Badge>
                <Badge tone={quotaTone(quotaState)}>
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
              </div>

              <div className={styles.metaRow}>
                {qv?.plan ? <small>{planKey ? t(planKey) : qv.plan}</small> : null}
                {qv?.account ? <small>{qv.account}</small> : null}
                <small>
                  <Clock size={11} />
                  {freshness}
                </small>
                {account.cooldown_until ? (
                  <small style={{ color: 'var(--nx-warning)' }}>
                    {t('usage.cooldown')}: {new Date(account.cooldown_until).toLocaleTimeString()}
                  </small>
                ) : null}
              </div>
            </div>
          </div>

          <DropdownMenu.Root>
            <DropdownMenu.Trigger asChild>
              <IconButton label={t('common.actions', 'Ações')} className={styles.menuButton}>
                <MoreVertical size={16} />
              </IconButton>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content className={styles.menuContent} side="bottom" align="end">
                {(account.available || account.authenticated) && (
                  <DropdownMenu.Item className={styles.menuItem} onClick={handleTerminalLaunch}>
                    <Terminal size={14} />
                    {t('usage.openTerminalCta')}
                  </DropdownMenu.Item>
                )}
                <DropdownMenu.Item
                  className={styles.menuItem}
                  onClick={() => void handleRefreshAccount()}
                >
                  <RefreshCw size={14} className={refreshing ? 'nx-spin' : ''} />
                  {t('usage.refreshAccountCta')}
                </DropdownMenu.Item>
                {(authRequired || account.provider?.toLowerCase() === 'opencode') && (
                  <DropdownMenu.Item
                    className={styles.menuItem}
                    onClick={() => onRequestLogin(account)}
                  >
                    <KeyRound size={14} />
                    {account.provider?.toLowerCase() === 'opencode'
                      ? t('usage.configProvidersCta', 'Gerenciar provedores')
                      : t('usage.loginCta')}
                  </DropdownMenu.Item>
                )}
                {!account.is_default && (
                  <DropdownMenu.Item
                    className={styles.menuItem}
                    onClick={() => void handleMakeDefault()}
                  >
                    <Star size={14} />
                    {t('usage.makeDefaultCta')}
                  </DropdownMenu.Item>
                )}
                <DropdownMenu.Item className={styles.menuItem} onClick={onRequestExplain}>
                  <HelpCircle size={14} />
                  {t('usage.explainSchedulerCta')}
                </DropdownMenu.Item>
                {onAddProfile && (
                  <DropdownMenu.Item
                    className={styles.menuItem}
                    onClick={() => onAddProfile(account.provider)}
                  >
                    <Plus size={14} />
                    {t('usage.addAnotherProfileCta', 'Adicionar outro perfil')}
                  </DropdownMenu.Item>
                )}
                <DropdownMenu.Separator className={styles.menuSeparator} />
                <DropdownMenu.Item
                  className={styles.menuItem}
                  data-danger="true"
                  onClick={() => void handleLogout()}
                >
                  <LogOut size={14} />
                  {t('usage.logoutCta')}
                </DropdownMenu.Item>
                <DropdownMenu.Item
                  className={styles.menuItem}
                  data-danger="true"
                  onClick={() => setConfirmRemove(true)}
                >
                  <Trash2 size={14} />
                  {t('usage.removeAccountCta')}
                </DropdownMenu.Item>
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        </header>

        {capabilities.length > 0 ? (
          <div className={styles.capsSection}>
            <span className={styles.capsLabel}>{t('usage.capabilitiesTitle')}</span>
            <div className={styles.capsList}>
              {capabilities.map(([key]) => (
                <span key={key} className={styles.capChip}>
                  {t(capabilityLabelKey(key), { defaultValue: key })}
                </span>
              ))}
            </div>
          </div>
        ) : null}

        <div className={styles.quota}>
          {monitorState && monitorState.state !== 'UNKNOWN' ? (
            <p className={styles.stateLine} data-state={monitorState.state.toLowerCase()}>
              <Badge tone={monitorTone(monitorState.state)}>
                {t(`attention.capacityState_${monitorState.state}`, {
                  defaultValue: monitorState.state,
                })}
              </Badge>
              {(() => {
                const moved = relativeAge(monitorState.transitioned_at, now);
                const stateLabel = t(`attention.capacityState_${monitorState.state}`, {
                  defaultValue: monitorState.state,
                });
                return moved
                  ? t(`usage.freshness.${moved.unit}`, {
                      status: stateLabel,
                      count: moved.count,
                    })
                  : stateLabel;
              })()}
              {monitorState.acked ? (
                <span>{t('settings.capacityAcked', 'Acknowledged')}</span>
              ) : monitorState.fingerprint ? (
                <Button size="sm" disabled={acking} onClick={() => void handleAckMonitor()}>
                  {t('settings.capacityAck', 'Acknowledge')}
                </Button>
              ) : null}
            </p>
          ) : null}
          {groups.length > 0 ? (
            groups.map((group, index) => {
              const groupLabelKey = displayLabelKey(group.name, group.key);
              return (
                <section key={group.key || group.name || index} className={styles.group}>
                  {multi && (groupLabelKey || group.name) ? (
                    <div className={styles.groupHeading}>
                      <span>{groupLabelKey ? t(groupLabelKey) : group.name}</span>
                      <Badge tone={groupHasCapacity(group) ? 'success' : 'danger'}>
                        {groupHasCapacity(group)
                          ? t('usage.groupAvailable')
                          : t('usage.groupUnavailable')}
                      </Badge>
                    </div>
                  ) : null}
                  {asArray<NonNullable<typeof group.windows>[number]>(group.windows).map(
                    (window) => {
                      const windowKey =
                        windowKindLabelKey(window.kind) || displayLabelKey(window.label);
                      const reset = formatResetDescription(
                        window.reset_desc,
                        i18n.language,
                        (key, options) => t(key, options),
                      );
                      const remainingVal = Number(window.remaining) || 0;
                      const tone = getQuotaProgressTone(remainingVal);
                      return (
                        <div key={window.kind || window.label} className={styles.window}>
                          <span className={styles.windowLabel}>
                            {windowKey ? t(windowKey) : window.label || window.kind}
                          </span>
                          <Progress
                            value={remainingVal}
                            label={`${Math.round(remainingVal)}%`}
                            data-tone={tone}
                          />
                          {reset ? (
                            <span className={styles.reset}>
                              <Clock size={10} />
                              {reset}
                            </span>
                          ) : null}
                        </div>
                      );
                    },
                  )}
                </section>
              );
            })
          ) : (
            <p className={styles.stateLine}>{t('resources.state.unknown')}</p>
          )}
        </div>

        {apiChannel ? (
          <section className={styles.capsSection} aria-label={t('usage.apiUsageTitle', 'Uso de API')}>
            <div className={styles.titleRow}>
              <span className={styles.capsLabel}>{t('usage.apiUsageTitle', 'Uso de API')}</span>
              <Badge tone={apiChannel.status === 'LIVE' || apiChannel.status === 'CACHED' ? 'success' : 'warning'}>
                {translateStatus(apiChannel.status)}
              </Badge>
              {apiChannel.source ? <Badge tone="default">{apiChannel.source}</Badge> : null}
            </div>
            <div className={styles.capsList}>
              {compactNumber(apiChannel.consumption?.input_tokens) ? (
                <span className={styles.capChip}>
                  {t('usage.apiInputTokens', 'Entrada')}: {compactNumber(apiChannel.consumption?.input_tokens)}
                </span>
              ) : null}
              {compactNumber(apiChannel.consumption?.cached_tokens) ? (
                <span className={styles.capChip}>
                  {t('usage.apiCachedTokens', 'Cache')}: {compactNumber(apiChannel.consumption?.cached_tokens)}
                </span>
              ) : null}
              {compactNumber(apiChannel.consumption?.output_tokens) ? (
                <span className={styles.capChip}>
                  {t('usage.apiOutputTokens', 'Saída')}: {compactNumber(apiChannel.consumption?.output_tokens)}
                </span>
              ) : null}
              {compactNumber(apiChannel.consumption?.requests) ? (
                <span className={styles.capChip}>
                  {t('usage.apiRequests', 'Requests')}: {compactNumber(apiChannel.consumption?.requests)}
                </span>
              ) : null}
              {money(apiChannel.cost?.currency, apiChannel.cost?.amount) ? (
                <span className={styles.capChip}>
                  {t('usage.apiCost', 'Custo')}: {money(apiChannel.cost?.currency, apiChannel.cost?.amount)}
                </span>
              ) : null}
              {money(apiChannel.cost?.currency, apiChannel.cost?.budget_remaining) ? (
                <span className={styles.capChip}>
                  {t('usage.apiBudgetRemaining', 'Orçamento restante')}:{' '}
                  {money(apiChannel.cost?.currency, apiChannel.cost?.budget_remaining)}
                </span>
              ) : null}
            </div>
            {apiChannel.observed_at ? (
              <small className={styles.stateLine}>
                {t('usage.apiObservedAt', 'Telemetria API')}: {new Date(apiChannel.observed_at).toLocaleString()}
              </small>
            ) : null}
          </section>
        ) : null}

        <footer className={styles.footer}>
          {account.eligibility ? (
            <span className={styles.eligibilityNote} data-eligible={account.eligibility.eligible}>
              {account.eligibility.eligible ? (
                <CheckCircle2 size={13} />
              ) : (
                <ShieldAlert size={13} />
              )}
              {t(`resources.eligibility.${account.eligibility.reason}`, {
                defaultValue: account.eligibility.message || account.eligibility.reason,
              })}
            </span>
          ) : (
            <span />
          )}

          <div className={styles.footerActions}>
            {authRequired ? (
              <Button size="sm" tone="warning" onClick={() => onRequestLogin(account)}>
                <KeyRound size={13} />
                {t('usage.loginCta')}
              </Button>
            ) : account.available || account.authenticated ? (
              <Button size="sm" tone="brand" onClick={handleTerminalLaunch}>
                <Terminal size={13} />
                {t('usage.openTerminalCta')}
              </Button>
            ) : null}
          </div>
        </footer>
      </article>

      <ConfirmDialog
        open={confirmRemove}
        title={t('usage.removeAccountConfirmTitle', { name: account.profile })}
        description={t('usage.removeAccountConfirmDesc')}
        confirmLabel={t('usage.removeAccountCta')}
        tone="danger"
        onConfirm={() => void handleRemoveProfile()}
        onCancel={() => setConfirmRemove(false)}
      />
    </>
  );
};
