import { useCallback, useEffect, useRef, useState } from 'react';

/** Director only observes the Nexus-owned SessionHost; it never starts or steers workers. */
export const DIRECTOR_POLL_INTERVAL_MS = 15_000;

export function shouldPollDirector(
  visibility: DocumentVisibilityState,
  pending: boolean,
  lastAttemptMs: number,
  nowMs: number,
): boolean {
  return (
    visibility === 'visible' &&
    !pending &&
    nowMs - lastAttemptMs >= DIRECTOR_POLL_INTERVAL_MS
  );
}

/**
 * Bounded refresh for the active Director dialog. No polling while the document
 * is hidden or a request is pending; returning to a visible tab checks once.
 * The dialog's unmount removes both timer and visibility listener.
 */
export function useDirectorPolling(pending: boolean) {
  const [revision, setRevision] = useState(0);
  const pendingRef = useRef(pending);
  // Avoid non-deterministic reads during React render; the first GET is already in flight.
  const lastAttemptRef = useRef(0);

  useEffect(() => {
    const wasPending = pendingRef.current;
    pendingRef.current = pending;
    // Rate-limit from completion too: a slow response must not cause
    // another immediate request as soon as it finally finishes.
    if (wasPending && !pending) lastAttemptRef.current = Date.now();
  }, [pending]);

  const refresh = useCallback(() => {
    if (pendingRef.current) return;
    lastAttemptRef.current = Date.now();
    setRevision((current) => current + 1);
  }, []);

  useEffect(() => {
    const check = () => {
      if (
        shouldPollDirector(
          document.visibilityState,
          pendingRef.current,
          lastAttemptRef.current,
          Date.now(),
        )
      ) {
        refresh();
      }
    };
    const timer = window.setInterval(check, DIRECTOR_POLL_INTERVAL_MS);
    document.addEventListener('visibilitychange', check);
    return () => {
      window.clearInterval(timer);
      document.removeEventListener('visibilitychange', check);
    };
  }, [refresh]);

  return { revision, refresh };
}
