import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Cpu, Gauge, PanelRightClose, PanelRightOpen, Plus, RefreshCw, Search } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Button, EmptyState, SearchInput, Spinner } from '../../design-system';
import { nexus } from '../../nexus/api';
import type {
  CapacityResourceStatus,
  CostSummary,
  ProviderAccount,
  ProviderInfo,
  RecommendationResult,
} from '../../types';
import { asArray } from '../../lib/safeArray';
import { AccountLoginModal } from './AccountLoginModal';
import { CostSummaryStrip } from './CostSummaryStrip';
import { RegisterProviderModal } from './RegisterProviderModal';
import { SchedulerExplainModal } from './SchedulerExplainModal';
import { UsageAccountCard } from './UsageAccountCard';
import { UsagePolicyPanel } from './UsagePolicyPanel';
import { UsageRecommendRail } from './UsageRecommendRail';
import { UsageSummaryStrip } from './UsageSummaryStrip';
import {
  filterAccounts,
  groupAccountsByProvider,
  installedProvidersWithoutAccounts,
  summarizeAccounts,
  type FleetFilterStatus,
} from './usageModel';
import { FRESHNESS_TICK_MS, USAGE_POLL_MS } from './usageRefresh';
import styles from './UsageSurface.module.scss';

// Worst-first rank so one card shows the most severe monitor state when a
// provider/profile has several windows. UNKNOWN never outranks data.
const monitorSeverityRank = (state: string): number => {
  switch (state) {
    case 'EXHAUSTED':
      return 5;
    case 'CRITICAL':
      return 4;
    case 'WARNING':
      return 3;
    case 'RECOVERED':
      return 2;
    case 'HEALTHY':
      return 1;
    default:
      return 0;
  }
};

