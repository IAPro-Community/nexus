import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { GripHorizontal } from 'lucide-react';
import {
  resolveNexBehaviorState,
  type NexBehavior,
  type NexIdleActionId,
  type NexOperationalMood,
} from './motion/nexBehaviorState';
import type { NexClockMetrics } from './motion/nexMotionClock';
import { NexRigCanvas } from './motion/nexRigCanvas';
import type { NexExpression } from './motion/nexEnvelope';
import styles from './NexAvatarView.module.scss';

export interface NexV2LabMetricsSnapshot {
  firstPaintMs: number | null;
  rigReady: boolean;
  atlasFallback: boolean;
  resizeCount: number;
  viewportWidth: number;
  viewportHeight: number;
  behavior: NexBehavior;
  expression: NexExpression;
  mood: NexOperationalMood;
  clock: NexClockMetrics | null;
}

declare global {
  interface Window {
    __nexV2Metrics?: NexV2LabMetricsSnapshot;
  }
}

export interface NexAvatarViewProps {
  mood?: NexOperationalMood;
  motionEnabled?: boolean;
  thinkingActive?: boolean;
  resolvedSignal?: number;
  presetSizePx?: number;
  labMode?: boolean;
  onActivate?: () => void;
}

const MOODS: readonly NexOperationalMood[] = [
  'idle',
  'working',
  'attention',
  'critical',
  'offline',
];
const BEHAVIORS: readonly NexBehavior[] = [
  'ambient',
  'greet',
  'look',
  'thinking',
  'attention',
  'error',
  'resolved',
  'rest',
];

