import React from 'react';
import { GripHorizontal, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { MascotAttention, MascotMood } from './mascotModel';
import { NexPixelSprite } from './NexPixelSprite';
import type { GuardianMood } from './guardianModel';
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
  const [blink, setBlink] = React.useState(false);
  const spriteMood: GuardianMood = mood === 'attention' ? 'warning' : mood;
  const mascotLabel =
    count > 0 && !expanded
      ? t('hud.mascot.openPendingCount', { count })
      : t(expanded ? 'hud.mascot.collapse' : 'hud.mascot.expand');

  React.useEffect(() => {
    setBlink(false);
    if (!motionEnabled || mood !== 'idle') return;

    let blinkTimer: number | undefined;
    let reopenTimer: number | undefined;
    let disposed = false;
    const scheduleBlink = () => {
      blinkTimer = window.setTimeout(() => {
        if (disposed) return;
        setBlink(true);
        reopenTimer = window.setTimeout(() => {
          if (disposed) return;
          setBlink(false);
          scheduleBlink();
        }, 120);
      }, 6400);
    };

    scheduleBlink();
    return () => {
      disposed = true;
      if (blinkTimer !== undefined) window.clearTimeout(blinkTimer);
      if (reopenTimer !== undefined) window.clearTimeout(reopenTimer);
    };
  }, [motionEnabled, mood]);

  return (
    <div
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
          onClick={onExpand}
          data-hud-mascot="true"
        >
          <span
            className={hudStyles.guardian}
            data-mood={spriteMood}
            data-motion={motionEnabled ? 'on' : 'off'}
            data-blink={blink ? 'true' : 'false'}
            aria-hidden="true"
          >
            <NexPixelSprite mood={spriteMood} />
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
