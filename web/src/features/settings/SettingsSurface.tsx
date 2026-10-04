import React, { useEffect, useMemo, useState } from 'react';
import {
  Accessibility,
  ArrowUpCircle,
  Bell,
  Check,
  ChevronDown,
  ChevronRight,
  Gauge,
  Globe,
  MonitorCog,
  MousePointer2,
  Palette,
  PanelsTopLeft,
  RefreshCw,
  RotateCcw,
  Sparkles,
  Type,
} from 'lucide-react';
import {
  Badge,
  Button,
  Card,
  InlineAlert,
  Input,
  Segmented,
  Switch,
  THEME_PRESETS,
  getThemePresetPalette,
} from '../../design-system';
import { useTheme, type ThemeDensity, type ThemePresetKey } from '../../design-system';
import { useWorkspace } from '../../workspace/WorkspaceProvider';
import { nexus } from '../../nexus/api';
import type { IntelligenceMode, IntelligenceStatus, ProviderAccount } from '../../types';
import { useTranslation } from 'react-i18next';
import { asArray } from '../../lib/safeArray';
import { intelligenceCLIProfiles } from './intelligenceProfiles';
import { IntelligenceProviderCombo } from './IntelligenceProviderCombo';
import {
  loadNotificationPrefs,
  saveNotificationPrefs,
  type NotificationPrefs,
} from '../../notifications/notificationPrefs';
import { pushNotifications } from '../../notifications/PushNotificationManager';
import { SystemDiagnosticsCard } from './SystemDiagnosticsCard';
import { RemoteAccessTab } from './RemoteAccessTab';
import { ComputerControlSettings } from './ComputerControlSettings';
import { IntelligenceMemoriesPanel } from './IntelligenceMemoriesPanel';
import { CapacityMonitorSettings } from './CapacityMonitorSettings';
import { CostIntelligenceSettings } from './CostIntelligenceSettings';
import { HudSettings } from '../hud/HudSettings';
import { VoiceSettings } from '../voice/VoiceSettings';
import type { SystemDoctorReport } from '../../nexus/api';
import styles from './SettingsSurface.module.scss';

type SettingsTab =
  | 'appearance'
  | 'accessibility'
  | 'updates'
  | 'intelligence'
  | 'capacity'
  | 'hud'
  | 'notifications'
  | 'computer'
  | 'remote';
const SETTINGS_TABS: SettingsTab[] = [
  'appearance',
  'accessibility',
  'updates',
  'intelligence',
  'capacity',
  'hud',
  'notifications',
  'computer',
  'remote',
];