export const NexAvatarView: React.FC<NexAvatarViewProps> = ({
  mood: initialMood = 'idle',
  motionEnabled: initialMotion = true,
  thinkingActive = false,
  resolvedSignal = 0,
  presetSizePx = 160,
  labMode = false,
  onActivate,
}) => {
  const { t } = useTranslation();
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const rigRef = useRef<NexRigCanvas | null>(null);
  const containerRef = useRef<HTMLDivElement | null>(null);
  const bootStartRef = useRef<number>(typeof performance !== 'undefined' ? performance.now() : 0);
  const firstPaintMsRef = useRef<number | null>(null);
  const resizeCountRef = useRef<number>(0);

  const [mood, setMood] = useState<NexOperationalMood>(initialMood);
  const [labBehaviorOverride, setLabBehaviorOverride] = useState<NexBehavior | null>(null);
  const [motionEnabled, setMotionEnabled] = useState<boolean>(initialMotion);

  useEffect(() => {
    if (!labMode) {
      setMood(initialMood);
    }
  }, [initialMood, labMode]);

  useEffect(() => {
    if (!labMode) {
      setMotionEnabled(initialMotion);
    }
  }, [initialMotion, labMode]);
  const [reducedMotion, setReducedMotion] = useState<boolean>(() =>
    typeof window !== 'undefined' && typeof window.matchMedia === 'function'
      ? window.matchMedia('(prefers-reduced-motion: reduce)').matches
      : false,
  );
  const [visible, setVisible] = useState<boolean>(() =>
    typeof document !== 'undefined' ? document.visibilityState === 'visible' : true,
  );
  const [localInteraction, setLocalInteraction] = useState<'none' | 'greet' | 'look'>('none');
  const [idleAction, setIdleAction] = useState<NexIdleActionId | null>(null);
  const [resolvedActive, setResolvedActive] = useState(false);
  const [rigReady, setRigReady] = useState(false);
  const [atlasFallback, setAtlasFallback] = useState(false);

  const resolvedState = useMemo(() => {
    if (labMode && labBehaviorOverride) {
      const motionAllowed = motionEnabled && !reducedMotion && visible && mood !== 'offline';
      const exprMap: Record<NexBehavior, NexExpression> = {
        ambient: 'ambient',
        greet: 'greet',
        look: 'look',
        thinking: 'thinking',
        attention: 'attention',
        error: 'error',
        resolved: 'resolved',
        rest: 'rest',
      };
      return {
        behavior: labBehaviorOverride,
        expression: motionAllowed ? exprMap[labBehaviorOverride] : ('still' as const),
        orbitMode:
          labBehaviorOverride === 'thinking' || mood === 'working'
            ? ('orbit' as const)
            : ('idle' as const),
        animated: motionAllowed,
      };
    }
    return resolveNexBehaviorState({
      mood,
      motionEnabled,
      reducedMotion,
      visible,
      localInteraction,
      idleAction,
      resolvedActive,
      thinkingActive,
    });
  }, [
    idleAction,
    labBehaviorOverride,
    labMode,
    localInteraction,
    mood,
    motionEnabled,
    reducedMotion,
    resolvedActive,
    thinkingActive,
    visible,
  ]);

  const latestStateRef = useRef({ mood, resolvedState });
  latestStateRef.current = { mood, resolvedState };

  useEffect(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') return;
    const media = window.matchMedia('(prefers-reduced-motion: reduce)');
    const onMediaChange = () => setReducedMotion(media.matches);
    const onVisChange = () => setVisible(document.visibilityState === 'visible');
    media.addEventListener('change', onMediaChange);
    document.addEventListener('visibilitychange', onVisChange);
    return () => {
      media.removeEventListener('change', onMediaChange);
      document.removeEventListener('visibilitychange', onVisChange);
    };
  }, []);

  useEffect(() => {
    if (
      labMode ||
      labBehaviorOverride ||
      !motionEnabled ||
      reducedMotion ||
      !visible ||
      mood !== 'idle' ||
      localInteraction !== 'none' ||
      thinkingActive ||
      resolvedActive
    ) {
      return;
    }
    const sequence: readonly NexIdleActionId[] = [
      'idle_glance',
      'idle_head_tilt',
      'idle_fin_twitch',
      'idle_orbit_breathe',
    ];
    let step = 0;
    let holdTimer: number | null = null;
    const interval = window.setInterval(() => {
      const nextAction = sequence[step % sequence.length];
      step += 1;
      setIdleAction(nextAction);
      holdTimer = window.setTimeout(() => {
        setIdleAction((current) => (current === nextAction ? null : current));
      }, 1350);
    }, 4200);
    return () => {
      window.clearInterval(interval);
      if (holdTimer !== null) window.clearTimeout(holdTimer);
    };
  }, [
    labBehaviorOverride,
    labMode,
    localInteraction,
    mood,
    motionEnabled,
    reducedMotion,
    resolvedActive,
    thinkingActive,
    visible,
  ]);

  useEffect(() => {
    if (resolvedSignal <= 0 || !motionEnabled || reducedMotion || !visible || mood === 'offline') {
      return;
    }
    setResolvedActive(true);
    const timer = window.setTimeout(() => setResolvedActive(false), 1250);
    return () => window.clearTimeout(timer);
  }, [mood, motionEnabled, reducedMotion, resolvedSignal, visible]);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    let disposed = false;

    const rig = new NexRigCanvas(canvas, {
      onReady: () => {
        if (disposed) return;
        if (firstPaintMsRef.current === null && typeof performance !== 'undefined') {
          firstPaintMsRef.current = Number((performance.now() - bootStartRef.current).toFixed(2));
        }
        setRigReady(true);
      },
      onError: () => {
        if (!disposed) {
          setRigReady(false);
          setAtlasFallback(true);
        }
      },
      onMetrics: (clock) => {
        if (disposed || typeof window === 'undefined') return;
        const box = containerRef.current?.getBoundingClientRect();
        const latest = latestStateRef.current;
        window.__nexV2Metrics = {
          firstPaintMs: firstPaintMsRef.current,
          rigReady: true,
          atlasFallback: false,
          resizeCount: resizeCountRef.current,
          viewportWidth: box ? Math.round(box.width) : 0,
          viewportHeight: box ? Math.round(box.height) : 0,
          behavior: latest.resolvedState.behavior,
          expression: latest.resolvedState.expression,
          mood: latest.mood,
          clock,
        };
      },
    });
    rigRef.current = rig;
    void rig.load();

    const onWindowResize = () => {
      resizeCountRef.current += 1;
      rig.resize();
    };
    window.addEventListener('resize', onWindowResize);

    const observer =
      typeof ResizeObserver !== 'undefined' && containerRef.current
        ? new ResizeObserver(() => {
            rig.resize();
          })
        : null;
    if (observer && containerRef.current) {
      observer.observe(containerRef.current);
    }

    return () => {
      disposed = true;
      window.removeEventListener('resize', onWindowResize);
      observer?.disconnect();
      rig.destroy();
      rigRef.current = null;
    };
    // Boot rig once per mount; behavior updates stream via updatePresentation below.
  }, []);

  useEffect(() => {
    rigRef.current?.resize();
  }, [presetSizePx, rigReady]);

  useEffect(() => {
    const rig = rigRef.current;
    if (rig) {
      rig.updatePresentation({
        expression: resolvedState.expression,
        orbitMode: resolvedState.orbitMode,
        animated: resolvedState.animated,
      });
    }
    if (typeof window !== 'undefined') {
      const box = containerRef.current?.getBoundingClientRect();
      window.__nexV2Metrics = {
        firstPaintMs: firstPaintMsRef.current,
        rigReady,
        atlasFallback,
        resizeCount: resizeCountRef.current,
        viewportWidth: box ? Math.round(box.width) : 0,
        viewportHeight: box ? Math.round(box.height) : 0,
        behavior: resolvedState.behavior,
        expression: resolvedState.expression,
        mood,
        clock: rig?.getMetrics() ?? null,
      };
    }
  }, [atlasFallback, mood, resolvedState, rigReady]);

  const applyLabBehavior = (behavior: NexBehavior) => {
    setLabBehaviorOverride(behavior);
    if (behavior === 'thinking') setMood('working');
    else if (behavior === 'attention') setMood('attention');
    else if (behavior === 'error') setMood('critical');
    else if (behavior === 'rest') setIdleAction('idle_drowse');
    else {
      setIdleAction(null);
      setMood('idle');
    }
  };

  return (
    <div
      ref={containerRef}
      className={labMode ? `${styles.root} ${styles.rootLab}` : styles.root}
      style={
        labMode
          ? undefined
          : ({ '--nx-nex-avatar-size': `${presetSizePx}px` } as React.CSSProperties)
      }
      data-lab-mode={labMode ? 'true' : 'false'}
      data-mood={mood}
      data-nex-behavior={resolvedState.behavior}
      data-nex-expression={resolvedState.expression}
      data-nex-motion={resolvedState.animated ? 'on' : 'off'}
      data-nex-ready={rigReady ? 'rig-v1' : atlasFallback ? 'atlas' : 'loading'}
    >
      <div className={styles.stage}>
        <button
          type="button"
          className={styles.dragHandle}
          data-wails-draggable="true"
          title={t('nexV2.dragTitle')}
          aria-label={t('nexV2.dragLabel')}
        >
          <GripHorizontal size={12} aria-hidden="true" />
        </button>
        <button
          type="button"
          className={styles.avatarButton}
          aria-label={`${t('nexV2.avatarLabel')} · ${t('nexV2.avatarMood', { mood })}`}
          onClick={() => {
            setLabBehaviorOverride(null);
            setLocalInteraction('greet');
            onActivate?.();
          }}
          onPointerEnter={() => setLocalInteraction('greet')}
          onPointerMove={(event) => {
            const rect = event.currentTarget.getBoundingClientRect();
            const nx = ((event.clientX - rect.left) / Math.max(1, rect.width) - 0.5) * 2;
            const ny = ((event.clientY - rect.top) / Math.max(1, rect.height) - 0.5) * 2;
            rigRef.current?.setPointerNormalized(nx, ny);
            setLocalInteraction('look');
          }}
          onPointerLeave={() => {
            rigRef.current?.setPointerNormalized(0, 0);
            setLocalInteraction('none');
          }}
          onFocus={() => setLocalInteraction('greet')}
          onBlur={() => setLocalInteraction('none')}
        >
          <canvas
            ref={canvasRef}
            className={rigReady ? styles.canvas : `${styles.canvas} ${styles.canvasHidden}`}
          />
          {!rigReady && (
            <span className={styles.atlasFallback} role="img" aria-label={t('nexV2.atlasAlt')} />
          )}
        </button>
      </div>

      {labMode && (
        <section className={styles.labPanel} aria-label={t('nexV2.labTitle')}>
          <div className={styles.labRow}>
            <span>{t('nexV2.moodLabel')}</span>
            <div className={styles.labChips}>
              {MOODS.map((m) => (
                <button
                  key={m}
                  type="button"
                  data-lab-mood={m}
                  aria-pressed={mood === m}
                  onClick={() => {
                    setLabBehaviorOverride(null);
                    setResolvedActive(false);
                    setIdleAction(null);
                    setMood(m);
                  }}
                >
                  {m}
                </button>
              ))}
            </div>
          </div>
          <div className={styles.labRow}>
            <span>{t('nexV2.behaviorLabel')}</span>
            <div className={styles.labChips}>
              {BEHAVIORS.map((b) => (
                <button
                  key={b}
                  type="button"
                  data-lab-behavior={b}
                  aria-pressed={resolvedState.behavior === b}
                  onClick={() => applyLabBehavior(b)}
                >
                  {b}
                </button>
              ))}
            </div>
          </div>
          <div className={styles.labRow}>
            <button
              type="button"
              data-lab-reduced-motion
              aria-pressed={!motionEnabled || reducedMotion}
              onClick={() => setMotionEnabled((prev) => !prev)}
            >
              {t('nexV2.reducedMotionLabel')}
            </button>
          </div>
        </section>
      )}
    </div>
  );
};
