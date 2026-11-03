import React from 'react';
import { GripHorizontal, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { MascotAttention, MascotMood } from './mascotModel';
import { NexPixelSprite } from './NexPixelSprite';
import type { GuardianMood } from './guardianModel';
import { NexIdleDirector, type NexIdleAction, type NexIdleMood } from './nexIdleDirector';
import { resolveNexCompanionBehavior, type NexCompanionInteraction } from './nexCompanionBehavior';
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
  /** Event ID from the canonical Attention Center completed bucket, never inferred. */
  completionEventId?: string | null;
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
  completionEventId,
}) => {
  const { t } = useTranslation();
  const [reducedMotion, setReducedMotion] = React.useState(
    () =>
      typeof window !== 'undefined' &&
      window.matchMedia('(prefers-reduced-motion: reduce)').matches,
  );
  const [visualVisible, setVisualVisible] = React.useState(false);
  const motionEnabled = motion && mood !== 'offline' && !reducedMotion && visualVisible;
  const [idleAction, setIdleAction] = React.useState<NexIdleAction | null>(null);
  const [interaction, setInteraction] = React.useState<NexCompanionInteraction>('none');
  const [lookSide, setLookSide] = React.useState<'left' | 'right'>('right');
  const [resolutionPlaying, setResolutionPlaying] = React.useState(false);
  const lastResolutionId = React.useRef<string | null>(null);
  const greetingAt = React.useRef(0);
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
      reducedMotion,
    });
  }, [idleDirector, idleMood, motion, reducedMotion]);

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
              setVisualVisible(document.visibilityState === 'visible' && idleVisibility.current);
              updateIdleEnvironment();
            },
            { threshold: 0.01 },
          )
        : null;
    if (root && observer) observer.observe(root);
    else {
      idleVisibility.current = Boolean(root?.getClientRects().length);
      setVisualVisible(document.visibilityState === 'visible' && idleVisibility.current);
    }
    updateIdleEnvironment();
    const syncReducedMotion = () => setReducedMotion(media.matches);
    const syncVisibility = () => {
      setVisualVisible(document.visibilityState === 'visible' && idleVisibility.current);
      updateIdleEnvironment();
    };
    document.addEventListener('visibilitychange', syncVisibility);
    media.addEventListener('change', syncReducedMotion);
    return () => {
      observer?.disconnect();
      document.removeEventListener('visibilitychange', syncVisibility);
      media.removeEventListener('change', syncReducedMotion);
    };
  }, [updateIdleEnvironment]);

  React.useEffect(() => {
    idleDirector.start();
    return () => idleDirector.stop();
  }, [idleDirector]);

  React.useEffect(() => {
    if (!completionEventId || completionEventId === lastResolutionId.current) return;
    lastResolutionId.current = completionEventId;
    if (!motionEnabled || mood !== 'idle') return;
    setResolutionPlaying(true);
    const timer = window.setTimeout(() => setResolutionPlaying(false), 1400);
    return () => window.clearTimeout(timer);
  }, [completionEventId, motionEnabled, mood]);

  React.useEffect(() => {
    if (!motionEnabled || mood !== 'idle') setResolutionPlaying(false);
  }, [motionEnabled, mood]);

  const behavior = resolveNexCompanionBehavior({
    mood,
    idleAction,
    interaction,
    resolved: resolutionPlaying,
    motionAllowed: motionEnabled,
  });

  const notifyInteraction = () => idleDirector.notifyInteraction();

  return (
    <div
      ref={mascotRoot}
      className={styles.root}
      data-mood={mood}
      data-motion={motionEnabled ? 'on' : 'off'}
      data-native={native ? 'true' : 'false'}
      data-nex-behavior={behavior}
      data-nex-look={lookSide}
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
            setInteraction('greet');
            onExpand();
          }}
          onPointerEnter={() => {
            greetingAt.current = Date.now();
            setInteraction('greet');
            notifyInteraction();
          }}
          onPointerMove={(event) => {
            if (event.pointerType !== 'mouse' || Date.now() - greetingAt.current <= 500) return;
            const rect = event.currentTarget.getBoundingClientRect();
            setLookSide(event.clientX < rect.left + rect.width / 2 ? 'left' : 'right');
            setInteraction('look');
          }}
          onPointerLeave={() => setInteraction('none')}
          onFocus={() => {
            setInteraction('greet');
            notifyInteraction();
          }}
          onBlur={() => setInteraction('none')}
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
