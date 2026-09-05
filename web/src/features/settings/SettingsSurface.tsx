import React, { useEffect, useMemo, useState } from 'react';
import {
  Accessibility,
  ArrowUpCircle,
  Bell,
  Check,
  ChevronDown,
  ChevronRight,
  MonitorCog,
  Palette,
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
import { CapacityMonitorSettings } from './CapacityMonitorSettings';

type SettingsTab = 'appearance' | 'accessibility' | 'updates' | 'intelligence' | 'notifications';

export const SettingsSurface: React.FC<{ onTour: () => void }> = ({ onTour }) => {
  const theme = useTheme();
  const { t } = useTranslation();
  const workspace = useWorkspace();
  const [activeTab, setActiveTab] = useState<SettingsTab>('appearance');

  const [checkingUpdates, setCheckingUpdates] = useState(false);
  const [updateInfo, setUpdateInfo] = useState<{
    nexus_version: string;
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
          aria-selected={activeTab === 'appearance'}
          className={`nx-settings-tab ${activeTab === 'appearance' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('appearance')}
        >
          <Palette size={15} />
          <span>Aparência & Estilo</span>
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'accessibility'}
          className={`nx-settings-tab ${activeTab === 'accessibility' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('accessibility')}
        >
          <Accessibility size={15} />
          <span>Acessibilidade</span>
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'updates'}
          className={`nx-settings-tab ${activeTab === 'updates' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('updates')}
        >
          <ArrowUpCircle size={15} />
          <span>Atualizações & Sistema</span>
          {updateInfo?.update_available && (
            <span className="nx-badge" data-tone="warning" style={{ fontSize: 10 }}>
              Update
            </span>
          )}
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'intelligence'}
          className={`nx-settings-tab ${activeTab === 'intelligence' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('intelligence')}
        >
          <Sparkles size={15} />
          <span>Nexus Intelligence</span>
        </button>
        <button
          type="button"
          role="tab"
          aria-selected={activeTab === 'notifications'}
          className={`nx-settings-tab ${activeTab === 'notifications' ? 'nx-settings-tab--active' : ''}`}
          onClick={() => setActiveTab('notifications')}
        >
          <Bell size={15} />
          <span>Notificações</span>
        </button>
      </div>

      <div className="nx-settings-grid">
        {/* Tab 1: Appearance */}
        {activeTab === 'appearance' && (
          <>
            <Card className="nx-settings-card" style={{ gridColumn: '1 / -1' }}>
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
              <p className="nx-muted-copy" style={{ fontSize: '12px' }}>
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
          </>
        )}

        {/* Tab 3: Updates & System */}
        {activeTab === 'updates' && (
          <Card className="nx-settings-card" style={{ gridColumn: '1 / -1' }}>
            <div className="nx-settings-card__title">
              <ArrowUpCircle size={17} />
              <div>
                <strong>{t('settings.updates')}</strong>
                <small>{t('settings.updatesDescription')}</small>
              </div>
            </div>
            {updateInfo && (
              <div
                style={{
                  display: 'flex',
                  flexDirection: 'column',
                  gap: 8,
                  fontSize: '13px',
                  maxWidth: 480,
                }}
              >
                <div
                  style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                >
                  <span>IAPro Nexus Core:</span>
                  <Badge tone="success">v{updateInfo.nexus_version}</Badge>
                </div>
                <div
                  style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                >
                  <span>Orquestrador Maestro:</span>
                  <Badge tone={updateInfo.maestro_available ? 'brand' : 'warning'}>
                    v{updateInfo.maestro_version}
                  </Badge>
                </div>
                {updateInfo.update_available ? (
                  <InlineAlert tone="warning" title="Atualização disponível">
                    Há uma nova versão do Orquestrador Maestro disponível (
                    {updateInfo.maestro_latest_version
                      ? `v${updateInfo.maestro_latest_version}`
                      : 'nova build'}
                    ).
                  </InlineAlert>
                ) : (
                  <InlineAlert tone="success" title="Sistema em dia">
                    Você está executando as versões mais recentes estáveis do Nexus e do Maestro.
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
                  ? t('maestroControl.updateMaestroDone', { version: updateResult.maestro_version })
                  : t('maestroControl.updateMaestroSame', {
                      version: updateResult.maestro_version,
                    })}{' '}
                {t('maestroControl.nexusBinaryNote', { version: updateResult.nexus_version })}
              </InlineAlert>
            )}
            <div style={{ display: 'flex', gap: 10, marginTop: 12 }}>
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
        )}

        {/* Tab 4: Intelligence */}
        {activeTab === 'intelligence' && (
          <Card className="nx-settings-card" style={{ gridColumn: '1 / -1' }}>
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
            <div style={{ marginTop: 8 }}>
              <Button
                tone="brand"
                onClick={() => void saveIntelligence()}
                disabled={intelligenceSaving}
              >
                {intelligenceSaving ? 'Saving…' : 'Save Intelligence'}
              </Button>
            </div>
          </Card>
        )}

        {/* Tab 5: Notifications */}
        {activeTab === 'notifications' && (
          <>
            <Card className="nx-settings-card" style={{ gridColumn: '1 / -1' }}>
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
            <CapacityMonitorSettings />
          </>
        )}
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
    title: 'Temas Escuros & Noturnos',
    description: 'Paletas modernas com contraste equilibrado para ambientes com pouca luz',
    presets: ['nexus-dark', 'midnight', 'nord', 'dracula', 'monokai'],
  },
  {
    id: 'light-clean',
    title: 'Temas Claros & Clean',
    description: 'Estilo clean e luminoso com legibilidade aprimorada',
    presets: ['nexus-light', 'solarized-light'],
  },
  {
    id: 'high-contrast',
    title: 'Acessibilidade & Alto Contraste (WCAG AAA)',
    description: 'Relações de contraste máximas (>= 7:1) para máxima distinção visual',
    presets: ['high-contrast-dark', 'high-contrast-light', 'solarized-dark'],
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
    <div className="nx-theme-accordion-container" style={{ display: 'grid', gap: 10 }}>
      {/* Grupos de Categorias em Accordion WAI-ARIA */}
      {THEME_CATEGORIES.map((cat) => {
        const isOpen = Boolean(openCategories[cat.id]);
        const headerId = `theme-cat-hdr-${cat.id}`;
        const panelId = `theme-cat-panel-${cat.id}`;

        return (
          <div
            key={cat.id}
            className="nx-theme-cat-group"
            style={{
              border: '1px solid var(--nx-border)',
              borderRadius: 8,
              background: 'var(--nx-bg-elevated)',
              overflow: 'hidden',
            }}
          >
            <button
              type="button"
              id={headerId}
              aria-expanded={isOpen}
              aria-controls={panelId}
              onClick={() => toggleCategory(cat.id)}
              className="nx-theme-accordion-hdr"
              style={{
                width: '100%',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: '10px 14px',
                border: 0,
                background: 'transparent',
                color: 'var(--nx-text)',
                cursor: 'pointer',
                textAlign: 'left',
                fontSize: 13,
                fontWeight: 650,
              }}
            >
              <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                {isOpen ? (
                  <ChevronDown size={15} color="var(--nx-accent)" />
                ) : (
                  <ChevronRight size={15} color="var(--nx-muted)" />
                )}
                <div>
                  <div style={{ color: 'var(--nx-text)' }}>{cat.title}</div>
                  <div style={{ fontSize: 11, color: 'var(--nx-muted)', fontWeight: 400 }}>
                    {cat.description}
                  </div>
                </div>
              </div>
              <span className="nx-badge" data-tone="brand" style={{ fontSize: 10 }}>
                {cat.presets.length} temas
              </span>
            </button>

            {/* Painel do Accordion com Grid de Presets */}
            {isOpen && (
              <div
                id={panelId}
                role="region"
                aria-labelledby={headerId}
                style={{
                  padding: '12px 14px',
                  borderTop: '1px solid var(--nx-border)',
                  background: 'var(--nx-surface-2)',
                }}
              >
                <div
                  style={{
                    display: 'grid',
                    gridTemplateColumns: 'repeat(auto-fill, minmax(220px, 1fr))',
                    gap: 8,
                  }}
                >
                  {cat.presets.map((presetKey) => {
                    const preset = THEME_PRESETS[presetKey];
                    if (!preset) return null;
                    const isSelected = theme.preset === presetKey && !theme.isCustomized;
                    const palette = getThemePresetPalette(preset);

                    return (
                      <div
                        key={presetKey}
                        role="button"
                        tabIndex={0}
                        onClick={() => theme.setPreset(presetKey as ThemePresetKey)}
                        onKeyDown={(e) => {
                          if (e.key === 'Enter' || e.key === ' ') {
                            e.preventDefault();
                            theme.setPreset(presetKey as ThemePresetKey);
                          }
                        }}
                        style={{
                          display: 'flex',
                          alignItems: 'center',
                          justifyContent: 'space-between',
                          gap: 8,
                          padding: '8px 10px',
                          borderRadius: 6,
                          border: isSelected
                            ? '1.5px solid var(--nx-accent)'
                            : '1px solid var(--nx-border)',
                          background: isSelected
                            ? 'var(--nx-accent-soft)'
                            : 'var(--nx-bg-elevated)',
                          cursor: 'pointer',
                          transition: 'all 0.15s ease',
                        }}
                      >
                        <div
                          style={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 6,
                            fontSize: 12,
                            fontWeight: isSelected ? 650 : 500,
                          }}
                        >
                          <span>{preset.name}</span>
                          {isSelected && <Check size={14} color="var(--nx-accent)" />}
                        </div>

                        {/* Amostras de Cores (Swatches) */}
                        <div
                          style={{ display: 'flex', alignItems: 'center', gap: 4 }}
                          aria-hidden="true"
                        >
                          <span
                            title={`Fundo: ${palette.bg}`}
                            style={{
                              width: 14,
                              height: 14,
                              borderRadius: 3,
                              background: palette.bg,
                              border: '1px solid var(--nx-border)',
                            }}
                          />
                          <span
                            title={`Acento: ${palette.accent}`}
                            style={{
                              width: 14,
                              height: 14,
                              borderRadius: 3,
                              background: palette.accent,
                            }}
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
