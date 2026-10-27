/**
 * Desktop command bridge.
 *
 * The HUD runs in its own window (and may even run as a separate process), so
 * it cannot open a Desktop dialog directly. It asks the Core, which delivers
 * the command in-process to the Desktop window; the Desktop emits a Wails
 * event and the web app reacts here.
 */
export const DESKTOP_COMMAND_EVENT = 'nexus:desktop-command';

export type DesktopCommandAction = 'new-agent';

interface WailsEventLike {
  data?: { action?: string } | null;
}

/**
 * Subscribes to desktop commands emitted by the native shell. Returns a no-op
 * unsubscribe outside the desktop runtime, and never throws when the Wails
 * runtime is unavailable (for example in a plain browser build).
 */
export function subscribeDesktopCommand(
  handler: (action: DesktopCommandAction) => void,
): () => void {
  let disposed = false;
  let unsubscribe: (() => void) | undefined;

  void import('@wailsio/runtime')
    .then(({ Events }) => {
      if (disposed) return;
      unsubscribe = Events.On(DESKTOP_COMMAND_EVENT, (event: WailsEventLike) => {
        const action = event?.data?.action;
        if (action === 'new-agent') handler('new-agent');
      });
    })
    .catch(() => undefined);

  return () => {
    disposed = true;
    unsubscribe?.();
  };
}
