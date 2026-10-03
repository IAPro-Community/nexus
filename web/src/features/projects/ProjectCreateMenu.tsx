import React from 'react';
import { ChevronDown, Plus, Sparkles, TerminalSquare } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { DropdownMenu } from '../../design-system';

export type ProjectCreateMenuProps = {
  onNewAgent?: () => void;
  onNewAISession?: () => void;
  onProjectShell?: () => void;
  size?: 'sm' | 'xs';
  variant?: 'topbar' | 'compact' | 'ghost';
  label?: string;
  className?: string;
};

export const ProjectCreateMenu: React.FC<ProjectCreateMenuProps> = ({
  onNewAgent,
  onNewAISession,
  onProjectShell,
  size = 'sm',
  variant = 'topbar',
  label,
  className,
}) => {
  const { t } = useTranslation();

  if (!onNewAgent && !onNewAISession && !onProjectShell) return null;

  const defaultLabel = label ?? t('overview.create');

  return (
    <div
      className={`nx-create-menu ${className ? className : ''}`}
      data-variant={variant}
      data-size={size}
    >
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <button
            type="button"
            data-testid="topbar-create-menu-btn"
            className="nx-create-menu__trigger nx-button"
            data-size={size}
            data-tone={variant === 'ghost' ? undefined : 'brand'}
            title={t('overview.createMenuTooltip')}
          >
            <Plus size={size === 'xs' ? 12 : 13} />
            <span className="nx-create-menu__label">{defaultLabel}</span>
            <ChevronDown size={11} className="nx-create-menu__chevron" />
          </button>
        </DropdownMenu.Trigger>

        <DropdownMenu.Portal>
          <DropdownMenu.Content
            className="nx-create-menu__panel"
            aria-label={t('overview.createMenuTooltip')}
            align="start"
            sideOffset={4}
            collisionPadding={8}
          >
            {onNewAgent && (
              <DropdownMenu.Item className="nx-create-menu__item" onSelect={onNewAgent}>
                <div className="nx-create-menu__icon nx-create-menu__icon--agent">
                  <Plus size={13} />
                </div>
                <div className="nx-create-menu__text">
                  <strong>{t('overview.newAgent')}</strong>
                  <small>{t('overview.newAgentHint')}</small>
                </div>
              </DropdownMenu.Item>
            )}

            {onNewAISession && (
              <DropdownMenu.Item className="nx-create-menu__item" onSelect={onNewAISession}>
                <div className="nx-create-menu__icon nx-create-menu__icon--session">
                  <Sparkles size={13} />
                </div>
                <div className="nx-create-menu__text">
                  <strong>{t('overview.newAISession')}</strong>
                  <small>{t('overview.newAISessionHint')}</small>
                </div>
              </DropdownMenu.Item>
            )}

            {onProjectShell && (
              <DropdownMenu.Item className="nx-create-menu__item" onSelect={onProjectShell}>
                <div className="nx-create-menu__icon nx-create-menu__icon--terminal">
                  <TerminalSquare size={13} />
                </div>
                <div className="nx-create-menu__text">
                  <strong>{t('overview.projectShell')}</strong>
                  <small>{t('overview.projectShellHint')}</small>
                </div>
              </DropdownMenu.Item>
            )}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </div>
  );
};
