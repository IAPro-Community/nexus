import React from 'react';
import { GripHorizontal, X } from 'lucide-react';
import type { CompanionAttention, CompanionMood } from './companionModel';
import styles from './NexusGuardian.module.scss';

interface NexusGuardianProps {
  mood: CompanionMood;
  count: number;
  expanded: boolean;
  motion: boolean;
  bubble?: CompanionAttention;
  native?: boolean;
  onExpand: () => void;
  onOpenPending: () => void;
  onOpenAttention: (item: CompanionAttention) => void;
  onSnooze: (item: CompanionAttention) => void;
  onMoveKeys?: (event: React.KeyboardEvent<HTMLButtonElement>) => void;
}

export const NexusGuardian: React.FC<NexusGuardianProps> = ({
  mood, count, expanded, motion, bubble, native,
  onExpand, onOpenPending, onOpenAttention, onSnooze, onMoveKeys,
}) => (
  <div className={styles.root} data-mood={mood} data-motion={motion ? 'on' : 'off'}>
    {bubble && !expanded && (
      <div className={styles.speech} role="status" data-companion-bubble>
        <button className={styles.close} type="button" aria-label="Dispensar aviso" onClick={() => onSnooze(bubble)}>
          <X size={13} />
        </button>
        <strong>{bubble.title}</strong>
        <div className={styles.bubbleActions}>
          <button type="button" onClick={() => onOpenAttention(bubble)}>Abrir</button>
          <button type="button" onClick={() => onSnooze(bubble)}>Depois</button>
        </div>
      </div>
    )}
    <div className={styles.group}>
      {native && (
        <button
          type="button"
          className={styles.dragHandle}
          data-hud-drag-handle="true"
          onKeyDown={onMoveKeys}
          title="Arraste para mover. Setas movem com o teclado."
          aria-label="Mover Nexus Companion"
        >
          <GripHorizontal size={13} />
        </button>
      )}
      <button
        type="button"
        className={styles.guardianButton}
        aria-label={expanded ? 'Recolher painel do Nexus' : 'Expandir painel do Nexus'}
        aria-expanded={expanded}
        aria-controls={expanded ? 'nexus-hud-popover' : undefined}
        onClick={onExpand}
        data-companion-guardian
      >
        <span className={styles.sprite} aria-hidden="true" />
      </button>
      <button
        type="button"
        className={styles.bucketButton}
        onClick={onOpenPending}
        aria-label={count > 0 ? `Abrir ${count} pendências no Nexus` : 'Ver pendências do Nexus'}
        data-companion-bucket
      >
        <img src="/companion/nexus-task-capsule.svg" width="49" height="54" alt="" draggable={false} />
        {count > 0 && <span className={styles.badge} aria-hidden="true">{count > 9 ? '9+' : count}</span>}
      </button>
    </div>
  </div>
);
