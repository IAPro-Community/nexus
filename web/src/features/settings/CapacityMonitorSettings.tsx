import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Activity, BellRing, Check, Clock, Gauge, RefreshCw, ShieldAlert } from 'lucide-react';
import type { CapacityMonitorStatus, CapacityPreferences } from '../../types';
import { Badge, Button, Card, InlineAlert, Input, Switch } from '../../design-system';
import { nexus } from '../../nexus/api';
import { validateCapacityThresholds } from './capacityMonitorSettingsModel';
import styles from './CapacityMonitorSettings.module.scss';

const DEFAULT_PREFS: CapacityPreferences = {
  enabled: true,
  background_enabled: false,
  warning_threshold: 20,
  critical_threshold: 10,
  exhausted_threshold: 0,
  recovery_threshold: 25,
  reminder_interval_seconds: 3600,
  quiet_hours_enabled: false,
  quiet_hours_start_utc: '22:00',
  quiet_hours_end_utc: '08:00',
  exhausted_bypass_quiet_hours: true,
  browser_notifications_enabled: true,
  native_notifications_enabled: false,
  lock_screen_safe_copy: true,
};

export const CapacityMonitorSettings: React.FC = () => {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [status, setStatus] = useState<CapacityMonitorStatus | null>(null);
  const [prefs, setPrefs] = useState<CapacityPreferences>(DEFAULT_PREFS);
  const [error, setError] = useState<string>('');
  const [success, setSuccess] = useState(false);
  const [validationError, setValidationError] = useState<string>('');

  const loadData = async () => {
    try {
      setLoading(true);
      setError('');
      const [fetchedStatus, fetchedPrefs] = await Promise.all([
        nexus.getCapacityMonitorStatus().catch(() => null),
        nexus.getCapacityMonitorPreferences().catch(() => DEFAULT_PREFS),
      ]);
      if (fetchedStatus) setStatus(fetchedStatus);
      if (fetchedPrefs) setPrefs(fetchedPrefs);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void loadData();
  }, []);

  const handleSave = async () => {
    const valErr = validateCapacityThresholds(prefs);
    if (valErr) {
      setValidationError(valErr);
      return;
    }
    setValidationError('');
    setSaving(true);
    setSuccess(false);
    setError('');
    try {
      const updated = await nexus.updateCapacityMonitorPreferences(prefs);
      setPrefs(updated);
      setSuccess(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSaving(false);
    }
  };

  const updateThreshold = (field: keyof CapacityPreferences, valStr: string) => {
    const num = parseFloat(valStr);
    if (!Number.isNaN(num)) {
      setPrefs((prev) => ({ ...prev, [field]: num }));
    }
  };

  return (
    <Card className={styles.container}>
      <div className={styles.section}>
        <div className={styles.sectionHeader}>
          <div className={styles.titleGroup}>
            <Gauge size={18} />
            <h3 className={styles.sectionTitle}>
              {t('capacityMonitor.title', 'Monitor de Capacidade & Cotas')}
            </h3>
          </div>
          <p className={styles.sectionSubtitle}>
            {t(
              'capacityMonitor.description',
              'Acompanhamento adaptativo do consumo de cotas de IA, alertas proativos e sugestões de substituição.',
            )}
          </p>
        </div>

        {status && (
          <div className={styles.statusRow}>
            <span>
              {t('capacityMonitor.daemonStatus', 'Status do Daemon')}:{' '}
              <strong>
                {status.running
                  ? t('capacityMonitor.running', 'Em execução')
                  : t('capacityMonitor.stopped', 'Parado')}
              </strong>
            </span>
            <div className={styles.badgeGroup}>
              <Badge tone={status.is_leader ? 'brand' : 'default'}>
                {status.is_leader
                  ? t('capacityMonitor.leader', 'Líder Ativo')
                  : t('capacityMonitor.standby', 'Standby')}
              </Badge>
              {status.degraded && (
                <Badge tone="warning">{t('capacityMonitor.degraded', 'Degradado')}</Badge>
              )}
            </div>
          </div>
        )}
      </div>

      <div className={styles.section}>
        <Switch
          checked={prefs.enabled}
          onChange={(checked) => setPrefs((prev) => ({ ...prev, enabled: checked }))}
          label={t('capacityMonitor.enableSwitch', 'Habilitar Monitoramento Contínuo')}
          description={t(
            'capacityMonitor.enableSwitchDesc',
            'Sondagem adaptativa e alertas locais quando o limite de tokens/requisições estiver próximo do fim.',
          )}
        />

        <Switch
          checked={prefs.background_enabled}
          onChange={(checked) => setPrefs((prev) => ({ ...prev, background_enabled: checked }))}
          label={t('capacityMonitor.backgroundSwitch', 'Permitir Execução em Background (Linux)')}
          description={t(
            'capacityMonitor.backgroundSwitchDesc',
            'Instala um serviço systemd user-scoped para monitorar cotas mesmo com a interface web fechada. Requer consentimento explícito.',
          )}
        />
      </div>

      <div className={styles.section}>
        <div className={styles.sectionHeader}>
          <h4 className={styles.sectionTitle}>
            {t('capacityMonitor.thresholdsTitle', 'Limiares de Alerta (%)')}
          </h4>
          <p className={styles.sectionSubtitle}>
            {t(
              'capacityMonitor.thresholdsDesc',
              'Percentual de cota restante para acionar cada estado de aviso.',
            )}
          </p>
        </div>

        <div className={styles.gridThree}>
          <div className={styles.inputGroup}>
            <label className={styles.inputLabel}>
              {t('capacityMonitor.warningThreshold', 'Aviso (Warning %)')}
            </label>
            <Input
              type="number"
              value={String(prefs.warning_threshold)}
              onChange={(val) => updateThreshold('warning_threshold', val)}
            />
          </div>
          <div className={styles.inputGroup}>
            <label className={styles.inputLabel}>
              {t('capacityMonitor.criticalThreshold', 'Crítico (Critical %)')}
            </label>
            <Input
              type="number"
              value={String(prefs.critical_threshold)}
              onChange={(val) => updateThreshold('critical_threshold', val)}
            />
          </div>
          <div className={styles.inputGroup}>
            <label className={styles.inputLabel}>
              {t('capacityMonitor.recoveryThreshold', 'Recuperação (Recovery %)')}
            </label>
            <Input
              type="number"
              value={String(prefs.recovery_threshold)}
              onChange={(val) => updateThreshold('recovery_threshold', val)}
            />
          </div>
        </div>
      </div>

      <div className={styles.section}>
        <div className={styles.sectionHeader}>
          <h4 className={styles.sectionTitle}>
            {t('capacityMonitor.quietHoursTitle', 'Horário de Silêncio (Quiet Hours)')}
          </h4>
          <p className={styles.sectionSubtitle}>
            {t(
              'capacityMonitor.quietHoursDesc',
              'Silencia alertas não críticos durante períodos de descanso.',
            )}
          </p>
        </div>

        <Switch
          checked={prefs.quiet_hours_enabled}
          onChange={(checked) => setPrefs((prev) => ({ ...prev, quiet_hours_enabled: checked }))}
          label={t('capacityMonitor.quietHoursToggle', 'Ativar Horário de Silêncio')}
        />

        {prefs.quiet_hours_enabled && (
          <div className={styles.gridTwo}>
            <div className={styles.inputGroup}>
              <label className={styles.inputLabel}>
                {t('capacityMonitor.quietHoursStart', 'Início (UTC)')}
              </label>
              <Input
                value={prefs.quiet_hours_start_utc}
                onChange={(val) => setPrefs((prev) => ({ ...prev, quiet_hours_start_utc: val }))}
                placeholder="22:00"
              />
            </div>
            <div className={styles.inputGroup}>
              <label className={styles.inputLabel}>
                {t('capacityMonitor.quietHoursEnd', 'Fim (UTC)')}
              </label>
              <Input
                value={prefs.quiet_hours_end_utc}
                onChange={(val) => setPrefs((prev) => ({ ...prev, quiet_hours_end_utc: val }))}
                placeholder="08:00"
              />
            </div>
          </div>
        )}

        <Switch
          checked={prefs.exhausted_bypass_quiet_hours}
          onChange={(checked) =>
            setPrefs((prev) => ({ ...prev, exhausted_bypass_quiet_hours: checked }))
          }
          label={t(
            'capacityMonitor.exhaustedBypassToggle',
            'Permitir alertas de esgotamento total durante silêncio',
          )}
          description={t(
            'capacityMonitor.exhaustedBypassDesc',
            'Alertas de cota 100% esgotada ainda serão emitidos.',
          )}
        />
      </div>

      <div className={styles.section}>
        <div className={styles.sectionHeader}>
          <h4 className={styles.sectionTitle}>
            {t('capacityMonitor.deliveryTitle', 'Canais de Entrega & Privacidade')}
          </h4>
        </div>

        <Switch
          checked={prefs.browser_notifications_enabled}
          onChange={(checked) =>
            setPrefs((prev) => ({ ...prev, browser_notifications_enabled: checked }))
          }
          label={t('capacityMonitor.browserNotify', 'Notificações no Navegador')}
        />

        <Switch
          checked={prefs.native_notifications_enabled}
          onChange={(checked) =>
            setPrefs((prev) => ({ ...prev, native_notifications_enabled: checked }))
          }
          label={t('capacityMonitor.nativeNotify', 'Notificações Nativas do Sistema Operacional')}
        />

        <Switch
          checked={prefs.lock_screen_safe_copy}
          onChange={(checked) => setPrefs((prev) => ({ ...prev, lock_screen_safe_copy: checked }))}
          label={t(
            'capacityMonitor.lockScreenSafe',
            'Ocultar Detalhes Sensíveis na Tela Bloqueada',
          )}
          description={t(
            'capacityMonitor.lockScreenSafeDesc',
            'Não exibe nomes de projetos, modelos ou chaves de API em notificações do sistema.',
          )}
        />
      </div>

      {validationError && (
        <InlineAlert tone="danger" title={t('common.error', 'Erro de validação')}>
          {validationError}
        </InlineAlert>
      )}

      {error && (
        <InlineAlert tone="danger" title={t('common.error', 'Erro ao salvar configurações')}>
          {error}
        </InlineAlert>
      )}

      {success && (
        <InlineAlert tone="success" title={t('common.success', 'Configurações salvas')}>
          {t(
            'capacityMonitor.saveSuccess',
            'Preferências do monitor de capacidade atualizadas com sucesso.',
          )}
        </InlineAlert>
      )}

      <div className={styles.actionsRow}>
        <Button
          tone="brand"
          onClick={() => void handleSave()}
          disabled={saving || loading}
          aria-label={t('common.save', 'Salvar Configurações')}
        >
          {saving ? <RefreshCw size={14} className="nx-spin" /> : <Check size={14} />}
          {saving ? t('common.saving', 'Salvando…') : t('common.save', 'Salvar Configurações')}
        </Button>
      </div>
    </Card>
  );
};
