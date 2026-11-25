import React from 'react';
import { LayoutGrid, RotateCcw } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { DropdownMenu } from '../design-system';
import { ARRANGE_MENU_PRESETS, WORKSPACE_RECIPES, type ArrangeMenuPreset } from './arrangePresets';
import { useOptionalFeatures } from '../app/composerFlowFlag';

export const ArrangeLayoutMenu: React.FC<{
  activePreset: ArrangeMenuPreset;
  onSelect: (preset: ArrangeMenuPreset) => void;
  onReset?: () => void;
}> = ({ activePreset, onSelect, onReset }) => {
  const { SMART_LAYOUTS_ENABLED } = useOptionalFeatures();
  const { t } = useTranslation();

  return (
    <div className="nx-arrange-menu">
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <button
            type="button"
            className="nx-arrange-menu__trigger"
            title={t('workspace.arrangeWindows', 'Arranjar janelas')}
            aria-label={t('workspace.arrangeWindows', 'Arranjar janelas')}
          >
            <LayoutGrid size={12} />
            <span>{t('workspace.arrange', 'Arranjar')}</span>
          </button>
        </DropdownMenu.Trigger>

        <DropdownMenu.Portal>
          <DropdownMenu.Content
            className="nx-arrange-menu__panel"
            aria-label={t('workspace.arrangeWindows', 'Arranjar janelas')}
            align="end"
            sideOffset={6}
            collisionPadding={8}
          >
            <DropdownMenu.RadioGroup
              value={activePreset}
              onValueChange={(value) => onSelect(value as ArrangeMenuPreset)}
            >
              {ARRANGE_MENU_PRESETS.filter(
                (preset) => SMART_LAYOUTS_ENABLED || preset.id !== 'automatic',
              ).map((preset) => (
                <DropdownMenu.RadioItem
                  key={preset.id}
                  value={preset.id}
                  className="nx-arrange-menu__item"
                  data-active={activePreset === preset.id ? 'true' : undefined}
                >
                  <span>{preset.label}</span>
                  <small>{preset.hint}</small>
                </DropdownMenu.RadioItem>
              ))}
            </DropdownMenu.RadioGroup>

            <DropdownMenu.Separator className="nx-arrange-menu__separator" />
            <DropdownMenu.Label className="nx-arrange-menu__item">
              <span>{t('workspace.recipes', 'Receitas de trabalho')}</span>
            </DropdownMenu.Label>
            {WORKSPACE_RECIPES.filter(
              (recipe) => SMART_LAYOUTS_ENABLED || recipe.preset !== 'automatic',
            ).map((recipe) => (
              <DropdownMenu.Item
                key={recipe.id}
                className="nx-arrange-menu__item"
                onSelect={() => onSelect(recipe.preset)}
              >
                <span>{t(recipe.labelKey)}</span>
                <small>{t(recipe.hintKey)}</small>
              </DropdownMenu.Item>
            ))}

            {onReset && (
              <>
                <DropdownMenu.Separator className="nx-arrange-menu__separator" />
                <DropdownMenu.Item className="nx-arrange-menu__item" onSelect={() => onReset()}>
                  <span className="nx-arrange-menu__reset-label">
                    <RotateCcw size={12} />
                    <span>{t('workspace.resetLayout', 'Redefinir layout')}</span>
                  </span>
                  <small>{t('workspace.restoreDefault', 'Restaurar padrão')}</small>
                </DropdownMenu.Item>
              </>
            )}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </div>
  );
};
