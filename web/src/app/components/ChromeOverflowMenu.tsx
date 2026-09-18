import React from 'react';
import {
  ArrowUpCircle,
  Command,
  Globe,
  Maximize2,
  Minimize2,
  MoonStar,
  MoreHorizontal,
  CircleHelp,
} from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { DropdownMenu, IconButton } from '../../design-system';
import styles from './ChromeOverflowMenu.module.scss';

export const ChromeOverflowMenu: React.FC<{
  zenMode?: boolean;
  updateAvailable?: boolean;
  onCommand: () => void;
  onOpenWelcome: () => void;
  onSettings: (tab?: 'appearance' | 'updates' | 'remote') => void;
  onToggleZenMode?: () => void;
}> = ({
  zenMode = false,
  updateAvailable = false,
  onCommand,
  onOpenWelcome,
  onSettings,
  onToggleZenMode,
}) => {
  const { t } = useTranslation();

  return (
    <div className={`nx-chrome-compact ${styles.root}`}>
      <DropdownMenu.Root>
        <DropdownMenu.Trigger asChild>
          <IconButton className={styles.trigger} label={t('shell.moreActions', 'More actions')}>
            <MoreHorizontal size={18} aria-hidden="true" />
          </IconButton>
        </DropdownMenu.Trigger>
        <DropdownMenu.Portal>
          <DropdownMenu.Content className={styles.menu} align="end" sideOffset={6}>
            <DropdownMenu.Item asChild onSelect={onCommand}>
              <button type="button" className={styles.item}>
                <Command size={16} aria-hidden="true" />
                {t('shell.search')}
              </button>
            </DropdownMenu.Item>
            <DropdownMenu.Item asChild onSelect={() => onSettings('appearance')}>
              <button type="button" className={styles.item}>
                <MoonStar size={16} aria-hidden="true" />
                {t('shell.appearance')}
              </button>
            </DropdownMenu.Item>
            <DropdownMenu.Item asChild onSelect={() => onSettings('remote')}>
              <button type="button" className={styles.item}>
                <Globe size={16} aria-hidden="true" />
                {t('settings.remote.title', 'Remote access')}
              </button>
            </DropdownMenu.Item>
            {updateAvailable ? (
              <DropdownMenu.Item asChild onSelect={() => onSettings('updates')}>
                <button type="button" className={styles.item}>
                  <ArrowUpCircle size={16} aria-hidden="true" />
                  {t('settings.updates')}
                </button>
              </DropdownMenu.Item>
            ) : null}
            <DropdownMenu.Item asChild onSelect={onOpenWelcome}>
              <button type="button" className={styles.item}>
                <CircleHelp size={16} aria-hidden="true" />
                {t('shell.tour')}
              </button>
            </DropdownMenu.Item>
            {onToggleZenMode ? (
              <DropdownMenu.Item asChild onSelect={onToggleZenMode}>
                <button
                  type="button"
                  className={styles.item}
                  data-active={zenMode ? 'true' : 'false'}
                >
                  {zenMode ? (
                    <Minimize2 size={16} aria-hidden="true" />
                  ) : (
                    <Maximize2 size={16} aria-hidden="true" />
                  )}
                  {zenMode ? t('workspace.exitFocusMode') : t('workspace.focusMode')}
                </button>
              </DropdownMenu.Item>
            ) : null}
          </DropdownMenu.Content>
        </DropdownMenu.Portal>
      </DropdownMenu.Root>
    </div>
  );
};
