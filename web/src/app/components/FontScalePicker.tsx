import React from 'react';
import { ChevronDown, Type } from 'lucide-react';
import * as DropdownMenu from '@radix-ui/react-dropdown-menu';
import { useTranslation } from 'react-i18next';
import { useTheme } from '../../design-system';
import styles from './FontScalePicker.module.scss';

const SCALE_OPTIONS = [0.9, 1, 1.1, 1.2] as const;

export const FontScalePicker: React.FC = () => {
  const { t } = useTranslation();
  const theme = useTheme();
  const currentScale = theme.fontScale || 1;
  const currentLabel = `${Math.round(currentScale * 100)}%`;

  return (
    <div className={styles.root} data-testid="font-scale-picker">
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <button
            type="button"
            className={styles.trigger}
            aria-label={t('shell.fontScaleLabel', 'Tamanho do texto')}
            title={t('shell.fontScaleTitle', 'Ajustar tamanho do texto')}
          >
            <Type size={14} aria-hidden="true" />
            <span className={styles.value}>{currentLabel}</span>
            <ChevronDown size={11} aria-hidden="true" />
          </button>
        </DropdownMenu.Trigger>

        <DropdownMenu.Portal>
          <DropdownMenu.Content className={styles.menu} sideOffset={6} align="end">
            <DropdownMenu.Label className={styles.menuLabel}>
              {t('shell.fontScaleMenu', 'Tamanho do texto')}
            </DropdownMenu.Label>
            {SCALE_OPTIONS.map((scale) => {
              const percentage = Math.round(scale * 100);
              const active = Math.abs(currentScale - scale) < 0.04;
              return (
                <DropdownMenu.Item key={scale} asChild onSelect={() => theme.setFontScale(scale)}>
                  <button type="button" className={styles.option} data-active={active}>
                    <span>{t(`shell.fontScale${percentage}`, `${percentage}%`)}</span>
                    {active && <span aria-hidden="true">✓</span>}
                  </button>
                </DropdownMenu.Item>
              );
            })}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </div>
  );
};
