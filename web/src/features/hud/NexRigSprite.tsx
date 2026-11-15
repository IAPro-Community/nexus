import React from 'react';
import type { GuardianMood } from './guardianModel';
import type { NexIdleAction } from './nexIdleDirector';
import { NexPixelSprite } from './NexPixelSprite';
import styles from './NexRigSprite.module.scss';

type NexRigSpriteProps = {
  mood: GuardianMood;
  idleAction?: NexIdleAction | null;
};

type RigPlayerInstance = {
  load: () => void;
  destroy: () => void;
};

type RigPlayerConstructor = new (
  canvas: HTMLCanvasElement,
  options?: {
    onReady?: (info: { layerCount: number }) => void;
    onError?: (error: Error) => void;
  },
) => RigPlayerInstance;

type RigPlayerModule = {
  NexRigPlayer?: RigPlayerConstructor;
};

/**
 * NEX HUD companion runtime render (owner-approved 2026-11-15, see
 * docs/design/nex/approval-records/NEX-HUD-RIG-V1-RENDER.md).
 *
 * The locked NEX-ID-v1 atlas stays the first paint and the permanent
 * fallback: this component renders <NexPixelSprite> until the layered
 * rig-v1 player reports ready, and swaps back to the atlas on any load
 * error. The rig animates the same approved hybrid-v1 layers (procedural
 * idle, real module orbit) and honors prefers-reduced-motion internally by
 * showing the byte-faithful source PNG as a static pose. No success or
 * progress is ever inferred: moods remain a color treatment, and offline
 * is still and desaturated.
 */
export const NexRigSprite: React.FC<NexRigSpriteProps> = ({ mood, idleAction = null }) => {
  const canvasRef = React.useRef<HTMLCanvasElement | null>(null);
  const [rigReady, setRigReady] = React.useState(false);

  React.useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || typeof window === 'undefined') return;
    let disposed = false;
    let player: RigPlayerInstance | null = null;

    const boot = async () => {
      try {
        const url = new URL('nex/rig-v1/nex-rig-player.mjs', window.location.origin).href;
        const mod = (await import(/* @vite-ignore */ url)) as RigPlayerModule;
        if (disposed || typeof mod?.NexRigPlayer !== 'function') return;
        player = new mod.NexRigPlayer(canvas, {
          onReady: () => {
            if (!disposed) setRigReady(true);
          },
          onError: () => {
            // Atlas is the permanent fallback: show it again.
            if (disposed) return;
            player?.destroy();
            player = null;
            setRigReady(false);
          },
        });
        player.load();
      } catch {
        // Rig unavailable (test environment, blocked asset): the locked
        // atlas keeps rendering — no empty state, no inferred status.
      }
    };
    void boot();

    return () => {
      disposed = true;
      player?.destroy();
      player = null;
    };
  }, []);

  return (
    <>
      <div
        className={styles.rig}
        data-nex-art={rigReady ? 'rig-v1' : undefined}
        data-nex-mood={mood}
        hidden={!rigReady}
        role="presentation"
      >
        <canvas ref={canvasRef} width={192} height={192} />
      </div>
      {!rigReady && <NexPixelSprite mood={mood} idleAction={idleAction} />}
    </>
  );
};
