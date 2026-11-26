import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Sparkles, Wrench } from 'lucide-react';
import type { AutoConfigureIntelligencePayload, NexV2ChatSetupAssistance } from './nexV2Types';
import { autoConfigureNexIntelligence } from './nexV2Client';
import styles from './NexChatSetupCard.module.scss';

export interface NexChatSetupCardProps {
  assistance: NexV2ChatSetupAssistance;
  onConfigured: () => Promise<void> | void;
  onOpenResources?: () => void;
  onOpenSettings?: () => void;
}

/**
 * Inline zero-friction recovery card rendered directly inside a Nex reply when
 * open-ended Intelligence is not yet configured or ready.
 */
export const NexChatSetupCard: React.FC<NexChatSetupCardProps> = ({
  assistance,
  onConfigured,
  onOpenResources,
  onOpenSettings,
}) => {
  const { t } = useTranslation();
  const [busy, setBusy] = useState(false);
  const [showForm, setShowForm] = useState(!assistance.can_auto_configure);
  const [baseUrl, setBaseUrl] = useState('http://127.0.0.1:11434/v1');
  const [modelName, setModelName] = useState('');
  const [apiKeyEnv, setApiKeyEnv] = useState('');
  const [errorMsg, setErrorMsg] = useState('');

  const runConfigure = async (payload: AutoConfigureIntelligencePayload) => {
    if (busy) return;
    setBusy(true);
    setErrorMsg('');
    const result = await autoConfigureNexIntelligence(payload);
    setBusy(false);
    if (!result.ok) {
      setErrorMsg(result.error || t('nexV2.setupFailed'));
      return;
    }
    await onConfigured();
  };

  const candidateLabel =
    assistance.recommended_provider && assistance.recommended_profile
      ? `${assistance.recommended_provider}:${assistance.recommended_profile}`
      : assistance.recommended_provider || 'CLI';

  return (
    <section
      className={styles.card}
      aria-label={t('nexV2.setupCardTitle')}
      data-nex-chat-setup={assistance.reason || 'unconfigured'}
    >
      <div className={styles.header}>
        <strong className={styles.title}>{t('nexV2.setupCardTitle')}</strong>
      </div>
      <p className={styles.hint}>
        {assistance.can_auto_configure
          ? t('nexV2.setupHintAuto', { target: candidateLabel })
          : assistance.recommended_action === 'authenticate_cli_profile'
            ? t('nexV2.setupHintAuthCli', { target: candidateLabel })
            : t('nexV2.setupHintManual')}
      </p>

      <div className={styles.actionsRow}>
        {assistance.can_auto_configure && (
          <button
            type="button"
            className={styles.primaryBtn}
            disabled={busy}
            data-nex-setup-auto
            onClick={() =>
              void runConfigure({
                mode: 'AUTO',
                provider: assistance.recommended_provider,
                profile: assistance.recommended_profile,
              })
            }
          >
            <Sparkles size={12} aria-hidden="true" />
            <span>
              {busy
                ? t('nexV2.setupApplying')
                : t('nexV2.setupAutoBtn', { target: candidateLabel })}
            </span>
          </button>
        )}

        <button
          type="button"
          className={styles.secondaryBtn}
          aria-expanded={showForm}
          data-nex-setup-toggle-form
          onClick={() => setShowForm((prev) => !prev)}
        >
          <Wrench size={12} aria-hidden="true" />
          <span>{t('nexV2.setupConfigureEndpointBtn')}</span>
        </button>

        {onOpenResources && (
          <button
            type="button"
            className={styles.secondaryBtn}
            data-nex-setup-open-resources
            onClick={onOpenResources}
          >
            <span>{t('nexV2.setupOpenResourcesBtn')}</span>
          </button>
        )}

        {onOpenSettings && (
          <button
            type="button"
            className={styles.secondaryBtn}
            data-nex-setup-open-settings
            onClick={onOpenSettings}
          >
            <span>{t('nexV2.setupOpenSettingsBtn')}</span>
          </button>
        )}
      </div>

      {showForm && (
        <form
          className={styles.form}
          data-nex-setup-form
          onSubmit={(event) => {
            event.preventDefault();
            if (!baseUrl.trim() || !modelName.trim()) {
              setErrorMsg(t('nexV2.setupValidationMissing'));
              return;
            }
            void runConfigure({
              mode: 'OPENAI_COMPATIBLE',
              base_url: baseUrl.trim(),
              model: modelName.trim(),
              api_key_env: apiKeyEnv.trim() || undefined,
            });
          }}
        >
          <div className={styles.fieldRow}>
            <label className={styles.fieldLabel} htmlFor="nex-setup-base-url">
              {t('nexV2.setupBaseUrlLabel')}
            </label>
            <input
              id="nex-setup-base-url"
              type="url"
              className={styles.input}
              value={baseUrl}
              onChange={(e) => setBaseUrl(e.target.value)}
              placeholder="http://127.0.0.1:11434/v1"
              data-nex-setup-url
            />
          </div>
          <div className={styles.fieldRow}>
            <label className={styles.fieldLabel} htmlFor="nex-setup-model">
              {t('nexV2.setupModelLabel')}
            </label>
            <input
              id="nex-setup-model"
              type="text"
              className={styles.input}
              value={modelName}
              onChange={(e) => setModelName(e.target.value)}
              placeholder={t('nexV2.setupModelPlaceholder')}
              data-nex-setup-model
            />
          </div>
          <div className={styles.fieldRow}>
            <label className={styles.fieldLabel} htmlFor="nex-setup-api-env">
              {t('nexV2.setupApiKeyEnvLabel')}
            </label>
            <input
              id="nex-setup-api-env"
              type="text"
              className={styles.input}
              value={apiKeyEnv}
              onChange={(e) => setApiKeyEnv(e.target.value)}
              placeholder="OPENAI_API_KEY"
              data-nex-setup-env
            />
          </div>
          <div className={styles.actionsRow}>
            <button
              type="submit"
              className={styles.primaryBtn}
              disabled={busy || !baseUrl.trim() || !modelName.trim()}
              data-nex-setup-submit
            >
              <span>{busy ? t('nexV2.setupApplying') : t('nexV2.setupSaveAndRetryBtn')}</span>
            </button>
          </div>
        </form>
      )}

      {errorMsg && (
        <p className={styles.error} role="alert" data-nex-setup-error>
          {errorMsg}
        </p>
      )}
    </section>
  );
};
