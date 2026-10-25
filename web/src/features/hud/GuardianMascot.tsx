import React from 'react';
import { GripHorizontal, X } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import type { CompanionAttention, CompanionMood } from './companionModel';
import { NexPixelSprite } from './NexPixelSprite';
import type { GuardianMood } from './guardianModel';
import styles from './GuardianMascot.module.scss';
import hudStyles from './NexusHud.module.scss';

interface GuardianMascotProps {
  mood: CompanionMood;
  count: number;
  expanded: boolean;
  motion: boolean;
  bubble?: CompanionAttention;
  native?: boolean;
  popoverId?: string;
  onExpand: () => void;
  onOpenAttention: (item: CompanionAttention) => void;
  onSnooze: (item: CompanionAttention) => void;
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
  const spriteMood: GuardianMood = mood === 'attention' ? 'warning' : mood;
  const mascotLabel =
    count > 0 && !expanded
      ? t('hud.companion.openPendingCount', { count })
      : t(expanded ? 'hud.companion.collapse' : 'hud.companion.expand');

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
          aria-label={t('hud.companion.attentionLabel')}
          data-companion-bubble
          data-hud-hit-area={native ? 'true' : undefined}
        >
          <button
            className={styles.close}
            type="button"
            aria-label={t('hud.companion.dismissAttention')}
            data-hud-hit-area="true"
            onClick={() => onSnooze(bubble)}
          >
            <X size={13} aria-hidden="true" />
          </button>
          <strong>{t(bubble.titleKey)}</strong>
          <span>{t(bubble.detailKey)}</span>
          <div className={styles.bubbleActions}>
            <button type="button" data-hud-hit-area="true" onClick={() => onOpenAttention(bubble)}>
              {t('hud.companion.open')}
            </button>
            <button type="button" data-hud-hit-area="true" onClick={() => onSnooze(bubble)}>
              {t('hud.companion.snooze')}
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
