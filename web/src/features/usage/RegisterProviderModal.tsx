import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { CheckCircle2, Cpu, Plus, Radio } from 'lucide-react';
import { toast } from 'sonner';
import {
  Badge,
  Button,
  ChoiceGroup,
  ChoiceOption,
  Dialog,
  EmptyState,
  Input,
  Spinner,
} from '../../design-system';
import { nexus } from '../../nexus/api';
import { OFFICIAL_PROVIDER_INSTALL_GUIDES } from '../projects/firstRunReadiness';
import type { ProviderInfo } from '../../types';
import styles from './RegisterProviderModal.module.scss';
import {
  preferredRegistrationProvider,
  requiresExplicitAdditionalProfileName,
} from './registerProviderModel';

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
      // An unavailable CLI must never be the default registration target.
      setSelectedProvider(preferredRegistrationProvider(list || [], initialProvider));
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

  const selectedRegistration = providers.find((p) => p.id === selectedProvider);
  const requiresName = requiresExplicitAdditionalProfileName(selectedRegistration, profileName);

  const handleSubmit = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!selectedProvider || !providers.some((p) => p.id === selectedProvider && p.installed)) {
      setError(t('usage.selectProviderRequired', 'Selecione um provedor instalado'));
      return;
    }
    if (requiresName) {
      setError(
        'Este provedor já possui um perfil. Conclua o login em Uso/Contas ou informe um nome diferente para adicionar outra conta.',
      );
      return;
    }
    const cleanProfile = profileName.trim() || selectedProvider;
    setSubmitting(true);
    setError('');
    try {
      const result = await nexus.registerProvider(selectedProvider, cleanProfile);
      const linkedFromHost = result.auth_source?.origin === 'host';
      if (linkedFromHost) {
        toast.success(t('usage.registerSuccess', 'Perfil registrado com sucesso!'), {
          description: t('usage.registerLinkedFromHost', {
            provider: selectedProvider,
            plan: result.auth_source?.plan || '',
            defaultValue:
              'Vinculamos o login de {{provider}} já existente neste computador ({{plan}}). Nada foi digitado e a conta atual continua a mesma.',
          }),
          duration: 9000,
        });
      } else {
        toast.success(t('usage.registerSuccess', 'Perfil registrado com sucesso!'));
      }
      onClose();
      try {
        await onRegistered(selectedProvider, cleanProfile);
      } catch (followUpError) {
        toast.error(
          'Perfil registrado, mas não foi possível abrir o próximo passo: ' +
            (followUpError instanceof Error ? followUpError.message : String(followUpError)),
        );
      }
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
    if (prov.registration_state === 'REGISTRATION_UNKNOWN') {
      return <Badge tone="warning">{t('usage.verificationUnknownBadge')}</Badge>;
    }
    return <Badge tone="brand">{t('usage.unregisteredBadge')}</Badge>;
  };

  return (
    <Dialog open={isOpen} onClose={onClose} title={t('usage.addAccount', 'Adicionar conta')}>
      <form onSubmit={(e) => void handleSubmit(e)} className={styles.root}>
        <p className={styles.intro}>{t('usage.addAccountDesc')}</p>

        {loading ? <Spinner label={t('common.loading', 'Carregando provedores…')} /> : null}

        {!loading && providers.every((provider) => !provider.installed) && (
          <div className={styles.intro}>
            Nenhuma CLI compatível está disponível para cadastro. Instale uma ferramenta oficial e
            volte para verificar o ambiente:{' '}
            {OFFICIAL_PROVIDER_INSTALL_GUIDES.map((guide) => (
              <a
                key={guide.label}
                href={guide.href}
                target="_blank"
                rel="noopener noreferrer"
                style={{ marginRight: 10 }}
              >
                {guide.label}
              </a>
            ))}
          </div>
        )}

        {error ? <p className={styles.error}>{error}</p> : null}

        {!loading && providers.length === 0 ? (
          <EmptyState
            icon={<Cpu size={20} />}
            title={t('legacy.providers', 'Nenhum provedor detectado')}
          />
        ) : null}

        {!loading && providers.length > 0 ? (
          <ChoiceGroup
            className={styles.providerList}
            ariaLabel={t('usage.availableProviders', 'Provedores disponíveis')}
          >
            {providers.map((prov) => {
              const isSelected = selectedProvider === prov.id;
              return (
                <ChoiceOption
                  key={prov.id}
                  className={styles.providerItem}
                  selected={isSelected}
                  onSelect={() => {
                    if (prov.installed) setSelectedProvider(prov.id);
                  }}
                >
                  <span className={styles.providerInfo}>
                    <span className={styles.providerHeader}>
                      <strong>{prov.id}</strong>
                      {getRegistrationBadge(prov)}
                    </span>
                    <span className={styles.providerMeta}>
                      <span>{prov.version || 'Versão indisponível'}</span>
                      {prov.binary_path ? <code>{prov.binary_path}</code> : null}
                    </span>
                  </span>
                  {isSelected ? (
                    <CheckCircle2 size={16} color="var(--nx-accent)" />
                  ) : (
                    <Radio size={16} color="var(--nx-muted)" />
                  )}
                </ChoiceOption>
              );
            })}
          </ChoiceGroup>
        ) : null}

        {requiresName ? (
          <p className={styles.intro}>
            Já existe um perfil deste provedor. Para adicionar outra conta, informe um nome de
            perfil diferente; para continuar com a conta atual, conclua o login em Uso/Contas.
          </p>
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
          <Button
            type="submit"
            tone="brand"
            disabled={
              submitting ||
              requiresName ||
              !providers.some((p) => p.id === selectedProvider && p.installed)
            }
          >
            <Plus size={14} />
            {submitting ? t('usage.registering') : t('usage.registerProvider', 'Registrar')}
          </Button>
        </div>
      </form>
    </Dialog>
  );
};
