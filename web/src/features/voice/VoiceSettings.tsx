import React, { useEffect, useMemo, useState } from 'react';
import { ArrowDown, ArrowUp, KeyRound, Mic, Save, Trash2 } from 'lucide-react';
import { Badge, Button, Card, InlineAlert, Switch } from '../../design-system';
import { nexus } from '../../nexus/api';
import {
  observedVoiceMinutes,
  voiceProviderStatus,
  type VoiceProviderUpdate,
  type VoiceProviderView,
  type VoiceSettingsView,
  type VoiceSettingsUpdate,
} from './voiceModel';
import { invalidateVoiceSettingsCache } from './VoiceCaptureButton';
import styles from './VoiceSettings.module.scss';

export const VoiceSettings: React.FC = () => {
  const [draft, setDraft] = useState<VoiceSettingsView | null>(null);
  const [keys, setKeys] = useState<Record<string, string>>({});
  const [clearKeys, setClearKeys] = useState<Record<string, boolean>>({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [saved, setSaved] = useState(false);

  const load = async () => {
    setLoading(true);
    setError('');
    try {
      setDraft(await nexus.getVoiceSettings());
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    void load();
  }, []);

  const providers = useMemo(
    () => [...(draft?.providers || [])].sort((a, b) => a.priority - b.priority),
    [draft?.providers],
  );

  const patchProvider = (id: string, patch: Partial<VoiceProviderView>) => {
    setDraft((current) =>
      current
        ? {
            ...current,
            providers: current.providers.map((provider) =>
              provider.id === id ? { ...provider, ...patch } : provider,
            ),
          }
        : current,
    );
  };

  const moveProvider = (id: string, delta: -1 | 1) => {
    setDraft((current) => {
      if (!current) return current;
      const ordered = [...current.providers].sort((a, b) => a.priority - b.priority);
      const index = ordered.findIndex((provider) => provider.id === id);
      const target = index + delta;
      if (index < 0 || target < 0 || target >= ordered.length) return current;
      [ordered[index], ordered[target]] = [ordered[target], ordered[index]];
      return {
        ...current,
        providers: ordered.map((provider, i) => ({ ...provider, priority: i + 1 })),
      };
    });
  };

  const save = async () => {
    if (!draft) return;
    setSaving(true);
    setSaved(false);
    setError('');
    try {
      const payload: VoiceSettingsUpdate = {
        enabled: draft.enabled,
        language: draft.language,
        terminal_enabled: draft.terminal_enabled,
        nexus_control_enabled: draft.nexus_control_enabled,
        automatic_fallback: draft.automatic_fallback,
        max_capture_seconds: draft.max_capture_seconds,
        providers: [...draft.providers]
          .sort((a, b) => a.priority - b.priority)
          .map((provider, index) => {
            const {
              has_api_key: _hasApiKey,
              configured: _configured,
              usage: _usage,
              ...config
            } = provider;
            return {
              ...config,
              priority: index + 1,
              api_key: keys[provider.id]?.trim() || undefined,
              clear_api_key: Boolean(clearKeys[provider.id]),
            } satisfies VoiceProviderUpdate;
          }),
      };
      const next = await nexus.putVoiceSettings(payload);
      setDraft(next);
      setKeys({});
      setClearKeys({});
      setSaved(true);
      invalidateVoiceSettingsCache();
      window.dispatchEvent(new CustomEvent('nexus:voice-settings'));
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setSaving(false);
    }
  };

  if (loading && !draft) {
    return (
      <Card className={styles.fullWidth}>
        <div className="nx-settings-card__title">
          <Mic size={17} />
          <div>
            <strong>Voz & Acessibilidade</strong>
            <small>Carregando configuração de reconhecimento de voz…</small>
          </div>
        </div>
      </Card>
    );
  }

  return (
    <Card className={styles.fullWidth}>
      <div className="nx-settings-card__title">
        <Mic size={17} />
        <div>
          <strong>Voz & Acessibilidade</strong>
          <small>
            Ditado em terminais e controle do Nexus usando STT remoto com fallback configurável.
          </small>
        </div>
      </div>

      {error && (
        <InlineAlert tone="danger" title="Configuração de voz indisponível">
          {error}
        </InlineAlert>
      )}
      {saved && (
        <InlineAlert tone="success" title="Configuração de voz salva">
          Os terminais e o controle global passam a usar esta ordem de providers.
        </InlineAlert>
      )}

      {draft && (
        <>
          <div className={styles.toggleGrid}>
            <Switch
              checked={draft.enabled}
              onChange={(enabled) => setDraft({ ...draft, enabled })}
              label="Ativar recursos de voz"
              description="Nenhum áudio é enviado quando esta opção está desligada."
            />
            <Switch
              checked={draft.terminal_enabled}
              onChange={(terminal_enabled) => setDraft({ ...draft, terminal_enabled })}
              label="Ditado nos terminais"
              description="Insere a transcrição no PTY ativo sem pressionar Enter."
            />
            <Switch
              checked={draft.nexus_control_enabled}
              onChange={(nexus_control_enabled) => setDraft({ ...draft, nexus_control_enabled })}
              label="Controle do Nexus por voz"
              description="Somente comandos determinísticos e não destrutivos na primeira versão."
            />
            <Switch
              checked={draft.automatic_fallback}
              onChange={(automatic_fallback) => setDraft({ ...draft, automatic_fallback })}
              label="Fallback automático"
              description="429, autenticação, rede ou indisponibilidade movem para o próximo provider; áudio inválido não é reenviado."
            />
          </div>

          <div className={styles.generalFields}>
            <label>
              <span>Idioma de reconhecimento</span>
              <select
                value={draft.language || ''}
                onChange={(event) => setDraft({ ...draft, language: event.target.value })}
              >
                <option value="">Auto</option>
                <option value="pt">Português</option>
                <option value="en">English</option>
                <option value="es">Español</option>
              </select>
            </label>
            <label>
              <span>Máximo por gravação</span>
              <select
                value={draft.max_capture_seconds}
                onChange={(event) =>
                  setDraft({ ...draft, max_capture_seconds: Number(event.target.value) })
                }
              >
                <option value={30}>30 segundos</option>
                <option value={60}>60 segundos</option>
                <option value={90}>90 segundos</option>
                <option value={120}>120 segundos</option>
              </select>
            </label>
          </div>

          <div className={styles.providerHeader}>
            <div>
              <strong>Speech-to-text providers</strong>
              <p>
                A ordem abaixo é a ordem real de fallback. Uso exibido é somente o observado pelo
                Nexus — não representa a quota global da sua conta.
              </p>
            </div>
          </div>

          <div className={styles.providerList}>
            {providers.map((provider, index) => (
              <section className={styles.providerCard} key={provider.id}>
                <div className={styles.providerTitle}>
                  <div>
                    <strong>{provider.label}</strong>
                    <div className={styles.badges}>
                      <Badge tone={provider.configured ? 'success' : 'warning'}>
                        {voiceProviderStatus(provider)}
                      </Badge>
                      {provider.free_eligible && <Badge tone="info">free-tier elegível</Badge>}
                      <Badge tone="default">prioridade {index + 1}</Badge>
                    </div>
                  </div>
                  <div className={styles.priorityActions}>
                    <button
                      type="button"
                      aria-label={`Subir ${provider.label}`}
                      disabled={index === 0}
                      onClick={() => moveProvider(provider.id, -1)}
                    >
                      <ArrowUp size={14} />
                    </button>
                    <button
                      type="button"
                      aria-label={`Descer ${provider.label}`}
                      disabled={index === providers.length - 1}
                      onClick={() => moveProvider(provider.id, 1)}
                    >
                      <ArrowDown size={14} />
                    </button>
                  </div>
                </div>

                <Switch
                  checked={provider.enabled}
                  onChange={(enabled) => patchProvider(provider.id, { enabled })}
                  label="Provider habilitado"
                />

                <div className={styles.providerFields}>
                  <label>
                    <span>Modelo</span>
                    <input
                      value={provider.model}
                      onChange={(event) =>
                        patchProvider(provider.id, { model: event.target.value })
                      }
                    />
                  </label>
                  {provider.kind === 'cloudflare' && (
                    <label>
                      <span>Cloudflare Account ID</span>
                      <input
                        value={provider.account_id || ''}
                        placeholder={provider.account_id_env || 'CLOUDFLARE_ACCOUNT_ID'}
                        onChange={(event) =>
                          patchProvider(provider.id, { account_id: event.target.value })
                        }
                      />
                    </label>
                  )}
                  <label className={styles.secretField}>
                    <span>
                      <KeyRound size={13} /> API key
                    </span>
                    <input
                      type="password"
                      autoComplete="off"
                      value={keys[provider.id] || ''}
                      placeholder={
                        clearKeys[provider.id]
                          ? 'Chave será removida ao salvar'
                          : provider.has_api_key
                            ? 'Chave já salva — deixe vazio para preservar'
                            : provider.api_key_env
                              ? `Ou use ${provider.api_key_env}`
                              : 'Informe a API key'
                      }
                      onChange={(event) => {
                        const value = event.target.value;
                        setKeys((current) => ({ ...current, [provider.id]: value }));
                        if (value)
                          setClearKeys((current) => ({ ...current, [provider.id]: false }));
                      }}
                    />
                  </label>
                </div>

                {provider.has_api_key && (
                  <Button
                    tone={clearKeys[provider.id] ? 'warning' : 'default'}
                    onClick={() =>
                      setClearKeys((current) => ({
                        ...current,
                        [provider.id]: !current[provider.id],
                      }))
                    }
                  >
                    <Trash2 size={13} />
                    {clearKeys[provider.id] ? 'Manter chave salva' : 'Remover chave ao salvar'}
                  </Button>
                )}

                <div className={styles.usage}>
                  <span>{provider.usage?.requests || 0} requisições observadas</span>
                  <span>{observedVoiceMinutes(provider.usage)} min de áudio com sucesso</span>
                  <span>{provider.usage?.failures || 0} falhas</span>
                  {provider.usage?.last_status && <span>último: {provider.usage.last_status}</span>}
                </div>
              </section>
            ))}
          </div>

          <InlineAlert tone="info" title="Privacidade e custo">
            O áudio é enviado somente após você acionar o microfone. Groq é a primeira opção e
            Cloudflare a segunda por padrão. O Nexus não ativa providers pagos escondidos nem
            inventa quota restante; contas com cobrança habilitada continuam sujeitas às regras do
            próprio fornecedor.
          </InlineAlert>

          <div className={styles.actions}>
            <Button tone="brand" onClick={() => void save()} disabled={saving}>
              <Save size={14} />
              {saving ? 'Salvando…' : 'Salvar voz'}
            </Button>
          </div>
        </>
      )}
    </Card>
  );
};
