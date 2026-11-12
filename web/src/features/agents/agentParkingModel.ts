/**
 * Agent parking model (capability agent-parking, issue 114).
 *
 * The product supports ONE honest parking action today: hiding the agent's
 * terminal surface. Hiding is purely visual — the runtime keeps executing
 * and is NOT paused. Suspending a runtime is NOT supported by the backend
 * (no suspend endpoint exists), so this model never offers a suspend
 * action and the UI copy must never claim a hidden agent is paused.
 */

export type ParkingActionKind = 'hide' | 'suspend';

export interface ParkingAction {
  kind: ParkingActionKind;
  /** Button label key. */
  labelKey: string;
  /** Tooltip/title key. */
  titleKey: string;
}

/** Stable logical key of an agent terminal surface. */
export const agentTerminalSurfaceKey = (agentId: string): string => `agent:${agentId}:terminal`;

/** True when the agent terminal surface is currently open in the workspace. */
export const isAgentSurfaceOpen = (openSurfaceKeys: string[], agentId: string): boolean =>
  openSurfaceKeys.includes(agentTerminalSurfaceKey(agentId));

/**
 * Parking actions available for an agent. `hide` is always available because
 * the workspace can always close a surface. `suspend` appears only when the
 * backend actually supports runtime suspension — which it never does today,
 * so offering it would be a lie.
 */
export const parkingOptions = (suspendSupported: boolean): ParkingAction[] => {
  const actions: ParkingAction[] = [
    {
      kind: 'hide',
      labelKey: 'agents.parking.hide',
      titleKey: 'agents.parking.hideTitle',
    },
  ];
  if (suspendSupported) {
    actions.push({
      kind: 'suspend',
      labelKey: 'agents.parking.suspend',
      titleKey: 'agents.parking.suspendTitle',
    });
  }
  return actions;
};

/** Tooltip key for a parking action; hide explicitly denies the paused claim. */
export const parkingHintKey = (kind: ParkingActionKind): string =>
  kind === 'hide' ? 'agents.parking.hideHint' : 'agents.parking.suspendHint';

/**
 * Status line key shown on the agent card. When the surface is hidden but
 * the runtime keeps working, the copy must say the runtime is STILL
 * RUNNING — never "paused". Returns null when there is nothing to clarify.
 */
export const parkingStatusKey = (surfaceOpen: boolean, runtimeWorking: boolean): string | null => {
  if (surfaceOpen) return null;
  return runtimeWorking ? 'agents.parking.hiddenWorking' : 'agents.parking.hiddenIdle';
};

/** True when surface closed but the runtime keeps running (the parked case). */
export const isParkedWorking = (surfaceOpen: boolean, runtimeWorking: boolean): boolean =>
  !surfaceOpen && runtimeWorking;
