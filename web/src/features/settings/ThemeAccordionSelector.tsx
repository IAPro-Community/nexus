import React, { useState } from 'react';
import { Check, ChevronDown, ChevronRight } from 'lucide-react';
import {
  ChoiceGroup,
  ChoiceOption,
  THEME_PRESETS,
  getThemePresetPalette,
  useTheme,
  type ThemePresetKey,
} from '../../design-system';
import { useTranslation } from 'react-i18next';
import styles from './SettingsSurface.module.scss';

/**
 * Appearance presets own their accordion and keyboard selection behavior.
 * Keep rendering and persistence delegated to existing design-system primitives.
 */
interface ThemeCategory {
  id: string;
  title: string;
  description: string;
  presets: ThemePresetKey[];
}

const THEME_CATEGORIES: ThemeCategory[] = [
  {
    id: 'soft-gradient',
    title: 'Degradê Suave',
    description: 'Grafite quase preto, profundidade discreta e acentos pouco saturados.',
    presets: ['nexus-dusk'],
  },
  {
    id: 'flat-essentials',
    title: 'Essenciais Flat',
    description: 'Menos efeitos, fundos sólidos e acentos discretos para uso prolongado.',
    presets: ['slate-flat', 'paper-flat', 'clay-flat'],
  },
  {
    id: 'dark-modern',
    title: 'Dark Modern',
    description: 'Paletas escuras premium, alto contraste de código e atmosfera imersiva.',
    presets: [
      'nexus-dark',
      'midnight',
      'dracula',
      'graphite-focus',
      'vasco-noir',
      'aurora-ops',
      'obsidian',
    ],
  },
  {
    id: 'light-clean',
    title: 'Light Clean',
    description: 'Interfaces diurnas, excelente legibilidade e visual limpo.',
    presets: ['nexus-light', 'nord-light', 'solarized-light'],
  },
  {
    id: 'classic-code',
    title: 'Clássicos do Código',
    description: 'Esquemas consagrados por gerações de editores de texto.',
    presets: ['monokai', 'nord', 'solarized-dark', 'ember-studio'],
  },
  {
    id: 'accessibility',
    title: 'Acessibilidade Extrema',
    description: 'Contraste máximo e certificação WCAG 2.2 AA para visualização sem fadiga.',
    presets: ['high-contrast-dark', 'high-contrast-light'],
  },
];

export const ThemeAccordionSelector: React.FC<{
  theme: ReturnType<typeof useTheme>;
}> = ({ theme }) => {
  const { t } = useTranslation();
  const [openCategories, setOpenCategories] = useState<Record<string, boolean>>(() => {
    // Abre por padrão a categoria onde o preset ativo se encontra
    const active = theme.preset || 'nexus-dark';
    const found = THEME_CATEGORIES.find((cat) => cat.presets.includes(active));
    return { [found ? found.id : 'dark-modern']: true };
  });

  const toggleCategory = (catId: string) => {
    setOpenCategories((curr) => ({ ...curr, [catId]: !curr[catId] }));
  };

  return (
    <div className={`nx-theme-accordion-container ${styles.accordionContainer}`}>
      {/* Grupos de Categorias em Accordion WAI-ARIA */}
      {THEME_CATEGORIES.map((cat) => {
        const isOpen = Boolean(openCategories[cat.id]);
        const headerId = `theme-cat-hdr-${cat.id}`;
        const panelId = `theme-cat-panel-${cat.id}`;

        return (
          <div key={cat.id} className={`nx-theme-cat-group ${styles.themeGroup}`}>
            <button
              type="button"
              id={headerId}
              aria-expanded={isOpen}
              aria-controls={panelId}
              onClick={() => toggleCategory(cat.id)}
              className={`nx-theme-accordion-hdr ${styles.themeHeader}`}
            >
              <div className={styles.themeHeaderLeft}>
                {isOpen ? (
                  <ChevronDown size={15} color="var(--nx-accent)" />
                ) : (
                  <ChevronRight size={15} color="var(--nx-muted)" />
                )}
                <div>
                  <div className={styles.themeTitle}>{cat.title}</div>
                  <div className={styles.themeDesc}>{cat.description}</div>
                </div>
              </div>
              <span className={`nx-badge ${styles.themeBadge}`} data-tone="brand">
                {cat.presets.length} temas
              </span>
            </button>

            {/* Painel do Accordion com Grid de Presets */}
            {isOpen && (
              <div
                id={panelId}
                role="region"
                aria-labelledby={headerId}
                className={styles.themePanel}
              >
                <ChoiceGroup
                  className={styles.presetGrid}
                  ariaLabel={`${cat.title}: ${t('settings.themePresets', 'Temas visuais')}`}
                >
                  {cat.presets.map((presetKey) => {
                    const preset = THEME_PRESETS[presetKey];
                    if (!preset) return null;
                    const isSelected = theme.preset === presetKey && !theme.isCustomized;
                    const palette = getThemePresetPalette(preset);

                    return (
                      <ChoiceOption
                        key={presetKey}
                        selected={isSelected}
                        onSelect={() => theme.setPreset(presetKey as ThemePresetKey)}
                        className={styles.themePresetItem}
                      >
                        <span className={styles.themePresetName} data-selected={isSelected}>
                          <span>{preset.name}</span>
                          {isSelected && <Check size={14} color="var(--nx-accent)" />}
                        </span>

                        {/* Amostras de Cores (Swatches) */}
                        <span className={styles.themeSwatches} aria-hidden="true">
                          <span
                            title={`Fundo: ${palette.bg}`}
                            className={styles.themeSwatchBg}
                            style={{ background: palette.bg }}
                          />
                          <span
                            title={`Acento: ${palette.accent}`}
                            className={styles.themeSwatchAccent}
                            style={{ background: palette.accent }}
                          />
                        </span>
                      </ChoiceOption>
                    );
                  })}
                </ChoiceGroup>
              </div>
            )}
          </div>
        );
      })}
    </div>
  );
};
