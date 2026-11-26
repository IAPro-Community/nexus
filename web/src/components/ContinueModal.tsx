import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { RuntimeSession, ProviderInfo, ProfileInfo } from '../types';
import { asArray } from '../lib/safeArray';
import { api } from '../api';
import { AlertTriangle } from 'lucide-react';
import { Button, Dialog, Select } from '../design-system';

interface ContinueModalProps {
  runtime: RuntimeSession;
  providers: ProviderInfo[];
  profiles: ProfileInfo[];
  onClose: () => void;
  onSuccess: (newSession: RuntimeSession) => void;
  open?: boolean;
}

export const ContinueModal: React.FC<ContinueModalProps> = ({
  runtime,
  providers,
  profiles,
  onClose,
  onSuccess,
  open = true,
}) => {
  const { t } = useTranslation();
  const safeProviders = asArray<ProviderInfo>(providers);
  const safeProfiles = asArray<ProfileInfo>(profiles);

  const runtimeProvider = runtime.provider_id || runtime.provider;
  const otherProviders = safeProviders.filter((p) => p.id !== runtimeProvider && p.installed);
  const [selectedProvider, setSelectedProvider] = useState<string>(
    otherProviders.length > 0 ? otherProviders[0].id : '',
  );

  const availableProfiles = safeProfiles.filter((p) => p.provider === selectedProvider);
  const [selectedProfile, setSelectedProfile] = useState<string>(
    availableProfiles.length > 0 ? availableProfiles[0].name : 'default',
  );

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleProviderChange = (prov: string) => {
    setSelectedProvider(prov);
    const profs = safeProfiles.filter((p) => p.provider === prov);
    setSelectedProfile(profs.length > 0 ? profs[0].name : 'default');
  };

  const handleContinue = async () => {
    if (!selectedProvider) return;
    setLoading(true);
    setError('');
    try {
      const res = await api.contextContinue(runtime.runtime_id, selectedProvider, selectedProfile);
      onSuccess(res);
      onClose();
    } catch (err: any) {
      setError(err.message || 'Context continue failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('legacy.continueModalTitle', 'Continue with Another AI')}
    >
      <div className="space-y-4 font-mono">
        <div
          className="p-3 rounded flex items-start space-x-2.5 text-xs"
          style={{
            background: 'var(--nx-surface-2)',
            border: '1px solid var(--nx-accent)',
            color: 'var(--nx-accent-text)',
          }}
        >
          <AlertTriangle
            className="w-4 h-4 flex-shrink-0 mt-0.5"
            style={{ color: 'var(--nx-accent-text)' }}
          />
          <div>
            <strong>{t('legacy.continueWarning', 'A NEW SESSION WILL BE CREATED.')}</strong>
            <p className="mt-1 text-[11px]" style={{ color: 'var(--nx-text-soft)' }}>
              {t(
                'legacy.continueDesc',
                'Captures current git status, diff stats, and modified files with sensitive secrets redacted, and boots a clean agent thread in the target CLI.',
              )}
            </p>
          </div>
        </div>

        {otherProviders.length === 0 ? (
          <div
            className="p-4 rounded text-xs"
            style={{
              background: 'var(--nx-surface-2)',
              border: '1px solid var(--nx-warning)',
              color: 'var(--nx-warning)',
            }}
          >
            {t(
              'legacy.noOtherClis',
              'No other installed coding CLIs detected. Install Claude, Codex, AGY, or OpenCode to continue across providers.',
            )}
          </div>
        ) : (
          <div className="space-y-3 text-xs">
            <div>
              <label
                className="text-xs"
                style={{ color: 'var(--nx-text-soft)', display: 'block', marginBottom: 4 }}
              >
                {t('legacy.destProvider', 'Destination Provider:')}
              </label>
              <Select
                value={selectedProvider}
                onChange={(val) => handleProviderChange(val)}
                options={otherProviders.map((p) => ({
                  value: p.id,
                  label: `${p.id.toUpperCase()} ${p.version ? `(${p.version})` : ''}`.trim(),
                }))}
              />
            </div>

            <div>
              <label
                className="text-xs"
                style={{ color: 'var(--nx-text-soft)', display: 'block', marginBottom: 4 }}
              >
                {t('legacy.profileAccount', 'Profile / Account:')}
              </label>
              <Select
                value={selectedProfile}
                onChange={(val) => setSelectedProfile(val)}
                options={
                  availableProfiles.length > 0
                    ? availableProfiles.map((p) => ({
                        value: p.name,
                        label: `${p.name} ${p.account_email ? `(${p.account_email})` : ''}`.trim(),
                      }))
                    : [{ value: 'default', label: 'default' }]
                }
              />
            </div>
          </div>
        )}

        {error && (
          <div
            className="p-3 rounded text-xs"
            style={{
              background: 'var(--nx-surface-2)',
              border: '1px solid var(--nx-danger)',
              color: 'var(--nx-danger)',
            }}
          >
            {error}
          </div>
        )}

        <div className="nx-dialog-actions">
          <Button onClick={onClose}>{t('directSession.cancel', 'Cancel')}</Button>
          <Button
            tone="brand"
            disabled={loading || otherProviders.length === 0}
            onClick={handleContinue}
          >
            {loading
              ? t('legacy.startingContinuedSession', 'Starting Continued Session...')
              : t('legacy.startContinuedSession', 'Start Continued Session')}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