export const SettingsSurface: React.FC<{
  onTour: () => void;
  initialTab?: SettingsTab;
  projectId?: string;
}> = ({ onTour, initialTab, projectId }) => {
  const theme = useTheme();
  const { t } = useTranslation();
  const workspace = useWorkspace();
  const [activeTab, setActiveTab] = useState<SettingsTab>(
    initialTab && SETTINGS_TABS.includes(initialTab) ? initialTab : 'appearance',
  );
  useEffect(() => {
    if (initialTab && SETTINGS_TABS.includes(initialTab)) setActiveTab(initialTab);
  }, [initialTab]);
  const onSettingsTabKeyDown = (event: React.KeyboardEvent<HTMLButtonElement>) => {
    const current = SETTINGS_TABS.indexOf(activeTab);
    if (event.key !== 'ArrowRight' && event.key !== 'ArrowLeft') return;
    event.preventDefault();
    const next =
      (current + (event.key === 'ArrowRight' ? 1 : -1) + SETTINGS_TABS.length) %
      SETTINGS_TABS.length;
    setActiveTab(SETTINGS_TABS[next]);
    document.getElementById(`settings-tab-${SETTINGS_TABS[next]}`)?.focus();
  };

  const [checkingUpdates, setCheckingUpdates] = useState(false);
  const [updateInfo, setUpdateInfo] = useState<{
    nexus_version: string;
    nexus_latest_version?: string;
    nexus_update_available?: boolean;
    nexus_update_error?: string;
    nexus_update_instruction?: string;
    nexus_allows_self_update?: boolean;
    channel?: string;
    installation_method?: string;
    allows_self_update?: boolean;
    maestro_version: string;
    maestro_latest_version?: string;
    maestro_available: boolean;
    update_available?: boolean;
  } | null>(null);
  const [updating, setUpdating] = useState(false);
  const [updateSuccess, setUpdateSuccess] = useState(false);
  const [updateError, setUpdateError] = useState('');
  const [updateResult, setUpdateResult] = useState<{
    maestro_updated: boolean;
    maestro_version: string;
    nexus_version: string;
    error?: string;
  } | null>(null);
  const [doctorReport, setDoctorReport] = useState<SystemDoctorReport | null>(null);
  const [checkingDoctor, setCheckingDoctor] = useState(false);
  const [doctorError, setDoctorError] = useState('');
  const [intelligence, setIntelligence] = useState<IntelligenceStatus | null>(null);
  const [intelligenceDraft, setIntelligenceDraft] = useState<IntelligenceStatus>({
    mode: 'OFF',
    available: false,
  });
  const [intelligenceResources, setIntelligenceResources] = useState<ProviderAccount[]>([]);
  const [intelligenceSaving, setIntelligenceSaving] = useState(false);
  const [intelligenceError, setIntelligenceError] = useState('');
  const [notifyPrefs, setNotifyPrefs] = useState<NotificationPrefs>(() => loadNotificationPrefs());

  const updateNotifyPrefs = (patch: Partial<NotificationPrefs>) => {
    setNotifyPrefs((current) => {
      const next = { ...current, ...patch };
      saveNotificationPrefs(next);
      window.dispatchEvent(new CustomEvent('nexus:notification-prefs'));
      if (patch.notificationsEnabled === true) {
        void pushNotifications.requestPermission().then((perm) => {
          if (perm === 'granted') pushNotifications.confirmEnabled();
        });
      }
      return next;
    });
  };

  const checkUpdates = async () => {
    try {
      setCheckingUpdates(true);
      setUpdateError('');
      const data = await nexus.getSystemUpdates();
      setUpdateInfo(data);
    } catch (error) {
      setUpdateInfo(null);
      setUpdateError(error instanceof Error ? error.message : String(error));
    } finally {
      setCheckingUpdates(false);
    }
  };

  const checkDoctor = async () => {
    try {
      setCheckingDoctor(true);
      setDoctorError('');
      setDoctorReport(await nexus.getSystemDoctor());
    } catch (error) {
      setDoctorReport(null);
      setDoctorError(error instanceof Error ? error.message : String(error));
    } finally {
      setCheckingDoctor(false);
    }
  };

  const loadIntelligence = async () => {
    try {
      setIntelligenceError('');
      const [status, resources] = await Promise.all([
        nexus.getIntelligence(),
        nexus.listResources(),
      ]);
      setIntelligence(status);
      setIntelligenceDraft(status);
      setIntelligenceResources(asArray(resources.accounts));
    } catch (error) {
      setIntelligenceError(error instanceof Error ? error.message : String(error));
    }
  };

  useEffect(() => {
    void checkUpdates();
    void checkDoctor();
    void loadIntelligence();
  }, []);

  const handleUpdate = async () => {
    setUpdating(true);
    setUpdateSuccess(false);
    setUpdateError('');
    setUpdateResult(null);
    try {
      const res = await nexus.performSystemUpdate();
      setUpdateResult(res);
      setUpdateSuccess(true);
      window.dispatchEvent(new CustomEvent('nexus:system-updates'));
      void checkUpdates();
    } catch (error) {
      setUpdateError(error instanceof Error ? error.message : String(error));
    } finally {
      setUpdating(false);
    }
  };

  const cliResources = useMemo(
    () => intelligenceCLIProfiles(intelligenceResources, intelligenceDraft),
    [intelligenceResources, intelligenceDraft],
  );

  const saveIntelligence = async () => {
    setIntelligenceSaving(true);
    setIntelligenceError('');
    try {
      const payload = {
        mode: intelligenceDraft.mode as IntelligenceMode,
        provider: intelligenceDraft.provider,
        profile: intelligenceDraft.profile,
        base_url: intelligenceDraft.base_url,
        model: intelligenceDraft.model,
        api_key_env: intelligenceDraft.api_key_env,
        api_key_file: intelligenceDraft.api_key_file,
      };
      const next = await nexus.updateIntelligence(payload);
      setIntelligence(next);
      setIntelligenceDraft(next);
    } catch (error) {
      setIntelligenceError(error instanceof Error ? error.message : String(error));
    } finally {
      setIntelligenceSaving(false);
    }
  };

  return (
    <div className="nx-surface-scroll">
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">{t('settings.eyebrow')}</span>
          <h1>{t('settings.title')}</h1>
          <p>{t('settings.intro')}</p>
        </div>
      </div>
      {/* 5 Modular Navigation Tabs */}
      <div className="nx-settings-tabs" role="tablist">
        <button
          type="button"
          role="tab"
          id="settings-tab-appearance"
          aria-controls="settings-panel"
          tabIndex={activeTab === 'appearance' ? 0 : -1}
          aria-selected={activeTab === 'appearance'}
          onKeyDown={onSettingsTabKeyDown}
          className={`nx-settings-tab ${activeTab === 'appearance' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('appearance')}
        >
          <Palette size={15} />
          <span>Aparência & Estilo</span>
        </button>
        <button
          type="button"
          role="tab"
          id="settings-tab-accessibility"
          aria-controls="settings-panel"
          tabIndex={activeTab === 'accessibility' ? 0 : -1}
          aria-selected={activeTab === 'accessibility'}
          onKeyDown={onSettingsTabKeyDown}
          className={`nx-settings-tab ${activeTab === 'accessibility' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('accessibility')}
        >
          <Accessibility size={15} />
          <span>Acessibilidade</span>
        </button>
        <button
          type="button"
          role="tab"
          id="settings-tab-updates"
          aria-controls="settings-panel"
          tabIndex={activeTab === 'updates' ? 0 : -1}
          aria-selected={activeTab === 'updates'}
          onKeyDown={onSettingsTabKeyDown}
          className={`nx-settings-tab ${activeTab === 'updates' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('updates')}
        >
          <ArrowUpCircle size={15} />
          <span>Atualizações & Sistema</span>
          {updateInfo?.update_available && (
            <span className={`nx-badge ${styles.updateBadge}`} data-tone="warning">
              Update
            </span>
          )}
        </button>
        <button
          type="button"
          role="tab"
          id="settings-tab-intelligence"
          aria-controls="settings-panel"
          tabIndex={activeTab === 'intelligence' ? 0 : -1}
          aria-selected={activeTab === 'intelligence'}
          onKeyDown={onSettingsTabKeyDown}
          className={`nx-settings-tab ${activeTab === 'intelligence' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('intelligence')}
        >
          <Sparkles size={15} />
          <span>Nexus Intelligence</span>
        </button>
        <button
          type="button"
          role="tab"
          id="settings-tab-capacity"
          aria-controls="settings-panel"
          tabIndex={activeTab === 'capacity' ? 0 : -1}
          aria-selected={activeTab === 'capacity'}
          onKeyDown={onSettingsTabKeyDown}
          className={`nx-settings-tab ${activeTab === 'capacity' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('capacity')}
        >
          <Gauge size={15} />
          <span>{t('settings.capacity', 'Capacity Monitor')}</span>
        </button>
        <button
          type="button"
          role="tab"
          id="settings-tab-hud"
          aria-controls="settings-panel"
          tabIndex={activeTab === 'hud' ? 0 : -1}
          aria-selected={activeTab === 'hud'}
          onKeyDown={onSettingsTabKeyDown}
          className={`nx-settings-tab ${activeTab === 'hud' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('hud')}
        >
          <PanelsTopLeft size={15} />
          <span>Nexus HUD</span>
        </button>
        <button
          type="button"
          role="tab"
          id="settings-tab-notifications"
          aria-controls="settings-panel"
          tabIndex={activeTab === 'notifications' ? 0 : -1}
          aria-selected={activeTab === 'notifications'}
          onKeyDown={onSettingsTabKeyDown}
          className={`nx-settings-tab ${activeTab === 'notifications' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('notifications')}
        >
          <Bell size={15} />
          <span>Notificações</span>
        </button>
        <button
          type="button"
          role="tab"
          id="settings-tab-computer"
          aria-controls="settings-panel"
          tabIndex={activeTab === 'computer' ? 0 : -1}
          aria-selected={activeTab === 'computer'}
          onKeyDown={onSettingsTabKeyDown}
          className={`nx-settings-tab ${activeTab === 'computer' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('computer')}
        >
          <MousePointer2 size={15} />
          <span>{t('settings.computer.title', { defaultValue: 'Computer Control' })}</span>
        </button>
        <button
          type="button"
          role="tab"
          id="settings-tab-remote"
          aria-controls="settings-panel"
          tabIndex={activeTab === 'remote' ? 0 : -1}
          aria-selected={activeTab === 'remote'}
          onKeyDown={onSettingsTabKeyDown}
          className={`nx-settings-tab ${activeTab === 'remote' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('remote')}
        >
          <Globe size={15} />
          <span>{t('settings.remote.title')}</span>
        </button>
      </div>

      <div
        className="nx-settings-grid"
        id="settings-panel"
        role="tabpanel"
        aria-labelledby={`settings-tab-${activeTab}`}
        tabIndex={0}
      >
        {/* Tab 1: Appearance */}
        {activeTab === 'appearance' && (
          <>
            <Card className={`nx-settings-card ${styles.gridFullWidth}`}>
              <div className="nx-settings-card__title">
                <Palette size={17} />
                <div>
                  <strong>{t('settings.theme')}</strong>
                  <small>{t('settings.themeDescription')}</small>
                </div>
              </div>

              <ThemeAccordionSelector theme={theme} />
            </Card>

            <Card className="nx-settings-card">
              <div className="nx-settings-card__title">
                <MonitorCog size={17} />
                <div>
                  <strong>{t('settings.workspace')}</strong>
                  <small>{t('settings.workspaceDescription')}</small>
                </div>
              </div>
              <p className={`nx-muted-copy ${styles.mutedTextSmall}`}>
                Restaura o layout original do projeto ativo, preservando workspaces e chaves
                seguras.
              </p>
              <Button tone="warning" onClick={workspace.reset}>
                <RotateCcw size={14} /> {t('settings.reset')}
              </Button>
            </Card>
          </>
        )}

        {/* Tab 2: Accessibility */}
        {activeTab === 'accessibility' && (
          <>
            <Card className="nx-settings-card">
              <div className="nx-settings-card__title">
                <Type size={17} />
                <div>
                  <strong>{t('settings.fontScale', 'Escala Tipográfica')}</strong>
                  <small>
                    {t(
                      'settings.fontScaleDesc',
                      'Ajuste proporcional do tamanho do texto sem distorcer o layout',
                    )}
                  </small>
                </div>
              </div>
              <div className="nx-font-scale-controls">
                <button
                  type="button"
                  className="nx-font-scale-btn"
                  data-active={(theme.fontScale || 1) <= 0.92}
                  onClick={() => theme.setFontScale(0.9)}
                >
                  A- (90%)
                </button>
                <button
                  type="button"
                  className="nx-font-scale-btn"
                  data-active={Math.abs((theme.fontScale || 1) - 1) < 0.04}
                  onClick={() => theme.setFontScale(1)}
                >
                  Padrão (100%)
                </button>
                <button
                  type="button"
                  className="nx-font-scale-btn"
                  data-active={(theme.fontScale || 1) >= 1.05 && (theme.fontScale || 1) < 1.15}
                  onClick={() => theme.setFontScale(1.1)}
                >
                  A+ (110%)
                </button>
                <button
                  type="button"
                  className="nx-font-scale-btn"
                  data-active={(theme.fontScale || 1) >= 1.15}
                  onClick={() => theme.setFontScale(1.2)}
                >
                  A++ (120%)
                </button>
              </div>
            </Card>

            <Card className="nx-settings-card">
              <div className="nx-settings-card__title">
                <Accessibility size={17} />
                <div>
                  <strong>{t('settings.accessibility')}</strong>
                  <small>{t('settings.accessibilityDescription')}</small>
                </div>
              </div>
              <Segmented
                ariaLabel={t('settings.density')}
                value={theme.density}
                onChange={(value) => theme.setDensity(value as ThemeDensity)}
                options={[
                  { value: 'compact', label: t('settings.compact') },
                  { value: 'comfortable', label: t('settings.comfortable') },
                ]}
              />
              <Switch
                checked={theme.reducedMotion}
                onChange={theme.setReducedMotion}
                label={t('settings.reduceMotion')}
                description={t('settings.reduceMotionDescription')}
              />
              <Button onClick={onTour}>{t('settings.replayTour')}</Button>
            </Card>

            <VoiceSettings />
          </>
        )}

        {/* Tab 3: Updates & System */}
        {activeTab === 'updates' && (
          <>
            <Card className={`nx-settings-card ${styles.gridFullWidth}`}>
              <div className="nx-settings-card__title">
                <ArrowUpCircle size={17} />
                <div>
                  <strong>{t('settings.updates')}</strong>
                  <small>{t('settings.updatesDescription')}</small>
                </div>
              </div>
              {updateInfo && (
                <div className={styles.infoList}>
                  <div className={styles.infoRow}>
                    <span>IAPro Nexus Core:</span>
                    <Badge tone="success">v{updateInfo.nexus_version}</Badge>
                  </div>
                  {updateInfo.nexus_latest_version && (
                    <div className={styles.infoRow}>
                      <span>{t('settings.nexusLatestVersion')}:</span>
                      <Badge tone={updateInfo.nexus_update_available ? 'warning' : 'success'}>
                        v{updateInfo.nexus_latest_version}
                      </Badge>
                    </div>
                  )}
                  {updateInfo.channel && (
                    <div className={styles.infoRow}>
                      <span>Canal de Lançamento:</span>
                      <Badge tone="brand">{updateInfo.channel}</Badge>
                    </div>
                  )}
                  {updateInfo.installation_method && (
                    <div className={styles.infoRow}>
                      <span>Método de Instalação:</span>
                      <Badge tone="info">{updateInfo.installation_method}</Badge>
                    </div>
                  )}
                  <div className={styles.infoRow}>
                    <span>Orquestrador Maestro:</span>
                    <Badge tone={updateInfo.maestro_available ? 'brand' : 'warning'}>
                      v{updateInfo.maestro_version}
                    </Badge>
                  </div>
                  {updateInfo.nexus_update_error ? (
                    <InlineAlert tone="warning" title={t('settings.nexusUpdateUnavailable')}>
                      {updateInfo.nexus_update_error}
                    </InlineAlert>
                  ) : updateInfo.nexus_update_available || updateInfo.update_available ? (
                    <InlineAlert tone="warning" title={t('settings.updateAvailable')}>
                      {t('settings.updateAvailableDescription', {
                        version:
                          updateInfo.nexus_latest_version ||
                          updateInfo.maestro_latest_version ||
                          t('settings.newBuild'),
                      })}
                      {updateInfo.nexus_update_available && (
                        <>
                          {' '}
                          {t('settings.nexusUpdateAction', {
                            action:
                              updateInfo.nexus_update_instruction ||
                              (updateInfo.nexus_allows_self_update ? 'nexus update' : ''),
                          })}
                        </>
                      )}
                    </InlineAlert>
                  ) : (
                    <InlineAlert tone="success" title={t('settings.upToDate')}>
                      {t('settings.upToDateDescription')}
                    </InlineAlert>
                  )}
                </div>
              )}
              {updateError && (
                <InlineAlert tone="danger" title="Status de atualização indisponível">
                  {updateError}
                </InlineAlert>
              )}
              {updateSuccess && updateResult && (
                <InlineAlert
                  tone={updateResult.maestro_updated ? 'success' : 'info'}
                  title={t('maestroControl.updateResult')}
                >
                  {updateResult.maestro_updated
                    ? t('maestroControl.updateMaestroDone', {
                        version: updateResult.maestro_version,
                      })
                    : t('maestroControl.updateMaestroSame', {
                        version: updateResult.maestro_version,
                      })}{' '}
                  {t('maestroControl.nexusBinaryNote', { version: updateResult.nexus_version })}
                </InlineAlert>
              )}
              <div className={styles.btnGroup}>
                <Button onClick={() => void checkUpdates()} disabled={checkingUpdates || updating}>
                  <RefreshCw size={14} className={checkingUpdates ? 'nx-spin' : ''} />
                  {checkingUpdates ? 'Verificando…' : 'Verificar Atualizações Agora'}
                </Button>
                <Button tone="brand" onClick={handleUpdate} disabled={updating}>
                  <RefreshCw size={14} className={updating ? 'nx-spin' : ''} />
                  {updating ? t('settings.updating') : t('settings.update')}
                </Button>
              </div>
            </Card>
            <SystemDiagnosticsCard
              report={doctorReport}
              loading={checkingDoctor}
              error={doctorError}
              onRefresh={() => void checkDoctor()}
            />
          </>
        )}

        {/* Tab 4: Intelligence */}
        {activeTab === 'intelligence' && (
          <Card className={`nx-settings-card ${styles.gridFullWidth}`}>
            <div className="nx-settings-card__title">
              <Sparkles size={17} />
              <div>
                <strong>Nexus Intelligence</strong>
                <small>
                  Planejamento semântico para Composer e Flow. O trabalho direto em terminais
                  continua sem isso.
                </small>
              </div>
            </div>
            <Segmented
              ariaLabel="Intelligence mode"
              value={intelligenceDraft.mode}
              onChange={(value) =>
                setIntelligenceDraft((prev) => ({ ...prev, mode: value as IntelligenceMode }))
              }
              options={[
                { value: 'OFF', label: 'Off' },
                { value: 'CLI', label: 'Coding CLI' },
                { value: 'OPENAI_COMPATIBLE', label: 'API' },
              ]}
            />
            {intelligenceDraft.mode === 'CLI' && (
              <IntelligenceProviderCombo
                accounts={cliResources}
                provider={intelligenceDraft.provider}
                profile={intelligenceDraft.profile}
                model={intelligenceDraft.model}
                onChange={(next) =>
                  setIntelligenceDraft((prev) => ({
                    ...prev,
                    provider: next.provider,
                    profile: next.profile,
                    model: next.model ?? prev.model,
                  }))
                }
              />
            )}
            {intelligenceDraft.mode === 'OPENAI_COMPATIBLE' && (
              <>
                <Input
                  value={intelligenceDraft.base_url || ''}
                  onChange={(value) =>
                    setIntelligenceDraft((prev) => ({ ...prev, base_url: value }))
                  }
                  placeholder="Base URL (OpenAI-compatible)"
                />
                <Input
                  value={intelligenceDraft.api_key_env || ''}
                  onChange={(value) =>
                    setIntelligenceDraft((prev) => ({ ...prev, api_key_env: value }))
                  }
                  placeholder="Environment variable containing API key"
                />
                <Input
                  value={intelligenceDraft.api_key_file || ''}
                  onChange={(value) =>
                    setIntelligenceDraft((prev) => ({ ...prev, api_key_file: value }))
                  }
                  placeholder="Or secret file path"
                />
              </>
            )}
            {intelligenceDraft.mode !== 'OFF' && (
              <Input
                value={intelligenceDraft.model || ''}
                onChange={(value) => setIntelligenceDraft((prev) => ({ ...prev, model: value }))}
                placeholder="Modelo (sobrescrever o padrão)"
              />
            )}
            {intelligence && (
              <InlineAlert
                tone={
                  intelligence.available
                    ? 'success'
                    : intelligence.mode === 'OFF'
                      ? 'info'
                      : 'warning'
                }
                title={
                  intelligence.available
                    ? 'Provedor de Intelligence pronto'
                    : intelligence.mode === 'OFF'
                      ? 'Intelligence desligada'
                      : 'Provedor não pronto'
                }
              >
                {intelligence.error ||
                  (intelligence.mode === 'OFF'
                    ? 'Sessões diretas e WorkPlans manuais continuam funcionando.'
                    : `${intelligence.provider || 'Provider'} ${intelligence.profile || ''}`)}
              </InlineAlert>
            )}
            {intelligenceError && (
              <InlineAlert tone="danger" title="Intelligence configuration error">
                {intelligenceError}
              </InlineAlert>
            )}
            <div className={styles.fieldGroupMargin}>
              <Button
                tone="brand"
                onClick={() => void saveIntelligence()}
                disabled={intelligenceSaving}
              >
                {intelligenceSaving ? 'Saving…' : 'Save Intelligence'}
              </Button>
            </div>
            <IntelligenceMemoriesPanel />
          </Card>
        )}

        {/* Tab 5: Notifications */}
        {activeTab === 'notifications' && (
          <Card className={`nx-settings-card ${styles.gridFullWidth}`}>
            <div className="nx-settings-card__title">
              <Bell size={17} />
              <div>
                <strong>{t('settings.notifications', 'Notificações')}</strong>
                <small>
                  {t(
                    'settings.notificationsDescription',
                    'Avisos in-app e do navegador quando um agente pergunta ou conclui. Título e marcadores nas janelas continuam ativos.',
                  )}
                </small>
              </div>
            </div>
            <Switch
              checked={notifyPrefs.notificationsEnabled}
              onChange={(checked) => updateNotifyPrefs({ notificationsEnabled: checked })}
              label={t('settings.notificationsToggle', 'Notificações no Navegador')}
              description={t(
                'settings.notificationsToggleDescription',
                'Toasts e push do navegador (com a aba em segundo plano).',
              )}
            />
            <Switch
              checked={notifyPrefs.soundEnabled}
              onChange={(checked) => updateNotifyPrefs({ soundEnabled: checked })}
              label={t('settings.soundToggle', 'Sons de Alerta')}
              description={t(
                'settings.soundToggleDescription',
                'Beep curto ao receber atenção. Independente das notificações visuais.',
              )}
            />
          </Card>
        )}
        {activeTab === 'capacity' && (
          <>
            <CapacityMonitorSettings />
            <CostIntelligenceSettings />
          </>
        )}
        {activeTab === 'hud' && <HudSettings />}
        {activeTab === 'computer' && <ComputerControlSettings projectId={projectId} />}
        {activeTab === 'remote' && <RemoteAccessTab projectId={projectId} />}
      </div>
    </div>
  );
};

interface ThemeCategory {
  id: string;
  title: string;
  description: string;
  presets: ThemePresetKey[];
}

const THEME_CATEGORIES: ThemeCategory[] = [
  {
    id: 'dark-modern',
    title: 'Dark Modern',
    description: 'Paletas escuras premium, alto contraste de código e atmosfera imersiva.',
    presets: [
      'nexus-dark',
      'night',
      'midnight',
      'dracula',
      'graphite-focus',
      'vasco-noir',
      'aurora-ops',
      'obsidian',
    ],
  },
  {
    id: 'light-clean',
    title: 'Light Clean',
    description: 'Interfaces diurnas, excelente legibilidade e visual limpo.',
    presets: ['nexus-light', 'nord-light', 'solarized-light'],
  },
  {
    id: 'classic-code',
    title: 'Clássicos do Código',
    description: 'Esquemas consagrados por gerações de editores de texto.',
    presets: ['monokai', 'nord', 'solarized-dark'],
  },
  {
    id: 'accessibility',
    title: 'Acessibilidade Extrema',
    description: 'Contraste máximo e certificação WCAG 2.2 AA para visualização sem fadiga.',
    presets: ['high-contrast-dark', 'high-contrast-light'],
  },
];

const ThemeAccordionSelector: React.FC<{ theme: ReturnType<typeof useTheme> }> = ({ theme }) => {
  const [openCategories, setOpenCategories] = useState<Record<string, boolean>>(() => {
    // Abre por padrão a categoria onde o preset ativo se encontra
    const active = theme.preset || 'nexus-dark';
    const found = THEME_CATEGORIES.find((cat) => cat.presets.includes(active));
    return { [found ? found.id : 'dark-modern']: true };
  });

  const toggleCategory = (catId: string) => {
    setOpenCategories((curr) => ({ ...curr, [catId]: !curr[catId] }));
  };

  return (
    <div className={`nx-theme-accordion-container ${styles.accordionContainer}`}>
      {/* Grupos de Categorias em Accordion WAI-ARIA */}
      {THEME_CATEGORIES.map((cat) => {
        const isOpen = Boolean(openCategories[cat.id]);
        const headerId = `theme-cat-hdr-${cat.id}`;
        const panelId = `theme-cat-panel-${cat.id}`;

        return (
          <div key={cat.id} className={`nx-theme-cat-group ${styles.themeGroup}`}>
            <button
              type="button"
              id={headerId}
              aria-expanded={isOpen}
              aria-controls={panelId}
              onClick={() => toggleCategory(cat.id)}
              className={`nx-theme-accordion-hdr ${styles.themeHeader}`}
            >
              <div className={styles.themeHeaderLeft}>
                {isOpen ? (
                  <ChevronDown size={15} color="var(--nx-accent)" />
                ) : (
                  <ChevronRight size={15} color="var(--nx-muted)" />
                )}
                <div>
                  <div className={styles.themeTitle}>{cat.title}</div>
                  <div className={styles.themeDesc}>{cat.description}</div>
                </div>
              </div>
              <span className={`nx-badge ${styles.themeBadge}`} data-tone="brand">
                {cat.presets.length} temas
              </span>
            </button>

            {/* Painel do Accordion com Grid de Presets */}
            {isOpen && (
              <div
                id={panelId}
                role="region"
                aria-labelledby={headerId}
                className={styles.themePanel}
              >
                <div className={styles.presetGrid}>
                  {cat.presets.map((presetKey) => {
                    const preset = THEME_PRESETS[presetKey];
                    if (!preset) return null;
                    const isSelected = theme.preset === presetKey && !theme.isCustomized;
                    const palette = getThemePresetPalette(preset);

                    return (
                      <div
                        key={presetKey}
                        role="radio"
                        aria-checked={isSelected}
                        data-selected={isSelected}
                        tabIndex={0}
                        onClick={() => theme.setPreset(presetKey as ThemePresetKey)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault();
                            theme.setPreset(presetKey as ThemePresetKey);
                          }
                        }}
                        className={styles.themePresetItem}
                      >
                        <div className={styles.themePresetName} data-selected={isSelected}>
                          <span>{preset.name}</span>
                          {isSelected && <Check size={14} color="var(--nx-accent)" />}
                        </div>

                        {/* Amostras de Cores (Swatches) */}
                        <div className={styles.themeSwatches} aria-hidden="true">
                          <span
                            title={`Fundo: ${palette.bg}`}
                            className={styles.themeSwatchBg}
                            style={{ background: palette.bg }}
                          />
                          <span
                            title={`Acento: ${palette.accent}`}
                            className={styles.themeSwatchAccent}
                            style={{ background: palette.accent }}
                          />
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};
