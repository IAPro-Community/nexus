import React, { useCallback, useEffect, useState } from 'react';
import { AlertCircle, RefreshCw } from 'lucide-react';
import { Button, Card } from '../../design-system';
import { nexus } from '../../nexus/api';
import {
  deriveFirstRunReadiness,
  OFFICIAL_PROVIDER_INSTALL_GUIDES,
} from '../projects/firstRunReadiness';
import type { ProviderInfo } from '../../types';
import styles from './ProviderReadinessBanner.module.scss';

// Discovery only. The HUD is neither mounted nor consulted for provider readiness.
export const ProviderReadinessBanner: React.FC<{ onOpenAccounts: () => void }> = ({
  onOpenAccounts,
}) => {
  const [providers, setProviders] = useState<ProviderInfo[]>([]);
  const [state, setState] = useState<'checking' | 'loaded' | 'unavailable'>('checking');

  const refresh = useCallback(async () => {
    setState('checking');
    try {
      const detected = await nexus.listProviderInstallations();
      setProviders(detected || []);
      setState('loaded');
    } catch {
      setState('unavailable');
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const readiness = deriveFirstRunReadiness(providers, state);
  if (readiness.status === 'checking' || readiness.status === 'ready') return null;

  return (
    <Card className={styles.banner} data-testid="provider-readiness-banner">
      <div className={styles.copy} role="status" aria-live="polite">
        <strong>
          <AlertCircle size={15} aria-hidden="true" /> {readiness.title}
        </strong>
        <p>{readiness.description}</p>
      </div>
      <div className={styles.actions}>
        {readiness.status === 'needs-provider' &&
          OFFICIAL_PROVIDER_INSTALL_GUIDES.map((guide) => (
            <a key={guide.label} href={guide.href} target="_blank" rel="noopener noreferrer">
              {guide.label}
            </a>
          ))}
        {readiness.status !== 'unavailable' && (
          <Button size="sm" tone="brand" onClick={onOpenAccounts}>
            {readiness.status === 'needs-login' ? 'Concluir login' : 'Configurar conta'}
          </Button>
        )}
        <Button size="sm" tone="ghost" onClick={() => void refresh()}>
          <RefreshCw size={13} /> Verificar novamente
        </Button>
      </div>
    </Card>
  );
};
