import React, { useEffect, useMemo, useState } from 'react';
import { RefreshCw, Stethoscope } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, EmptyState, IconButton } from '../../design-system';
import { nexus } from '../../nexus/api';
import { asArray } from '../../lib/safeArray';
import { translateStatus } from '../../i18n';
import type {
  CapacityResourceStatus,
  MaestroStatus,
  ProviderAccount,
  ProviderInfo,
} from '../../types';
import {
  collectObservations,
  deriveDiagnosticsProblems,
  isObservationStale,
  type DiagnosticAction,
  type DiagnosticProblem,
  type ObservationState,
} from './usageDiagnosticsModel';
import styles from './UsageDiagnosticsPanel.module.scss';

const stateTone = (state: ObservationState) =>
  state === 'LIVE'
    ? 'success'
    : state === 'CACHED' || state === 'ESTIMATED' || state === 'RATE_LIMITED'
      ? 'warning'
      : state === 'UNAVAILABLE'
        ? 'danger'
        : 'default';

const severityTone = (severity: DiagnosticProblem['severity']) =>
  severity === 'critical' ? 'danger' : severity === 'warning' ? 'warning' : 'info';

function observationAgeKey(ageSeconds: number): string {
  if (ageSeconds < 60) return 'usage.diagnostics.age.seconds';
  if (ageSeconds < 3600) return 'usage.diagnostics.age.minutes';
  return 'usage.diagnostics.age.hours';
}

export interface UsageDiagnosticsPanelProps {
  accounts: ProviderAccount[];
  monitorByAccount: Record<string, CapacityResourceStatus>;
  unconfiguredProviders: ProviderInfo[];
  now?: number;
  onRefreshAccount: (provider: string, profile: string) => void;
  onRequestLogin: (account: ProviderAccount) => void;
  onOpenProviderTerminal?: (provider: string, profile: string) => void;
  onRegisterProvider: (provider: string) => void;
}