export const UsageSurface: React.FC = () => {
  const { t } = useTranslation();
  const [accounts, setAccounts] = useState<ProviderAccount[]>([]);
  const [policy, setPolicy] = useState('BALANCED');
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [error, setError] = useState('');
  const [loadedAt, setLoadedAt] = useState<number | null>(null);
  const [now, setNow] = useState(() => Date.now());
  const [recommend, setRecommend] = useState<RecommendationResult | null>(null);
  const [recommendLoading, setRecommendLoading] = useState(false);
  const [recommendError, setRecommendError] = useState('');
  const [installedProviders, setInstalledProviders] = useState<ProviderInfo[]>([]);
  const [monitorByAccount, setMonitorByAccount] = useState<Record<string, CapacityResourceStatus>>(
    {},
  );
  const [costSummary, setCostSummary] = useState<CostSummary | null>(null);

  // Search & Filtering State
  const [search, setSearch] = useState('');
  const [providerFilter, setProviderFilter] = useState('all');
  const [statusFilter, setStatusFilter] = useState<FleetFilterStatus>('all');
  const [railCollapsed, setRailCollapsed] = useState(false);

  // Modals
  const [loginAccount, setLoginAccount] = useState<ProviderAccount | null>(null);
  const [showAddAccount, setShowAddAccount] = useState(false);
  const [addAccountTargetProvider, setAddAccountTargetProvider] = useState<string>('');
  const [showExplain, setShowExplain] = useState(false);

  const loadRecommend = useCallback(async (activePolicy: string) => {
    setRecommendLoading(true);
    setRecommendError('');
    try {
      const result = await nexus.recommendResources(
        { task_kind: 'general', role: 'developer' },
        activePolicy || 'BALANCED',
      );
      setRecommend(result);
    } catch (err) {
      setRecommend(null);
      setRecommendError(err instanceof Error ? err.message : String(err));
    } finally {
      setRecommendLoading(false);
    }
  }, []);

  const loadInstalledProviders = useCallback(async () => {
    try {
      const list = await nexus.listProviderInstallations();
      setInstalledProviders(list || []);
    } catch {
      // Non-blocking
    }
  }, []);

  const loadCostSummary = useCallback(async () => {
    try {
      setCostSummary(await nexus.getCostSummary(7));
    } catch {
      // Cost Intelligence is advisory. Missing evidence must not break quota UX.
      setCostSummary(null);
    }
  }, []);

  const applyPayload = useCallback(
    (
      data: { accounts?: ProviderAccount[]; policy?: string },
      options?: { recommend?: boolean },
    ) => {
      const nextAccounts = asArray<ProviderAccount>(data.accounts);
      const nextPolicy = data.policy || 'BALANCED';
      setAccounts(nextAccounts);
      setPolicy(nextPolicy);
      setLoadedAt(Date.now());
      setNow(Date.now());
      if (options?.recommend !== false) {
        void loadRecommend(nextPolicy);
      }
    },
    [loadRecommend],
  );

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      applyPayload(await nexus.listResources());
      void loadInstalledProviders();
      void loadCostSummary();
      // Capacity status is advisory: a failure here must never break usage.
      try {
        const status = await nexus.getCapacityStatus();
        const byAccount: Record<string, CapacityResourceStatus> = {};
        for (const res of asArray<CapacityResourceStatus>(status.resources)) {
          const key = `${res.provider}/${res.profile}`;
          const prev = byAccount[key];
          if (!prev || monitorSeverityRank(res.state) > monitorSeverityRank(prev.state)) {
            byAccount[key] = res;
          }
        }
        setMonitorByAccount(byAccount);
      } catch {
        setMonitorByAccount({});
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setAccounts([]);
    } finally {
      setLoading(false);
    }
  }, [applyPayload, loadCostSummary, loadInstalledProviders]);

  const softReload = useCallback(async () => {
    if (document.hidden) return;
    try {
      applyPayload(await nexus.listResources(), { recommend: false });
    } catch {
      // Keep last good snapshot
    }
  }, [applyPayload]);

  const refresh = useCallback(
    async (provider?: string, profile?: string) => {
      setRefreshing(true);
      setError('');
      try {
        applyPayload(await nexus.refreshResources(provider, profile));
        void loadInstalledProviders();
        void loadCostSummary();
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setRefreshing(false);
      }
    },
    [applyPayload, loadCostSummary, loadInstalledProviders],
  );

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    const onVisible = () => {
      if (!document.hidden) void softReload();
    };
    const poll = window.setInterval(() => {
      void softReload();
    }, USAGE_POLL_MS);
    const tick = window.setInterval(() => setNow(Date.now()), FRESHNESS_TICK_MS);
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      window.clearInterval(poll);
      window.clearInterval(tick);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [softReload]);

  const summary = useMemo(() => summarizeAccounts(accounts), [accounts]);

  const availableProviders = useMemo(() => {
    const set = new Set<string>();
    for (const a of accounts) {
      if (a.provider) set.add(a.provider);
    }
    return Array.from(set).sort();
  }, [accounts]);

  const filteredAccounts = useMemo(() => {
    return filterAccounts(accounts, {
      search,
      provider: providerFilter,
      status: statusFilter,
    });
  }, [accounts, search, providerFilter, statusFilter]);

  const groups = useMemo(() => groupAccountsByProvider(filteredAccounts), [filteredAccounts]);

  const bestLabel = recommend?.recommended
    ? recommend.recommended.account.display_name ||
      `${recommend.recommended.account.provider}:${recommend.recommended.account.profile}`
    : null;

  const unconfiguredClis = useMemo(
    () => installedProvidersWithoutAccounts(installedProviders, accounts),
    [accounts, installedProviders],
  );

  const handleSelectBest = () => {
    if (recommend?.recommended) {
      setSearch(recommend.recommended.account.profile || '');
    }
  };

  return (
    <div className={`nx-surface-scroll ${styles.root}`}>
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">{t('surfaces.resourcesEyebrow')}</span>
          <h1>{t('surfaces.resourcesTitle')}</h1>
          <p>{t('surfaces.resourcesIntro')}</p>
        </div>
        <div className={styles.headerActions}>
          {loadedAt ? (
            <small className={styles.loadedAt}>
              {t('usage.loadedAt', 'Atualizado')}: {new Date(loadedAt).toLocaleTimeString()}
            </small>
          ) : null}
          <Button
            size="sm"
            tone="brand"
            disabled={loading || refreshing}
            onClick={() => setShowAddAccount(true)}
          >
            <Plus size={13} />
            {t('usage.addAccount', 'Adicionar conta')}
          </Button>
          <Button
            size="sm"
            tone="default"
            disabled={loading || refreshing}
            onClick={() => void refresh()}
          >
            <RefreshCw size={13} className={refreshing ? 'nx-spin' : ''} />
            {refreshing
              ? t('usage.refreshing', 'Atualizando…')
              : t('usage.refresh', 'Atualizar cota')}
          </Button>
        </div>
      </div>

      {error ? <div className="nx-inline-error">{error}</div> : null}
      {loading ? <Spinner label={t('resources.loading')} /> : null}

      {!loading && unconfiguredClis.length > 0 ? (
        <div className={styles.detectedBanner}>
          <div className={styles.detectedCopy}>
            <strong>
              <Cpu size={15} color="var(--nx-accent)" />
              {t('usage.installedCliTitle', 'Provedores CLI detectados no sistema')}
            </strong>
            <span>{t('usage.installedCliSubtitle')}</span>
            <div className={styles.detectedChips}>
              {unconfiguredClis.map((provider) => (
                <span key={provider.id} className={styles.detectedChip}>
                  {provider.id} ({provider.version || 'instalado'})
                </span>
              ))}
            </div>
          </div>
          <Button
            size="sm"
            tone="brand"
            onClick={() => {
              setAddAccountTargetProvider(unconfiguredClis.length === 1 ? unconfiguredClis[0].id : '');
              setShowAddAccount(true);
            }}
          >
            <Plus size={13} />
            {t('usage.addAccount', 'Registrar')}
          </Button>
        </div>
      ) : null}

      {!loading && accounts.length === 0 && unconfiguredClis.length === 0 ? (
        <EmptyState
          icon={<Gauge size={20} />}
          title={t('resources.empty')}
          hint={t('resources.emptyHint')}
          action={
            <Button tone="brand" onClick={() => setShowAddAccount(true)}>
              <Plus size={14} />
              {t('usage.addAccount', 'Adicionar conta')}
            </Button>
          }
        />
      ) : null}

      {!loading && accounts.length > 0 ? (
        <>
          <UsageSummaryStrip
            summary={summary}
            bestLabel={bestLabel}
            activeStatusFilter={statusFilter}
            onSelectStatusFilter={setStatusFilter}
            onSelectBest={handleSelectBest}
          />

          <CostSummaryStrip summary={costSummary} />

          <div className={styles.toolbar}>
            <div className={styles.searchWrap}>
              <SearchInput
                value={search}
                onChange={setSearch}
                placeholder={t('usage.searchPlaceholder')}
              />
            </div>

            <div className={styles.filterPills} role="group" aria-label="Filtro de provedores">
              <button
                type="button"
                className={styles.filterPill}
                data-active={providerFilter === 'all'}
                onClick={() => setProviderFilter('all')}
              >
                {t('usage.filterProviderAll', 'Todos provedores')} ({accounts.length})
              </button>
              {availableProviders.map((prov) => {
                const count = accounts.filter((a) => a.provider === prov).length;
                return (
                  <button
                    key={prov}
                    type="button"
                    className={styles.filterPill}
                    data-active={providerFilter === prov}
                    onClick={() => setProviderFilter(providerFilter === prov ? 'all' : prov)}
                  >
                    {prov.toUpperCase()} ({count})
                  </button>
                );
              })}
            </div>

            <Button
              size="sm"
              tone="default"
              onClick={() => setRailCollapsed((prev) => !prev)}
              title={t('usage.toggleSidebar', 'Alternar scheduler')}
            >
              {railCollapsed ? <PanelRightOpen size={14} /> : <PanelRightClose size={14} />}
              {railCollapsed ? 'Scheduler' : null}
            </Button>
          </div>

          <div className={styles.layout} data-rail-collapsed={railCollapsed}>
            <div className={styles.main}>
              {groups.length === 0 ? (
                <EmptyState
                  icon={<Search size={20} />}
                  title={t('common.noResults', 'Nenhuma conta encontrada')}
                  hint="Nenhum provedor corresponde aos filtros selecionados."
                  action={
                    <Button
                      size="sm"
                      onClick={() => {
                        setSearch('');
                        setProviderFilter('all');
                        setStatusFilter('all');
                      }}
                    >
                      Limpar filtros
                    </Button>
                  }
                />
              ) : null}

              <div className={styles.fleet}>
                {groups.map((group) => (
                  <section key={group.provider} className={styles.providerGroup}>
                    <h2 className={styles.providerHeading}>
                      <span>{group.provider}</span>
                      <small>{t('resources.accountCount', { count: group.accounts.length })}</small>
                    </h2>
                    <div className={styles.accountGrid}>
                      {group.accounts.map((account) => (
                        <UsageAccountCard
                          key={account.id}
                          account={account}
                          now={now}
                          isRecommended={
                            recommend?.recommended?.account?.id === account.id ||
                            (recommend?.recommended?.account?.provider === account.provider &&
                              recommend?.recommended?.account?.profile === account.profile)
                          }
                          monitorState={
                            monitorByAccount[`${account.provider}/${account.profile}`] || null
                          }
                          onRefreshNeeded={() => refresh(account.provider, account.profile)}
                          onCapacityChange={() => void load()}
                          onRequestLogin={(acc) => setLoginAccount(acc)}
                          onRequestExplain={() => setShowExplain(true)}
                          onAddProfile={(prov) => {
                            setAddAccountTargetProvider(prov);
                            setShowAddAccount(true);
                          }}
                        />
                      ))}
                    </div>
                  </section>
                ))}
              </div>
            </div>

            {!railCollapsed && (
              <aside className={styles.rail}>
                <UsageRecommendRail
                  loading={recommendLoading}
                  result={recommend}
                  error={recommendError}
                  onRequestExplain={() => setShowExplain(true)}
                />
                <UsagePolicyPanel policy={policy} />
              </aside>
            )}
          </div>
        </>
      ) : null}

      <AccountLoginModal
        account={loginAccount}
        isOpen={Boolean(loginAccount)}
        onClose={() => setLoginAccount(null)}
        onLoginSuccess={async () => {
          await refresh();
        }}
      />

      <RegisterProviderModal
        isOpen={showAddAccount}
        initialProvider={addAccountTargetProvider}
        onClose={() => {
          setShowAddAccount(false);
          setAddAccountTargetProvider('');
        }}
        onRegistered={async (prov, name) => {
          await refresh();
          // Prompt login if new account needs it
          const newlyAdded = accounts.find((a) => a.provider === prov && a.profile === name);
          if (newlyAdded) setLoginAccount(newlyAdded);
        }}
      />

      <SchedulerExplainModal
        isOpen={showExplain}
        onClose={() => setShowExplain(false)}
        result={recommend}
        policy={policy}
      />
    </div>
  );
};
