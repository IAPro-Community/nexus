import React from 'react';
import { useTranslation } from 'react-i18next';
import type { HudSizePreset } from '../../platform';
import styles from './HudSizeControls.module.scss';

const LABEL_KEYS: Record<string, string> = {
  compact: 'hud.sizeCompact',
  comfortable: 'hud.sizeComfortable',
  large: 'hud.sizeLarge',
};

export interface HudSizeControlsProps {
  presets: HudSizePreset[];
  active: string;
  pending: boolean;
  onSelect: (preset: string) => void;
}

export const HudSizeControls: React.FC<HudSizeControlsProps> = ({
  presets,
  active,
  pending,
  onSelect,
}) => {
  const { t } = useTranslation();
  if (presets.length === 0) return null;
  const current = presets.find((preset) => preset.id === active);

  return (
    <div className={styles.controls} data-hud-size-controls>
      <div className={styles.header}>
        <span className={styles.caption}>{t('hud.sizeTitle')}</span>
        <span className={styles.current}>
          {current
            ? t('hud.sizeCurrent', {
                compactWidth: current.compactWidth,
                compactHeight: current.compactHeight,
                expandedWidth: current.expandedWidth,
                expandedHeight: current.expandedHeight,
              })
            : t('hud.sizeCustom')}
        </span>
      </div>
      <div className={styles.options} role="group" aria-label={t('hud.sizeTitle')}>
        {presets.map((preset) => {
          const labelKey = LABEL_KEYS[preset.id];
          const label = labelKey ? t(labelKey) : preset.label;
          return (
            <button
              key={preset.id}
              type="button"
              className={styles.option}
              data-active={preset.id === active ? 'true' : 'false'}
              aria-pressed={preset.id === active}
              disabled={pending}
              title={`${label} · ${preset.compactWidth}×${preset.compactHeight} / ${preset.expandedWidth}×${preset.expandedHeight}`}
              onClick={() => onSelect(preset.id)}
            >
              {label}
            </button>
          );
        })}
      </div>
    </div>
  );
};
