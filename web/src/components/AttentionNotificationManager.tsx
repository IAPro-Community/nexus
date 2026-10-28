import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import { toast, Toaster } from 'sonner';
import type { RuntimeSession } from '../types';
import { attentionToastKey } from './attentionToastModel';
import { AttentionNotificationCard, shouldRenderAttentionCard } from './AttentionNotificationCard';

export interface AttentionNotificationManagerProps {
  runtimes: RuntimeSession[];
  focusedProjectId?: string;
  onFocusRuntime: (runtimeId: string) => void;
}

export const AttentionNotificationManager: React.FC<AttentionNotificationManagerProps> = ({
  runtimes,
  onFocusRuntime,
}) => {
  const [dismissedFingerprints, setDismissedFingerprints] = useState<string[]>([]);
  const activeToastIds = useRef<Map<string, { id: string | number; key: string }>>(new Map());
  const toastSequence = useRef(0);

  // Universal attention delivery: all honest needs_user waits across all projects.
  // Delivery operates independently of the currently active tab or focused project.
  const attentionRuntimes = useMemo(
    () =>
      runtimes.filter((runtime) => {
        if (!shouldRenderAttentionCard(runtime)) return false;
        if ((runtime.provider_id || runtime.provider || '').toLowerCase() === 'shell') return false;
        const key = attentionToastKey(runtime);
        if (dismissedFingerprints.includes(key)) return false;
        return true;
      }),
    [runtimes, dismissedFingerprints],
  );

  // Each runtime requiring intervention gets its own distinct, correlated card.
  // Two agents asking the same question remain separate and independently actionable.
  const toastRuntimes = useMemo(() => {
    const uniqueByRuntime = new Map<string, RuntimeSession>();
    for (const runtime of attentionRuntimes) {
      if (!uniqueByRuntime.has(runtime.runtime_id)) {
        uniqueByRuntime.set(runtime.runtime_id, runtime);
      }
    }
    return [...uniqueByRuntime.values()];
  }, [attentionRuntimes]);

  const handleDismiss = useCallback(
    (runtimeId: string) => {
      const runtime = runtimes.find((item) => item.runtime_id === runtimeId);
      const key = runtime ? attentionToastKey(runtime) : runtimeId;
      setDismissedFingerprints((prev) => (prev.includes(key) ? prev : [...prev, key]));
      const active = activeToastIds.current.get(runtimeId);
      if (active) {
        toast.dismiss(active.id);
        activeToastIds.current.delete(runtimeId);
      }
    },
    [runtimes],
  );

  useEffect(() => {
    const currentRuntimeIds = new Set(toastRuntimes.map((runtime) => runtime.runtime_id));

    activeToastIds.current.forEach((entry, runtimeId) => {
      if (!currentRuntimeIds.has(runtimeId)) {
        toast.dismiss(entry.id);
        activeToastIds.current.delete(runtimeId);
      }
    });

    toastRuntimes.forEach((runtime) => {
      const key = attentionToastKey(runtime);
      const previous = activeToastIds.current.get(runtime.runtime_id);
      if (previous?.key === key) return;
      // A later permission/question for the same runtime must replace the
      // existing card, not leave a stale closure pointing at old options.
      if (previous) toast.dismiss(previous.id);
      const toastId = `attention-${runtime.runtime_id}-${++toastSequence.current}`;
      const id = toast.custom(
        () => (
          <AttentionNotificationCard
            runtime={runtime}
            onFocusRuntime={onFocusRuntime}
            onDismiss={handleDismiss}
          />
        ),
        {
          id: toastId,
          duration: Infinity,
          dismissible: false,
        },
      );
      activeToastIds.current.set(runtime.runtime_id, { id, key });
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
