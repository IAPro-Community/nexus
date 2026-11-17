import type { NexIdleAction } from './nexIdleDirector';
import type { NexCompanionBehavior } from './nexCompanionBehavior';

export type NexRigMode = 'idle' | 'orbit';
export type NexRigPresentation = { mode: NexRigMode; expression: NexCompanionBehavior };

/**
 * A presentation-only projection. Idle gestures must never preempt operational
 * warnings, a verified completion, or the explicit motion/offline freeze.
 *
 * The procedural rig cannot yet independently articulate the helmet or fins.
 * Only map gestures with a faithful representation in the approved layered art;
 * unknown gestures deliberately retain ambient instead of faking anatomy.
 */
export function resolveNexRigPresentation(
  behavior: NexCompanionBehavior,
  action: NexIdleAction | null,
  motionEnabled: boolean,
): NexRigPresentation {
  if (!motionEnabled || behavior === 'still') return { mode: 'idle', expression: 'still' };
  if (behavior !== 'ambient' && behavior !== 'rest') return { mode: 'idle', expression: behavior };
  if (behavior === 'rest' && action !== 'idle_wake') return { mode: 'idle', expression: 'rest' };

  switch (action) {
    case 'idle_orbit_shuffle':
    case 'idle_orbit_breathe':
      return { mode: 'orbit', expression: 'ambient' };
    case 'idle_glance':
    case 'idle_curiosity':
    case 'idle_module_inspect':
      return { mode: 'idle', expression: 'look' };
    case 'idle_head_tilt':
    case 'idle_micro_stretch':
    case 'idle_wake':
      return { mode: 'idle', expression: 'greet' };
    case 'idle_drowse':
      return { mode: 'idle', expression: 'rest' };
    default:
      return { mode: 'idle', expression: 'ambient' };
  }
}
