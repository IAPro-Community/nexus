import { useCallback, useState } from 'react';
import type { NexusAttachment } from './types';
import { isAllowedImageMime } from './types';

export type UseNexusAttachmentsOptions = {
  sessionId: string;
  maxBytes?: number;
  onInvalid?: (reason: string, file: File) => void;
};

/**
 * Local pending-attachment state for the conversation composer.
 * Upload to Nexus backend happens at submit time via onUpload.
 */
export function useNexusAttachments(options: UseNexusAttachmentsOptions) {
  const [pending, setPending] = useState<NexusAttachment[]>([]);
  const maxBytes = options.maxBytes ?? 8 * 1024 * 1024;

  const addFiles = useCallback(
    (files: File[]) => {
      const next: NexusAttachment[] = [];
      for (const file of files) {
        if (!isAllowedImageMime(file.type)) {
          options.onInvalid?.('unsupported_type', file);
          continue;
        }
        if (file.size > maxBytes) {
          options.onInvalid?.('too_large', file);
          continue;
        }
        const id = `att_local_${crypto.randomUUID()}`;
        next.push({
          id,
          sessionId: options.sessionId,
          name: file.name,
          mimeType: file.type,
          size: file.size,
          type: 'image',
          storageRef: '',
          createdAt: new Date().toISOString(),
          previewUrl: URL.createObjectURL(file),
          metadata: { file },
        });
      }
      if (next.length) setPending((prev) => [...prev, ...next]);
    },
    [maxBytes, options],
  );

  const remove = useCallback((id: string) => {
    setPending((prev) => {
      const target = prev.find((item) => item.id === id);
      if (target?.previewUrl) URL.revokeObjectURL(target.previewUrl);
      return prev.filter((item) => item.id !== id);
    });
  }, []);

  const clear = useCallback(() => {
    setPending((prev) => {
      for (const item of prev) {
        if (item.previewUrl) URL.revokeObjectURL(item.previewUrl);
      }
      return [];
    });
  }, []);

  const takeFiles = useCallback((): File[] => {
    return pending
      .map((item) => item.metadata?.file)
      .filter((file): file is File => file instanceof File);
  }, [pending]);

  return { pending, addFiles, remove, clear, takeFiles };
}
