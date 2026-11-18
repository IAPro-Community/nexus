import React from 'react';
import type { GuardianMood } from './guardianModel';
import type { NexIdleAction } from './nexIdleDirector';
import { NexPixelSprite } from './NexPixelSprite';
import type { NexCompanionBehavior } from './nexCompanionBehavior';
import { resolveNexRigPresentation } from './nexRigProjection';
import styles from './NexRigSprite.module.scss';

type NexRigSpriteProps = {
  mood: GuardianMood;
  idleAction?: NexIdleAction | null;
  /** Motion authority from the mascot (settings, offline, visibility, reduced motion). */
  motionEnabled?: boolean;
  /** Companion behavior from the mascot; drives the rig expression ('ambient' when absent). */
  behavior?: NexCompanionBehavior | null;
};

type RigPlayerInstance = {
  load: () => void;
  destroy: () => void;
  setMotion: (allowed: boolean) => void;
  setMode: (mode: 'idle' | 'orbit') => void;
  setExpression: (name: string) => void;
  getExpression: () => string;
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
 * rendering one frozen representative rig pose. Expressions are
 * owner-approved procedural choreography over those same approved layers;
 * 'still' is not an expression — it is handled by setMotion(false). No
 * success or progress is ever inferred: moods remain a color treatment, and
 * offline is still and desaturated.
 */
export const NexRigSprite: React.FC<NexRigSpriteProps> = ({
  mood,
  idleAction = null,
  motionEnabled = true,
  behavior = null,
}) => {
  const canvasRef = React.useRef<HTMLCanvasElement | null>(null);
  const playerRef = React.useRef<RigPlayerInstance | null>(null);
  const [rigReady, setRigReady] = React.useState(false);
  const presentation = resolveNexRigPresentation(behavior ?? 'ambient', idleAction, motionEnabled);

  React.useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas || typeof window === 'undefined') return;
    let disposed = false;

    const boot = async () => {
      try {
        const url = new URL('nex/rig-v1/nex-rig-player.mjs', window.location.origin).href;
        const mod = (await import(/* @vite-ignore */ url)) as RigPlayerModule;
        if (disposed || typeof mod?.NexRigPlayer !== 'function') return;
        const player = new mod.NexRigPlayer(canvas, {
          onReady: () => {
            if (!disposed) setRigReady(true);
          },
          onError: () => {
            // Atlas is the permanent fallback: show it again.
            if (disposed) return;
            player.destroy();
            playerRef.current = null;
            setRigReady(false);
          },
        });
        playerRef.current = player;
        player.setMotion(motionEnabled);
        player.load();
      } catch {
        // Rig unavailable (test environment, blocked asset): the locked
        // atlas keeps rendering — no empty state, no inferred status.
      }
    };
    void boot();

    return () => {
      disposed = true;
      playerRef.current?.destroy();
      playerRef.current = null;
    };
    // Boot once; motion changes flow through the effect below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  React.useEffect(() => {
    playerRef.current?.setMotion(motionEnabled);
  }, [motionEnabled, rigReady]);

  // Re-runs when rigReady flips so the freshly booted player receives the
  // current expression (the player sets canvas.dataset.nexExpression).
  React.useEffect(() => {
    const player = playerRef.current;
    if (!player) return;
    player.setMode(presentation.mode);
    player.setExpression(presentation.expression);
  }, [presentation.mode, presentation.expression, rigReady]);

  return (
    <>
      <div
        className={styles.rig}
        data-nex-art={rigReady ? 'rig-v1' : undefined}
        data-nex-mood={mood}
        data-nex-idle-action={idleAction ?? undefined}
        data-nex-rig-mode={presentation.mode}
        hidden={!rigReady}
        role="presentation"
      >
        <canvas ref={canvasRef} width={192} height={192} />
      </div>
      {!rigReady && (
        <NexPixelSprite mood={mood} idleAction={idleAction} motionEnabled={motionEnabled} />
      )}
    </>
  );
};
