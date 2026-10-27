import React, { useEffect, useState } from 'react';
import { Megaphone } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, DropdownMenu } from '../../design-system';
import {
  announcementStorageKey, featureAnnouncements, isAnnouncementUnread,
} from '../../features/work/featureAnnouncements';
import styles from './ProductUpdatesCenter.module.scss';

/**
 * Global What's New entry. Uses the same local, versioned catalog as the
 * contextual tours; no duplicated news feed or backend polling is required.
 */
export const ProductUpdatesCenter: React.FC<{
  enabled: boolean;
  onOpenSurface?: (kind: string) => void;
}> = ({ enabled, onOpenSurface }) => {
  const { t } = useTranslation();
  const [, refresh] = useState(0);
  useEffect(() => {
    const update = () => refresh((current) => current + 1);
    window.addEventListener('storage', update);
    window.addEventListener('nexus:feature-discovery-updated', update);
    return () => {
      window.removeEventListener('storage', update);
      window.removeEventListener('nexus:feature-discovery-updated', update);
    };
  }, []);
  if (!enabled) return null;

  const unread = featureAnnouncements.filter((item) => {
    try {
      return isAnnouncementUnread(item, window.localStorage.getItem(announcementStorageKey(item.id)));
    } catch {
      return false;
    }
  });

  return (
    <DropdownMenu.Root>
      <DropdownMenu.Trigger asChild>
        <button
          type="button"
          className={styles.trigger}
          title={t('discovery.globalHint', 'Veja os novos recursos desta versão')}
          aria-label={t('discovery.globalUnread', '{{count}} novidades do Nexus', { count: unread.length })}
        >
          <Megaphone size={16} aria-hidden="true" />
          {unread.length > 0 && <Badge tone="brand">{unread.length}</Badge>}
        </button>
      </DropdownMenu.Trigger>
      <DropdownMenu.Portal>
        <DropdownMenu.Content className={styles.menu} align="end" sideOffset={8}>
          <div className={styles.heading}>
            <strong>{t('discovery.globalTitle', 'Novidades do Nexus')}</strong>
            <small>{t('discovery.globalDescription', 'Descubra melhorias na própria interface')}</small>
          </div>
          {featureAnnouncements.map((item) => {
            const newItem = unread.some((entry) => entry.id === item.id);
            return (
              <DropdownMenu.Item
                key={item.id}
                className={styles.item}
                onSelect={() => onOpenSurface?.(item.openSurface)}
                disabled={!onOpenSurface}
              >
                <span className={styles.title}>
                  {t(item.title)}
                  {newItem && <span className={styles.dot} aria-label={t('discovery.unread', 'Não lida')} />}
                </span>
                <span className={styles.body}>{t(item.summary)}</span>
                <span className={styles.action}>{t('discovery.openArea', 'Abrir área')}</span>
              </DropdownMenu.Item>
            );
          })}
        </DropdownMenu.Content>
      </DropdownMenu.Portal>
    </DropdownMenu.Root>
  );
};
