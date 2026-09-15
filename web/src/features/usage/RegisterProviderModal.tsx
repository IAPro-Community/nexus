import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { CheckCircle2, Cpu, Plus, Radio } from 'lucide-react';
import { toast } from 'sonner';
import { Badge, Button, Dialog, EmptyState, Input, Spinner } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { ProviderInfo } from '../../types';
import styles from './RegisterProviderModal.module.scss';

export interface RegisterProviderModalProps {
  isOpen: boolean;
  onClose: () => void;
  onRegistered: (provider: string, profile: string) => Promise<void>;
  initialProvider?: string;
}

export const RegisterProviderModal: React.FC<RegisterProviderModalProps> = ({
  isOpen,
  onClose,
  onRegistered,
  initialProvider,
}) => {
  const { t } = useTranslation();
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [loading, setLoading] = useState(false);
  const [selectedProvider, setSelectedProvider] = useState<string>(initialProvider || '');
  const [profileName, setProfileName] = useState<string>('');
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string>('');

  const loadProviders = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const list = await nexus.listProviderInstallations();
      setProviders(list || []);
      if (list && list.length > 0) {
        if (initialProvider && list.some((p) => p.id === initialProvider)) {
          setSelectedProvider(initialProvider);
        } else {
          // Default to first installed or unregistered
          const preferred =
            list.find((p) => p.installed && p.registration_state !== 'REGISTERED_AUTHENTICATED') ||
            list[0];
          setSelectedProvider(preferred.id);
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setLoading(false);
    }
  }, [initialProvider]);

  useEffect(() => {
    if (isOpen) {
      void loadProviders();
      setProfileName('');
      if (initialProvider) {
        setSelectedProvider(initialProvider);
      }
    }
  }, [isOpen, initialProvider, loadProviders]);

  if (!isOpen) return null;

  const handleSubmit = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!selectedProvider) {
      setError(t('usage.selectProviderRequired', 'Selecione um provedor'));
      return;
    }
    const cleanProfile = profileName.trim() || selectedProvider;
    setSubmitting(true);
    setError('');
    try {
      await nexus.registerProvider(selectedProvider, cleanProfile);
      toast.success(t('usage.registerSuccess', 'Perfil registrado com sucesso!'));
      await onRegistered(selectedProvider, cleanProfile);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  };

  const getRegistrationBadge = (prov: ProviderInfo) => {
    if (!prov.installed) {
      return <Badge tone="default">{t('legacy.no', 'Não instalado')}</Badge>;
    }
    if (prov.registration_state === 'REGISTERED_AUTHENTICATED') {
      return <Badge tone="success">{t('usage.registeredBadge')}</Badge>;
    }
    if (prov.registration_state === 'PENDING_AUTH') {
      return <Badge tone="warning">{t('usage.pendingAuthBadge')}</Badge>;
    }
    return <Badge tone="brand">{t('usage.unregisteredBadge')}</Badge>;
  };

  return (
    <Dialog open={isOpen} onClose={onClose} title={t('usage.addAccount', 'Adicionar conta')}>
      <form onSubmit={(e) => void handleSubmit(e)} className={styles.root}>
        <p className={styles.intro}>{t('usage.addAccountDesc')}</p>

        {loading ? <Spinner label={t('common.loading', 'Carregando provedores…')} /> : null}

        {error ? <p className={styles.error}>{error}</p> : null}

        {!loading && providers.length === 0 ? (
          <EmptyState
            icon={<Cpu size={20} />}
            title={t('legacy.providers', 'Nenhum provedor detectado')}
          />
        ) : null}

        {!loading && providers.length > 0 ? (
          <div
            className={styles.providerList}
            role="radiogroup"
            aria-label="Provedores disponíveis"
          >
            {providers.map((prov) => {
              const isSelected = selectedProvider === prov.id;
              return (
                <div
                  key={prov.id}
                  className={styles.providerItem}
                  data-selected={isSelected}
                  onClick={() => setSelectedProvider(prov.id)}
                  role="radio"
                  aria-checked={isSelected}
                  tabIndex={0}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' || e.key === ' ') {
                      setSelectedProvider(prov.id);
                    }
                  }}
                >
                  <div className={styles.providerInfo}>
                    <div className={styles.providerHeader}>
                      <strong>{prov.id}</strong>
                      {getRegistrationBadge(prov)}
                    </div>
                    <div className={styles.providerMeta}>
                      <span>{prov.version || 'v1.0'}</span>
                      {prov.binary_path ? <code>{prov.binary_path}</code> : null}
                    </div>
                  </div>
                  {isSelected ? (
                    <CheckCircle2 size={16} color="var(--nx-accent)" />
                  ) : (
                    <Radio size={16} color="var(--nx-muted)" />
                  )}
                </div>
              );
            })}
          </div>
        ) : null}

        <div className={styles.formGroup}>
          <label className={styles.formLabel} htmlFor="nx-profile-name-input">
            {t('usage.profileNameLabel', 'Nome do perfil')}
          </label>
          <Input
            id="nx-profile-name-input"
            value={profileName}
            onChange={setProfileName}
            placeholder={
              selectedProvider
                ? `ex: ${selectedProvider}-work, pessoal`
                : t('usage.profileNamePlaceholder')
            }
          />
        </div>

        <div className={styles.footerActions}>
          <Button onClick={onClose}>{t('common.cancel', 'Cancelar')}</Button>
          <Button type="submit" tone="brand" disabled={submitting || !selectedProvider}>
            <Plus size={14} />
            {submitting ? t('usage.registering') : t('usage.registerProvider', 'Registrar')}
          </Button>
        </div>
      </form>
    </Dialog>
  );
};
