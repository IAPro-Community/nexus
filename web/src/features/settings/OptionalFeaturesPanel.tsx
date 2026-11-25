// Read-only system diagnostics panel inside Settings > Updates (alongside
// SystemDiagnosticsCard). Intentionally internal diagnostics rather than a
// user-facing product feature announcement in featureAnnouncements.ts.
import React from 'react';
import { CheckCircle2, Circle, Sliders } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Card, InlineAlert } from '../../design-system';
import {
  OPTIONAL_FEATURE_KEYS,
  useOptionalFeatures,
  type OptionalFeatureKey,
} from '../../app/composerFlowFlag';
import styles from './OptionalFeaturesPanel.module.scss';

const FEATURE_LABEL_KEYS: Record<OptionalFeatureKey, { titleKey: string; descKey: string }> = {
  COMPOSER: {
    titleKey: 'settings.features.composerTitle',
    descKey: 'settings.features.composerDesc',
  },
  FLOW: {
    titleKey: 'settings.features.flowTitle',
    descKey: 'settings.features.flowDesc',
  },
  MISSIONS: {
    titleKey: 'settings.features.missionsTitle',
    descKey: 'settings.features.missionsDesc',
  },
  PARALLEL_CHALLENGE: {
    titleKey: 'settings.features.parallelChallengeTitle',
    descKey: 'settings.features.parallelChallengeDesc',
  },
  COMPARISON_LAB: {
    titleKey: 'settings.features.comparisonLabTitle',
    descKey: 'settings.features.comparisonLabDesc',
  },
  VOICE_ACTIONS: {
    titleKey: 'settings.features.voiceActionsTitle',
    descKey: 'settings.features.voiceActionsDesc',
  },
  NEX_PROACTIVITY: {
    titleKey: 'settings.features.nexProactivityTitle',
    descKey: 'settings.features.nexProactivityDesc',
  },
  DESIGN_INSPECT: {
    titleKey: 'settings.features.designInspectTitle',
    descKey: 'settings.features.designInspectDesc',
  },
  VISUAL_DIFF: {
    titleKey: 'settings.features.visualDiffTitle',
    descKey: 'settings.features.visualDiffDesc',
  },
  SMART_LAYOUTS: {
    titleKey: 'settings.features.smartLayoutsTitle',
    descKey: 'settings.features.smartLayoutsDesc',
  },
  DEVICE_PREVIEW: {
    titleKey: 'settings.features.devicePreviewTitle',
    descKey: 'settings.features.devicePreviewDesc',
  },
};

export const OptionalFeaturesPanel: React.FC<{
  snapshot?: ReturnType<typeof useOptionalFeatures>;
}> = ({ snapshot }) => {
  const { t } = useTranslation();
  const storeFeatures = useOptionalFeatures();
  const features = snapshot ?? storeFeatures;

  return (
    <Card className={`${styles.card} nx-settings-card`}>
      <div className="nx-settings-card__title">
        <Sliders size={17} />
        <div>
          <strong>{t('settings.features.title')}</strong>
          <small>{t('settings.features.subtitle')}</small>
        </div>
      </div>

      <div className={styles.globalStatus}>
        <span>{t('settings.features.globalMode')}</span>
        <Badge tone={features.OPTIONAL_FEATURES_ENABLED ? 'brand' : 'default'}>
          {features.OPTIONAL_FEATURES_ENABLED
            ? t('settings.features.advancedMode')
            : t('settings.features.coreDirectMode')}
        </Badge>
      </div>

      {!features.OPTIONAL_FEATURES_ENABLED && (
        <InlineAlert tone="info" title={t('settings.features.howToEnableTitle')}>
          {t('settings.features.howToEnableDesc')}
        </InlineAlert>
      )}

      <ul className={styles.featureList} role="list" aria-label={t('settings.features.listAria')}>
        {OPTIONAL_FEATURE_KEYS.map((key) => {
          const enabled = features.FLAGS[key];
          const meta = FEATURE_LABEL_KEYS[key];
          return (
            <li key={key} className={styles.featureItem} data-testid={`feature-flag-${key}`}>
              <div className={styles.featureInfo}>
                <div className={styles.featureHeader}>
                  {enabled ? (
                    <CheckCircle2 size={15} className={styles.iconEnabled} aria-hidden="true" />
                  ) : (
                    <Circle size={15} className={styles.iconDisabled} aria-hidden="true" />
                  )}
                  <span className={styles.featureName}>{t(meta.titleKey)}</span>
                  <code className={styles.envFlag}>{`NEXUS_FEATURE_${key}=true`}</code>
                </div>
                <p className={styles.featureDesc}>{t(meta.descKey)}</p>
              </div>
              <Badge tone={enabled ? 'success' : 'default'}>
                {enabled ? t('settings.features.active') : t('settings.features.inactive')}
              </Badge>
            </li>
          );
        })}
      </ul>
    </Card>
  );
};
