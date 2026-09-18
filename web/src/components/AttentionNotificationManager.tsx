import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { toast, Toaster } from 'sonner';
import type { RuntimeSession } from '../types';
import { attentionMessageKey } from '../app/documentTitle';
import { AttentionNotificationCard, shouldRenderAttentionCard } from './AttentionNotificationCard';

export interface AttentionNotificationManagerProps {
  runtimes: RuntimeSession[];
  focusedProjectId?: string;
  onFocusRuntime: (runtimeId: string) => void;
}

export const AttentionNotificationManager: React.FC<AttentionNotificationManagerProps> = ({
  runtimes,
  focusedProjectId,
  onFocusRuntime,
}) => {
  const [dismissedFingerprints, setDismissedFingerprints] = useState<string[]>([]);
  const activeToastIds = useRef<Map<string, string | number>>(new Map());

  // Toast only for honest needs_user in the focused project (radar stays global).
  const attentionRuntimes = useMemo(
    () =>
      runtimes.filter((runtime) => {
        if (!shouldRenderAttentionCard(runtime)) return false;
        if ((runtime.provider_id || runtime.provider || '').toLowerCase() === 'shell') return false;
        if (!focusedProjectId || !runtime.project_id || runtime.project_id !== focusedProjectId) {
          return false;
        }
        const fingerprint = attentionMessageKey(runtime);
        if (dismissedFingerprints.includes(fingerprint)) return false;
        return true;
      }),
    [runtimes, focusedProjectId, dismissedFingerprints],
  );

  // One toast per message content (not per runtime).
  const toastRuntimes = useMemo(() => {
    const uniqueByFingerprint = new Map<string, RuntimeSession>();
    for (const runtime of attentionRuntimes) {
      const fingerprint = attentionMessageKey(runtime);
      if (!uniqueByFingerprint.has(fingerprint)) uniqueByFingerprint.set(fingerprint, runtime);
    }
    return [...uniqueByFingerprint.values()];
  }, [attentionRuntimes]);

  const handleDismiss = useCallback(
    (runtimeId: string) => {
      const runtime = runtimes.find((item) => item.runtime_id === runtimeId);
      const fingerprint = runtime ? attentionMessageKey(runtime) : runtimeId;
      setDismissedFingerprints((prev) => [...prev, fingerprint]);
      const toastId = activeToastIds.current.get(fingerprint);
      if (toastId) {
        toast.dismiss(toastId);
        activeToastIds.current.delete(fingerprint);
      }
    },
    [runtimes],
  );

  useEffect(() => {
    const currentFingerprints = new Set(
      toastRuntimes.map((runtime) => attentionMessageKey(runtime)),
    );

    activeToastIds.current.forEach((toastId, fingerprint) => {
      if (!currentFingerprints.has(fingerprint)) {
        toast.dismiss(toastId);
        activeToastIds.current.delete(fingerprint);
      }
    });

    toastRuntimes.forEach((runtime) => {
      const fingerprint = attentionMessageKey(runtime);
      if (activeToastIds.current.has(fingerprint)) return;
      const id = toast.custom(
        () => (
          <AttentionNotificationCard
            runtime={runtime}
            onFocusRuntime={onFocusRuntime}
            onDismiss={handleDismiss}
          />
        ),
        {
          id: `attention-${fingerprint}`,
          duration: Infinity,
          dismissible: false,
        },
      );
      activeToastIds.current.set(fingerprint, id);
    });
  }, [toastRuntimes, onFocusRuntime, handleDismiss]);

  return createPortal(
    <div className="nx-attention-toaster-root">
      <Toaster
        position="top-right"
        expand
        richColors={false}
        gap={12}
        visibleToasts={4}
        offset={{ top: 52, right: 16 }}
        toastOptions={{
          className: 'nx-sonner-toast-wrapper',
          unstyled: true,
        }}
      />
    </div>,
    document.body,
  );
};
