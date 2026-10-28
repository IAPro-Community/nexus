import React, { useCallback, useEffect, useRef, useState } from 'react';
import { Bell, Compass, Sparkles } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, DropdownMenu, Tooltip } from '../../design-system';
import { ProductTour } from '../../app/tour/ProductTour';
import { availableTourSteps } from '../../app/tour/tour';
import {
  announcementStorageKey,
  announcementTourIndex,
  announcementsForSurface,
  isAnnouncementUnread,
  type FeatureAnnouncement,
} from './featureAnnouncements';
import {
  featureTourSteps,
  featureTourStorageKey,
  parseFeatureTourProgress,
  type FeatureTourSurface,
} from './featureTourDefinitions';
import {
  FEATURE_REVEAL_REQUEST_EVENT, consumeFeatureReveal,
  hasPendingFeatureReveal,
} from './featureReveal';
import styles from './FeatureTour.module.scss';

const readProgress = (key: string, length: number): number => {
  try {
    return parseFeatureTourProgress(window.localStorage.getItem(key), length);
  } catch {
    return 0;
  }
};

/** Never search the whole page for contextual tour targets: Nexus can keep
 * several projects and split workspaces mounted at the same time. */
export function hasTourTargetInScope(
  scope: Element | null | undefined,
  selector: string,
): boolean {
  const el = scope?.querySelector(selector) as HTMLElement | null;
  if (!el) return false;
  const r = el.getBoundingClientRect();
  return r.width > 0 && r.height > 0 && r.bottom > 0 && r.right > 0;
}

/**
 * Scope-aware feature discovery, built on the design system's Radix Tooltip
 * and DropdownMenu and the already-installed ProductTour. No second tour
 * runtime, notification SDK, tracking endpoint or new bundled dependency.
 *
 * New functionality is registered once in featureAnnouncements.ts; revision
 * changes cause a new, dismissible, unobtrusive notice on the relevant surface.
 */
