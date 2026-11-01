import React from 'react';
import { GripHorizontal, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { MascotAttention, MascotMood } from './mascotModel';
import { NexPixelSprite } from './NexPixelSprite';
import type { GuardianMood } from './guardianModel';
import { NexIdleDirector, type NexIdleAction, type NexIdleMood } from './nexIdleDirector';
import styles from './GuardianMascot.module.scss';
import hudStyles from './NexusHud.module.scss';

interface GuardianMascotProps {
  mood: MascotMood;
  count: number;
  expanded: boolean;
  motion: boolean;
  bubble?: MascotAttention;
  native?: boolean;
  popoverId?: string;
  onExpand: () => void;
  onOpenAttention: (item: MascotAttention) => void;
  onSnooze: (item: MascotAttention) => void;
  onMoveKeys?: (event: React.KeyboardEvent<HTMLButtonElement>) => void;
}

export const GuardianMascot: React.FC<GuardianMascotProps> = ({
  mood,
  count,
  expanded,
  motion,
  bubble,
  native = false,
  popoverId,
  onExpand,
  onOpenAttention,
  onSnooze,
  onMoveKeys,
}) => {
  const { t } = useTranslation();
  const motionEnabled = motion && mood !== 'offline';
  const [idleAction, setIdleAction] = React.useState<NexIdleAction | null>(null);
  const mascotRoot = React.useRef<HTMLDivElement>(null);
  const idleVisibility = React.useRef(false);
  const spriteMood: GuardianMood = mood === 'attention' ? 'warning' : mood;
  const idleMood: NexIdleMood = mood === 'attention' ? 'warning' : mood;
  const idleDirector = React.useMemo(() => new NexIdleDirector(setIdleAction), []);
  const mascotLabel =
    count > 0 && !expanded
      ? t('hud.mascot.openPendingCount', { count })
      : t(expanded ? 'hud.mascot.collapse' : 'hud.mascot.expand');

  const updateIdleEnvironment = React.useCallback(() => {
    idleDirector.updateEnvironment({
      mood: idleMood,
      motionEnabled: motion,
      visible:
        typeof document !== 'undefined' &&
        document.visibilityState === 'visible' &&
        idleVisibility.current,
      reducedMotion:
        typeof window !== 'undefined' &&
        window.matchMedia('(prefers-reduced-motion: reduce)').matches,
    });
  }, [idleDirector, idleMood, motion]);

  React.useEffect(() => {
    updateIdleEnvironment();
    const media = window.matchMedia('(prefers-reduced-motion: reduce)');
    const root = mascotRoot.current;
    const observer =
      root && typeof IntersectionObserver !== 'undefined'
        ? new IntersectionObserver(
            ([entry]) => {
              const style = getComputedStyle(entry.target);
              idleVisibility.current =
                entry.isIntersecting &&
                entry.intersectionRatio > 0 &&
                style.visibility !== 'hidden' &&
                style.display !== 'none' &&
                Number(style.opacity) > 0;
              updateIdleEnvironment();
            },
            { threshold: 0.01 },
          )
        : null;
    if (root && observer) observer.observe(root);
    else idleVisibility.current = Boolean(root?.getClientRects().length);
    updateIdleEnvironment();
    document.addEventListener('visibilitychange', updateIdleEnvironment);
    media.addEventListener('change', updateIdleEnvironment);
    return () => {
      observer?.disconnect();
      document.removeEventListener('visibilitychange', updateIdleEnvironment);
      media.removeEventListener('change', updateIdleEnvironment);
    };
  }, [updateIdleEnvironment]);

  React.useEffect(() => {
    idleDirector.start();
    return () => idleDirector.stop();
  }, [idleDirector]);

  const notifyInteraction = () => idleDirector.notifyInteraction();

  return (
    <div
      ref={mascotRoot}
      className={styles.root}
      data-mood={mood}
      data-motion={motionEnabled ? 'on' : 'off'}
      data-native={native ? 'true' : 'false'}
    >
      {bubble && !expanded && (
        <aside
          className={styles.speech}
          aria-label={t('hud.mascot.attentionLabel')}
          data-mascot-bubble
          data-hud-hit-area={native ? 'true' : undefined}
        >
          <button
            className={styles.close}
            type="button"
            aria-label={t('hud.mascot.dismissAttention')}
            data-hud-hit-area="true"
            onClick={() => onSnooze(bubble)}
          >
            <X size={13} aria-hidden="true" />
          </button>
          <strong>{t(bubble.titleKey)}</strong>
          <span>{t(bubble.detailKey)}</span>
          <div className={styles.bubbleActions}>
            <button type="button" data-hud-hit-area="true" onClick={() => onOpenAttention(bubble)}>
              {t('hud.mascot.open')}
            </button>
            <button type="button" data-hud-hit-area="true" onClick={() => onSnooze(bubble)}>
              {t('hud.mascot.snooze')}
            </button>
          </div>
        </aside>
      )}
      <div className={styles.group}>
        {native && (
          <button
            type="button"
            className={styles.dragHandle}
            data-hud-drag-handle="true"
            data-hud-hit-area="true"
            onKeyDown={onMoveKeys}
            title={t('hud.dragTitle')}
            aria-label={t('hud.dragLabel')}
          >
            <GripHorizontal size={13} aria-hidden="true" />
          </button>
        )}
        <button
          type="button"
          className={styles.guardianButton}
          aria-label={mascotLabel}
          aria-expanded={expanded}
          aria-controls={expanded ? popoverId : undefined}
          data-hud-hit-area="true"
          onClick={() => {
            notifyInteraction();
            onExpand();
          }}
          onPointerEnter={notifyInteraction}
          onFocus={notifyInteraction}
          data-hud-mascot="true"
        >
          <span
            className={hudStyles.guardian}
            data-mood={spriteMood}
            data-motion={motionEnabled ? 'on' : 'off'}
            data-blink={idleAction === 'idle_blink' ? 'true' : 'false'}
            aria-hidden="true"
          >
            <NexPixelSprite
              mood={spriteMood}
              idleAction={motionEnabled && mood === 'idle' ? idleAction : null}
            />
          </span>
          {count > 0 && (
            <span className={styles.badge} aria-hidden="true">
              {count > 9 ? '9+' : count}
            </span>
          )}
        </button>
      </div>
    </div>
  );
};