export function UsageDiagnosticsPanel({
  accounts,
  monitorByAccount,
  unconfiguredProviders,
  now,
  onRefreshAccount,
  onRequestLogin,
  onOpenProviderTerminal,
  onRegisterProvider,
}: UsageDiagnosticsPanelProps) {
  const { t } = useTranslation();
  const [maestroStatus, setMaestroStatus] = useState<MaestroStatus | null>(null);
  const [maestroError, setMaestroError] = useState('');
  const [maestroLoading, setMaestroLoading] = useState(true);
  const [maestroRevision, setMaestroRevision] = useState(0);

  useEffect(() => {
    let active = true;
    const loadMaestro = async () => {
      setMaestroLoading(true);
      setMaestroError('');
      try {
        const status = await nexus.getMaestroStatus();
        if (active) setMaestroStatus(status || null);
      } catch (reason) {
        if (active) {
          setMaestroStatus(null);
          setMaestroError(reason instanceof Error ? reason.message : String(reason));
        }
      } finally {
        if (active) setMaestroLoading(false);
      }
    };
    void loadMaestro();
    return () => {
      active = false;
    };
  }, [maestroRevision]);

  const at = now ?? Date.now();

  const observations = useMemo(
    () => collectObservations(accounts, monitorByAccount, at),
    [accounts, monitorByAccount, at],
  );

  const problems = useMemo(
    () =>
      deriveDiagnosticsProblems({
        accounts,
        monitorByAccount,
        maestroStatus,
        maestroError,
        installedWithoutAccounts: unconfiguredProviders,
        now: at,
      }),
    [accounts, monitorByAccount, maestroStatus, maestroError, unconfiguredProviders, at],
  );

  const stateProblems = problems.filter(
    (problem) => problem.kind === 'monitorCritical' || problem.kind === 'staleObservation',
  );

  const liveSummary = accounts.length
    ? t('usage.diagnostics.problemSummary', { count: problems.length })
    : t('usage.diagnostics.noAccounts');

  const handleAction = (action: DiagnosticAction) => {
    switch (action.kind) {
      case 'refresh':
        onRefreshAccount(action.provider, action.profile);
        break;
      case 'login': {
        const target = accounts.find((account) => account.id === action.accountId);
        if (target) onRequestLogin(target);
        break;
      }
      case 'terminal':
        onOpenProviderTerminal?.(action.provider, action.profile);
        break;
      case 'register':
        onRegisterProvider(action.provider);
        break;
    }
  };

  const renderProblemAction = (problem: DiagnosticProblem) => {
    const action = problem.action;
    if (!action) return null;
    const labels: Record<DiagnosticAction['kind'], string> = {
      refresh: t('usage.diagnostics.actions.refresh'),
      login: t('usage.diagnostics.actions.login'),
      terminal: t('usage.diagnostics.actions.terminal'),
      register: t('usage.diagnostics.actions.register'),
    };
    return (
      <Button size="sm" tone="ghost" onClick={() => handleAction(action)}>
        {labels[action.kind]}
      </Button>
    );
  };

  const renderProblemReason = (problem: DiagnosticProblem) => {
    const params =
      problem.kind === 'monitorCritical' && !problem.params.reason
        ? { ...problem.params, reason: t('usage.diagnostics.noReason') }
        : problem.params;
    return t(problem.reasonKey, params);
  };

  const renderProblem = (problem: DiagnosticProblem) => (
    <li key={problem.id} className={styles.problemRow} data-severity={problem.severity}>
      <div className={styles.problemCopy}>
        <Badge tone={severityTone(problem.severity)}>
          {t(`usage.diagnostics.severity.${problem.severity}`)}
        </Badge>
        <span>{renderProblemReason(problem)}</span>
      </div>
      {renderProblemAction(problem)}
    </li>
  );

  const maestroCapabilities = maestroStatus?.capabilities;
  const lastCheck = maestroStatus?.last_check ? Date.parse(maestroStatus.last_check) : NaN;
  const noEvidenceAtAll =
    accounts.length === 0 &&
    unconfiguredProviders.length === 0 &&
    !maestroError &&
    maestroStatus?.available !== false;

  return (
    <section className={styles.root} aria-label={t('usage.diagnostics.title')}>
      <header className={styles.header}>
        <div className={styles.heading}>
          <Stethoscope size={18} aria-hidden="true" />
          <div>
            <h2>{t('usage.diagnostics.title')}</h2>
            <p>{t('usage.diagnostics.subtitle')}</p>
          </div>
        </div>
        <p role="status" aria-live="polite" className={styles.liveStatus}>
          {liveSummary}
        </p>
      </header>

      <section className={styles.block} aria-labelledby="usage-diagnostics-collection">
        <h3 id="usage-diagnostics-collection">{t('usage.diagnostics.collectionTitle')}</h3>
        {observations.length === 0 ? (
          <p className={styles.empty}>{t('usage.diagnostics.collectionEmpty')}</p>
        ) : (
          <ul className={styles.observations}>
            {observations.map((observation) => {
              const stale = isObservationStale(observation, at);
              return (
                <li
                  key={observation.accountId}
                  className={styles.observationRow}
                  data-state={observation.state.toLowerCase()}
                >
                  <span className={styles.observationId}>
                    <strong>{observation.provider}</strong>
                    <small>{observation.profile}</small>
                  </span>
                  <span className={styles.observationSource}>
                    {observation.source
                      ? t('usage.diagnostics.sourceValue', { source: observation.source })
                      : t('usage.diagnostics.noSource')}
                  </span>
                  <Badge tone={stateTone(observation.state)}>
                    {translateStatus(observation.state)}
                  </Badge>
                  {observation.ageSeconds !== null ? (
                    <span className={styles.observationAge} data-stale={stale ? 'true' : 'false'}>
                      {t(observationAgeKey(observation.ageSeconds), {
                        count: observation.ageSeconds,
                      })}
                    </span>
                  ) : (
                    <span className={styles.observationAge} data-absent="true">
                      {t('usage.diagnostics.noObservationTime')}
                    </span>
                  )}
                  <IconButton
                    label={t('usage.diagnostics.refreshObservation', {
                      provider: observation.provider,
                      profile: observation.profile,
                    })}
                    onClick={() => onRefreshAccount(observation.provider, observation.profile)}
                  >
                    <RefreshCw size={14} aria-hidden="true" />
                  </IconButton>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <section className={styles.block} aria-labelledby="usage-diagnostics-states">
        <h3 id="usage-diagnostics-states">{t('usage.diagnostics.statesTitle')}</h3>
        {accounts.length === 0 ? (
          <p className={styles.empty}>{t('usage.diagnostics.noAccounts')}</p>
        ) : stateProblems.length === 0 ? (
          <p className={styles.empty}>{t('usage.diagnostics.statesEmpty')}</p>
        ) : (
          <ul className={styles.problems}>{stateProblems.map(renderProblem)}</ul>
        )}
      </section>

      <section className={styles.block} aria-labelledby="usage-diagnostics-maestro">
        <h3 id="usage-diagnostics-maestro">{t('usage.diagnostics.maestroTitle')}</h3>
        <div className={styles.maestroHead}>
          {maestroStatus ? (
            <Badge tone={maestroStatus.available ? 'success' : 'warning'}>
              {maestroStatus.available
                ? t('usage.diagnostics.maestroAvailable')
                : t('usage.diagnostics.maestroUnavailable')}
            </Badge>
          ) : null}
          <IconButton
            label={t('usage.diagnostics.maestroRefresh')}
            disabled={maestroLoading}
            onClick={() => setMaestroRevision((value) => value + 1)}
          >
            <RefreshCw size={14} className={maestroLoading ? 'nx-spin' : ''} aria-hidden="true" />
          </IconButton>
        </div>
        {maestroLoading ? (
          <p role="status" className={styles.note}>
            {t('usage.diagnostics.maestroChecking')}
          </p>
        ) : null}
        {!maestroLoading && maestroError ? (
          <p role="alert" className={styles.error}>
            {t('usage.diagnostics.maestroError')}: {maestroError}
          </p>
        ) : null}
        {!maestroLoading && !maestroError && maestroStatus ? (
          maestroStatus.available ? (
            <dl className={styles.maestroFacts}>
              <div className={styles.maestroFact}>
                <dt>{t('usage.diagnostics.maestroMode')}</dt>
                <dd>{maestroStatus.mode}</dd>
              </div>
              {maestroCapabilities?.version ? (
                <div className={styles.maestroFact}>
                  <dt>{t('usage.diagnostics.maestroVersion')}</dt>
                  <dd>{maestroCapabilities.version}</dd>
                </div>
              ) : null}
              {maestroCapabilities?.modes && maestroCapabilities.modes.length > 0 ? (
                <div className={styles.maestroFact}>
                  <dt>{t('usage.diagnostics.maestroModes')}</dt>
                  <dd>{maestroCapabilities.modes.join(', ')}</dd>
                </div>
              ) : null}
              {maestroCapabilities?.skills ? (
                <div className={styles.maestroFact}>
                  <dt>{t('usage.diagnostics.maestroSkills')}</dt>
                  <dd>{asArray(maestroCapabilities.skills).length}</dd>
                </div>
              ) : null}
              {maestroCapabilities?.gates ? (
                <div className={styles.maestroFact}>
                  <dt>{t('usage.diagnostics.maestroGates')}</dt>
                  <dd>{maestroCapabilities.gates.length}</dd>
                </div>
              ) : null}
              {Number.isFinite(lastCheck) ? (
                <div className={styles.maestroFact}>
                  <dt>{t('usage.diagnostics.maestroLastCheck')}</dt>
                  <dd>
                    <time dateTime={maestroStatus.last_check}>
                      {new Date(lastCheck).toLocaleString(undefined, {
                        dateStyle: 'short',
                        timeStyle: 'short',
                      })}
                    </time>
                  </dd>
                </div>
              ) : null}
            </dl>
          ) : (
            <p className={styles.note}>{t('usage.diagnostics.maestroUnavailableHint')}</p>
          )
        ) : null}
      </section>

      <section className={styles.block} aria-labelledby="usage-diagnostics-problems">
        <h3 id="usage-diagnostics-problems">{t('usage.diagnostics.problemsTitle')}</h3>
        {noEvidenceAtAll ? (
          <p className={styles.empty}>{t('usage.diagnostics.noAccounts')}</p>
        ) : problems.length === 0 ? (
          <p className={styles.empty}>{t('usage.diagnostics.noProblems')}</p>
        ) : (
          <ul className={styles.problems}>{problems.map(renderProblem)}</ul>
        )}
      </section>

      {observations.length === 0 && accounts.length === 0 ? (
        <EmptyState icon={<Stethoscope size={18} />} title={t('usage.diagnostics.noAccounts')} />
      ) : null}
    </section>
  );
}
