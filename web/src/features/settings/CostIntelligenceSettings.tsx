import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, InlineAlert, Input, Switch } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { CostPreferences, ExternalClaudeSummary } from '../../types';
import styles from './CostIntelligenceSettings.module.scss';
import { downloadAuthenticatedCsv } from '../../lib/downloadAuthenticatedCsv';
import { isLocalTranscriptOrigin } from './externalClaudeAvailability';

const numberValue = (raw: string, fallback: number): number => {
  const value = Number.parseFloat(raw);
  return Number.isFinite(value) ? value : fallback;
};

const dollarsToMicroUSD = (raw: string, fallbackMicroUSD: number): number => {
  const fallbackUSD = fallbackMicroUSD / 1_000_000;
  return Math.max(0, Math.round(numberValue(raw, fallbackUSD) * 1_000_000));
};

const microUSDToDollars = (value: number): string => {
  if (!value) return '0';
  return String(value / 1_000_000);
};

export const CostIntelligenceSettings: React.FC = () => {
  const { t } = useTranslation();
  const [prefs, setPrefs] = useState<CostPreferences | null>(null);
  const [draft, setDraft] = useState({
    windowDays: '',
    softUSD: '',
    hardUSD: '',
    emergencyUSD: '',
    anomalyMultiplier: '',
    anomalyMinimumSamples: '',
    anomalyMinUSD: '',
  });
  const [saving, setSaving] = useState(false);
  const [saved, setSaved] = useState(false);
  const [external, setExternal] = useState<ExternalClaudeSummary | null>(null);
  const [externalBusy, setExternalBusy] = useState(false);
  const [externalResult, setExternalResult] = useState('');
  const canScanExternal =
    typeof window !== 'undefined' && isLocalTranscriptOrigin(window.location.hostname);
  const [error, setError] = useState('');

  const applyPreferences = useCallback((next: CostPreferences) => {
    setPrefs(next);
    setDraft({
      windowDays: String(next.window_days),
      softUSD: microUSDToDollars(next.soft_limit_microusd),
      hardUSD: microUSDToDollars(next.hard_limit_microusd),
      emergencyUSD: microUSDToDollars(next.emergency_limit_microusd),
      anomalyMultiplier: String(next.anomaly_multiplier),
      anomalyMinimumSamples: String(next.anomaly_minimum_samples),
      anomalyMinUSD: microUSDToDollars(next.anomaly_min_cost_microusd),
    });
  }, []);

  const refresh = useCallback(async () => {
    try {
      setError('');
      applyPreferences(await nexus.getCostPreferences());
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [applyPreferences]);

  useEffect(() => {
    void refresh();
    void nexus
      .getExternalClaudeUsage()
      .then(setExternal)
      .catch(() => setExternal(null));
  }, [refresh]);

  const patch = (next: Partial<CostPreferences>) => {
    setPrefs((current) => (current ? { ...current, ...next } : current));
    setSaved(false);
  };

  const save = async () => {
    if (!prefs) return;

    const soft = dollarsToMicroUSD(draft.softUSD, prefs.soft_limit_microusd);
    const hard = dollarsToMicroUSD(draft.hardUSD, prefs.hard_limit_microusd);
    const emergency = dollarsToMicroUSD(draft.emergencyUSD, prefs.emergency_limit_microusd);
    const configured = [soft, hard, emergency].filter((value) => value > 0);
    if (configured.some((value, index) => index > 0 && value <= configured[index - 1])) {
      setError(
        t(
          'settings.costInvalidBudgets',
          'Configured limits must increase from soft to hard to emergency. Use 0 to disable a threshold.',
        ),
      );
      return;
    }

    const windowDays = Math.trunc(numberValue(draft.windowDays, prefs.window_days));
    const multiplier = numberValue(draft.anomalyMultiplier, prefs.anomaly_multiplier);
    const minimumSamples = Math.trunc(
      numberValue(draft.anomalyMinimumSamples, prefs.anomaly_minimum_samples),
    );
    if (windowDays < 1 || windowDays > 365 || multiplier < 1 || minimumSamples < 3) {
      setError(
        t(
          'settings.costInvalidPolicy',
          'Window must be 1–365 days, anomaly multiplier at least 1, and history at least 3 samples.',
        ),
      );
      return;
    }

    setSaving(true);
    setSaved(false);
    try {
      setError('');
      const next = await nexus.putCostPreferences({
        ...prefs,
        window_days: windowDays,
        soft_limit_microusd: soft,
        hard_limit_microusd: hard,
        emergency_limit_microusd: emergency,
        anomaly_multiplier: multiplier,
        anomaly_minimum_samples: minimumSamples,
        anomaly_min_cost_microusd: dollarsToMicroUSD(
          draft.anomalyMinUSD,
          prefs.anomaly_min_cost_microusd,
        ),
      });
      applyPreferences(next);
      setSaved(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const clearLocalClaude = async () => {
    if (
      !window.confirm(
        t('settings.externalClaudeConfirmClear', 'Excluir os metadados de uso externo importados?'),
      )
    ) {
      return;
    }
    setExternalBusy(true);
    setExternalResult('');
    try {
      await nexus.clearExternalClaudeUsage();
      setExternal(await nexus.getExternalClaudeUsage());
      setExternalResult(t('settings.externalClaudeCleared', 'Metadados externos excluídos.'));
    } catch (err) {
      setExternalResult(err instanceof Error ? err.message : String(err));
    } finally {
      setExternalBusy(false);
    }
  };

  // Consent is scoped to this click only. No background scanning is enabled.
  const scanLocalClaude = async () => {
    setExternalBusy(true);
    setExternalResult('');
    try {
      const result = await nexus.scanExternalClaudeUsage();
      setExternal(result.summary);
      setExternalResult(
        t('settings.externalClaudeScanResult', {
          defaultValue:
            '{{messages}} registros processados nesta leitura. Histórico separado do custo real.',
          messages: result.scan.messages,
        }),
      );
    } catch (err) {
      setExternalResult(err instanceof Error ? err.message : String(err));
    } finally {
      setExternalBusy(false);
    }
  };

  if (!prefs) {
    return (
      <Card className={styles.root}>
        {error ? (
          <InlineAlert tone="danger" title={t('common.error', 'Error')}>
            {error}
          </InlineAlert>
        ) : (
          <p className={styles.empty}>
            {t('settings.costUnavailable', 'Cost Intelligence preferences unavailable.')}
          </p>
        )}
      </Card>
    );
  }

  return (
    <Card className={styles.root}>
      <div className={styles.header}>
        <strong>{t('settings.costIntelligence', 'Cost Intelligence')}</strong>
        <p>
          {t(
            'settings.costIntelligenceDescription',
            'Local attribution, budget warnings and anomaly detection. Unknown provider cost remains UNKNOWN, never zero.',
          )}
        </p>
      </div>

      {error ? (
        <InlineAlert tone="danger" title={t('common.error', 'Error')}>
          {error}
        </InlineAlert>
      ) : null}
      {saved ? (
        <InlineAlert
          tone="success"
          title={t('settings.costSaved', 'Cost Intelligence preferences saved.')}
        >
          {t(
            'settings.costSavedDescription',
            'The policy applies to new local budget assessments.',
          )}
        </InlineAlert>
      ) : null}

      <Switch
        checked={prefs.enabled}
        onChange={(checked) => patch({ enabled: checked })}
        label={t('settings.costEnabled', 'Cost Intelligence enabled')}
        description={t(
          'settings.costEnabledDescription',
          'Keep local execution attribution and evaluate configured cost policy.',
        )}
      />

      <section className={styles.section} aria-label={t('settings.costBudgets', 'Budgets')}>
        <h3>{t('settings.costBudgets', 'Budgets')}</h3>
        <p className={styles.note}>
          {t(
            'settings.costBudgetNote',
            'Values are USD for the selected window. 0 disables a threshold. Hard and emergency levels require confirmation before new expensive work; they do not kill an in-flight process.',
          )}
        </p>
        <div className={styles.grid}>
          <label>
            <span>{t('settings.costWindowDays', 'Window (days)')}</span>
            <Input
              value={draft.windowDays}
              onChange={(value) => setDraft((current) => ({ ...current, windowDays: value }))}
              inputMode="numeric"
            />
          </label>
          <label>
            <span>{t('settings.costSoftUSD', 'Soft limit (USD)')}</span>
            <Input
              value={draft.softUSD}
              onChange={(value) => setDraft((current) => ({ ...current, softUSD: value }))}
              inputMode="decimal"
            />
          </label>
          <label>
            <span>{t('settings.costHardUSD', 'Hard limit (USD)')}</span>
            <Input
              value={draft.hardUSD}
              onChange={(value) => setDraft((current) => ({ ...current, hardUSD: value }))}
              inputMode="decimal"
            />
          </label>
          <label>
            <span>{t('settings.costEmergencyUSD', 'Emergency limit (USD)')}</span>
            <Input
              value={draft.emergencyUSD}
              onChange={(value) => setDraft((current) => ({ ...current, emergencyUSD: value }))}
              inputMode="decimal"
            />
          </label>
        </div>
      </section>

      <section
        className={styles.section}
        aria-label={t('settings.costAnomalyPolicy', 'Anomaly detection')}
      >
        <h3>{t('settings.costAnomalyPolicy', 'Anomaly detection')}</h3>
        <p className={styles.note}>
          {t(
            'settings.costAnomalyNote',
            'Only measured costs with the same provider, profile and cost evidence kind are compared.',
          )}
        </p>
        <div className={styles.grid}>
          <label>
            <span>{t('settings.costAnomalyMultiplier', 'Baseline multiplier')}</span>
            <Input
              value={draft.anomalyMultiplier}
              onChange={(value) =>
                setDraft((current) => ({ ...current, anomalyMultiplier: value }))
              }
              inputMode="decimal"
            />
          </label>
          <label>
            <span>{t('settings.costAnomalySamples', 'Minimum historical samples')}</span>
            <Input
              value={draft.anomalyMinimumSamples}
              onChange={(value) =>
                setDraft((current) => ({ ...current, anomalyMinimumSamples: value }))
              }
              inputMode="numeric"
            />
          </label>
          <label>
            <span>{t('settings.costAnomalyMinUSD', 'Minimum anomaly cost (USD)')}</span>
            <Input
              value={draft.anomalyMinUSD}
              onChange={(value) => setDraft((current) => ({ ...current, anomalyMinUSD: value }))}
              inputMode="decimal"
            />
          </label>
        </div>
      </section>

      <section
        className={styles.section}
        aria-label={t('settings.externalClaude', 'Uso externo do Claude Code')}
      >
        <h3>{t('settings.externalClaude', 'Uso externo do Claude Code')}</h3>
        <p>
          {t(
            'settings.externalClaudeDescription',
            'Leitura manual dos metadados da máquina que executa o servidor Nexus (não necessariamente este navegador). Não envia prompts, não estima custos da assinatura e não soma os tokens ao ledger.',
          )}
        </p>
        <Button onClick={() => void scanLocalClaude()} disabled={externalBusy || !canScanExternal}>
          {externalBusy
            ? t('settings.externalClaudeScanning', 'Lendo registros…')
            : t('settings.externalClaudeConsent', 'Autorizar uma leitura local agora')}
        </Button>
        {!canScanExternal ? (
          <p className={styles.note}>
            {t(
              'settings.externalClaudeLocalOnly',
              'A importação requer acesso local no computador que executa o Nexus. Em túnel ou rede remota, apenas a consulta e a exclusão dos metadados ficam disponíveis.',
            )}
          </p>
        ) : null}
        {external ? (
          <Button onClick={() => void clearLocalClaude()} disabled={externalBusy}>
            {t('settings.externalClaudeClear', 'Limpar metadados importados')}
          </Button>
        ) : null}
        {external ? (
          <p className={styles.externalSummary}>
            {t('settings.externalClaudeObserved', {
              defaultValue:
                '{{messages}} mensagens · {{tokens}} tokens observados (fonte externa, não atribuída)',
              messages: external.messages,
              tokens:
                external.input_tokens +
                external.output_tokens +
                external.cache_read_tokens +
                external.cache_write_tokens,
            })}
          </p>
        ) : null}
        <p role="status" aria-live="polite">
          {externalResult}
        </p>
      </section>
      <div className={styles.actions}>
        <Button
          onClick={() =>
            void downloadAuthenticatedCsv(
              '/api/v1/cost/observations.csv?days=30',
              'nexus-cost-observations.csv',
            ).catch((err) => setError(err instanceof Error ? err.message : String(err)))
          }
        >
          {t('settings.costExport', 'Exportar até 1.000 observações (CSV)')}
        </Button>
        <Button tone="brand" onClick={() => void save()} disabled={saving}>
          {saving
            ? t('settings.costSaving', 'Saving…')
            : t('settings.costSave', 'Save Cost Intelligence')}
        </Button>
      </div>
    </Card>
  );
};
