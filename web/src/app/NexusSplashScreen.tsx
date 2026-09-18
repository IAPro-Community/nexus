import React from 'react';
import { useTranslation } from 'react-i18next';

export const NexusSplashScreen: React.FC<{
  message?: string;
  submessage?: string;
  stage?: 'starting' | 'loading' | 'switching';
}> = ({ message, submessage, stage = 'loading' }) => {
  const { t } = useTranslation();
  const title =
    message ||
    (stage === 'starting'
      ? t('app.starting', 'Iniciando IAPro Nexus…')
      : t('app.loading', 'Carregando Projetos e Agentes…'));
  const subtitle =
    submessage || (stage === 'starting' ? t('app.startingSubtitle') : t('app.loadingSubtitle'));

  return (
    <div className="nx-splash-screen" role="status" aria-live="polite" aria-label={title}>
      <div className="nx-splash-card">
        <div className="nx-splash-logo-wrap">
          <div className="nx-splash-logo-glow" />
          <img src="./nexus-icon.png" alt="IAPro Nexus Logo" className="nx-splash-logo" />
        </div>

        <div className="nx-splash-brand">
          <span className="nx-splash-eyebrow">{t('app.splashEyebrow')}</span>
          <h1 className="nx-splash-title">IAPro Nexus</h1>
        </div>

        <div className="nx-splash-progress-track">
          <div className="nx-splash-progress-shimmer" />
        </div>

        <div className="nx-splash-status">
          <span className="nx-splash-dot" />
          <span className="nx-splash-message">{title}</span>
        </div>
        <span className="nx-splash-submessage">{subtitle}</span>

        <div className="nx-splash-footer">
          <span>v0.5.0-beta</span>
          <span>•</span>
          <span>{t('app.protectedLocalEnv')}</span>
        </div>
      </div>
    </div>
  );
};