export const FeatureTour: React.FC<{
  projectId: string;
  surface: FeatureTourSurface;
}> = ({ projectId, surface }) => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const hostRef = useRef<HTMLDivElement>(null);
  const visibleTarget = useCallback(
    (selector: string) =>
      hasTourTargetInScope(
        hostRef.current?.closest('[data-feature-tour-scope]'),
        selector,
      ),
    [],
  );
  const [initialIndex, setInitialIndex] = useState(0);
  const [updates, setUpdates] = useState<FeatureAnnouncement[]>([]);
  const [, refreshReadState] = useState(0);
  const steps = featureTourSteps[surface];
  const key = featureTourStorageKey(projectId, surface);
  const seen = (item: FeatureAnnouncement): boolean => {
    try {
      return !isAnnouncementUnread(
        item,
        window.localStorage.getItem(announcementStorageKey(item.id)),
      );
    } catch {
      // Do not pester users when persistence is restricted.
      return true;
    }
  };
  const unread = updates.filter((item) => !seen(item));

  useEffect(() => {
    const onUpdated = () => refreshReadState((count) => count + 1);
    window.addEventListener('storage', onUpdated);
    window.addEventListener('nexus:feature-discovery-updated', onUpdated);
    return () => {
      window.removeEventListener('storage', onUpdated);
      window.removeEventListener('nexus:feature-discovery-updated', onUpdated);
    };
  }, []);

  useEffect(() => {
    // An explicit selection in the global "What's New" menu opens the
    // appropriate workspace first, then points the tour at its real control.
    // Conditional panels may mount after the workspace itself.
    let observer: MutationObserver | null = null;
    let expiry: number | undefined;
    let frame: number | undefined;
    const stopObserving = () => {
      observer?.disconnect();
      observer = null;
      if (expiry !== undefined) window.clearTimeout(expiry);
      expiry = undefined;
    };
    const tryReveal = () => {
      const item = consumeFeatureReveal(projectId, surface, visibleTarget);
      if (!item) return false;
      stopObserving();
      const visibleSteps = availableTourSteps(steps, visibleTarget);
      setInitialIndex(announcementTourIndex(item, visibleSteps.map((step) => step.id)));
      setOpen(true);
      try {
        window.localStorage.setItem(announcementStorageKey(item.id), String(item.revision));
      } catch { /* read marker is local-only and optional */ }
      window.dispatchEvent(new Event('nexus:feature-discovery-updated'));
      return true;
    };
    const onRequest = () => {
      if (tryReveal()) return;
      if (observer || !hasPendingFeatureReveal(projectId, surface)) return;
      observer = new MutationObserver(() => { tryReveal(); });
      observer.observe(document.body, {
        childList: true, subtree: true, attributes: true,
        attributeFilter: ['hidden', 'style', 'aria-hidden'],
      });
      expiry = window.setTimeout(stopObserving, 30_000);
    };
    window.addEventListener(FEATURE_REVEAL_REQUEST_EVENT, onRequest);
    frame = window.requestAnimationFrame(onRequest);
    return () => {
      window.removeEventListener(FEATURE_REVEAL_REQUEST_EVENT, onRequest);
      if (frame !== undefined) window.cancelAnimationFrame(frame);
      stopObserving();
    };
  }, [projectId, surface, steps, visibleTarget]);

  useEffect(() => {
    setOpen(false);
    setUpdates([]);
    // Target content is rendered alongside this component and may be
    // conditional on a selected draft or session. Discover post-commit.
    const frame = window.requestAnimationFrame(() => {
      setUpdates(announcementsForSurface(surface, visibleTarget));
    });
    return () => window.cancelAnimationFrame(frame);
  }, [projectId, surface, visibleTarget]);

  const markRead = (items: readonly FeatureAnnouncement[]) => {
    items.forEach((item) => {
      try {
        window.localStorage.setItem(announcementStorageKey(item.id), String(item.revision));
      } catch {
        /* local-only persistence is optional */
      }
    });
    refreshReadState((current) => current + 1);
    window.dispatchEvent(new Event('nexus:feature-discovery-updated'));
  };

  // Explicit scan on opening a menu lets newly mounted conditional controls
  // become discoverable after selecting a plan without adding DOM observers.
  const refreshUpdates = () => {
    setUpdates(announcementsForSurface(surface, visibleTarget));
  };
  const save = (index: number) => {
    try {
      window.localStorage.setItem(key, String(index));
    } catch {
      /* tour remains usable without storage */
    }
  };
  const begin = (item?: FeatureAnnouncement) => {
    refreshUpdates();
    const visibleSteps = availableTourSteps(steps, visibleTarget);
    const savedIndex = readProgress(key, visibleSteps.length);
    const first = item
      ? announcementTourIndex(
          item,
          visibleSteps.map((step) => step.id),
        )
      : savedIndex;
    if (item) markRead([item]);
    setInitialIndex(first);
    setOpen(true);
  };
  const finish = () => {
    save(0);
    setOpen(false);
  };
  const first = unread[0];
  return (
    <>
      <div ref={hostRef} className={styles.root} data-feature-discovery={surface}>
        <Tooltip content={t('discovery.helpHint', 'Passo a passo interativo desta área')}>
          <Button size="sm" onClick={() => begin()}>
            <Compass size={14} aria-hidden="true" />
            {t('tour.featureOpen', 'Conhecer esta área')}
          </Button>
        </Tooltip>
        {updates.length > 0 && (
          <DropdownMenu.Root
            onOpenChange={(next) => {
              if (next) refreshUpdates();
            }}
          >
            <DropdownMenu.Trigger asChild>
              <Button size="sm" aria-label={t('discovery.updates', 'Novidades desta área')}>
                <Bell size={14} aria-hidden="true" />
                {t('discovery.updates', 'Novidades')}
                {unread.length > 0 && <Badge tone="brand">{unread.length}</Badge>}
              </Button>
            </DropdownMenu.Trigger>
            <DropdownMenu.Portal>
              <DropdownMenu.Content className={styles.menu} align="end" sideOffset={8}>
                <div className={styles.menuHeading}>
                  {t('discovery.updates', 'Novidades desta área')}
                </div>
                {updates.map((item) => (
                  <DropdownMenu.Item
                    key={item.id}
                    className={styles.menuItem}
                    onSelect={() => begin(item)}
                  >
                    <span className={styles.menuTitle}>
                      {t(item.title)}
                      {!seen(item) && (
                        <span
                          className={styles.unreadDot}
                          aria-label={t('discovery.unread', 'Não lida')}
                        />
                      )}
                    </span>
                    <span className={styles.menuSummary}>{t(item.summary)}</span>
                    <span className={styles.menuAction}>
                      {t('discovery.showLocation', 'Mostrar no componente')}
                    </span>
                  </DropdownMenu.Item>
                ))}
                {unread.length > 0 && (
                  <DropdownMenu.Item className={styles.markRead} onSelect={() => markRead(unread)}>
                    {t('discovery.markAllRead', 'Marcar todas como lidas')}
                  </DropdownMenu.Item>
                )}
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        )}
        {first && (
          <div className={styles.notice} role="status">
            <Sparkles size={14} aria-hidden="true" />
            <span>{t(first.title)}</span>
            <Button size="sm" onClick={() => begin(first)}>
              {t('discovery.showLocation', 'Mostrar no componente')}
            </Button>
            <Button size="sm" onClick={() => markRead([first])}>
              {t('discovery.dismiss', 'Dispensar')}
            </Button>
          </div>
        )}
      </div>
      <ProductTour
        open={open}
        steps={steps}
        targetScope={hostRef.current?.closest('[data-feature-tour-scope]')}
        initialIndex={initialIndex}
        onStepChange={save}
        onClose={() => setOpen(false)}
        onComplete={finish}
      />
    </>
  );
};
