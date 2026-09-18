import { useCallback, useMemo, useRef } from 'react';
import {
  useExternalStoreRuntime,
  type AppendMessage,
  type ThreadMessageLike,
} from '@assistant-ui/react';
import type { NexusConversationEvent } from './types';
import { nexusEventsToThreadMessages } from './nexusConversationMapper';

export type NexusConversationSendInput = {
  text: string;
  attachments?: AppendMessage['attachments'];
};

export type UseNexusConversationRuntimeOptions = {
  events: readonly NexusConversationEvent[];
  isRunning?: boolean;
  isDisabled?: boolean;
  onSend: (input: NexusConversationSendInput) => void | Promise<void>;
  onCancel?: () => void;
  /** Extra messages already in ThreadMessageLike form (e.g. ephemeral activity). */
  extras?: readonly ThreadMessageLike[];
};

const identityConvert = (message: ThreadMessageLike) => message;

/**
 * Bridges Nexus-owned conversation events into assistant-ui ExternalStoreRuntime.
 * Persistence and provider calls stay outside this hook.
 *
 * Adapter callbacks must stay referentially stable: a new `convertMessage` on
 * every render flushes the ExternalStore converter cache, remounts the thread,
 * and (with Viewport MutationObserver auto-scroll) can freeze the main thread.
 */
export function useNexusConversationRuntime(options: UseNexusConversationRuntimeOptions) {
  const messages = useMemo(() => {
    const mapped = nexusEventsToThreadMessages(options.events);
    return options.extras?.length ? [...mapped, ...options.extras] : mapped;
  }, [options.events, options.extras]);

  const onSendRef = useRef(options.onSend);
  onSendRef.current = options.onSend;
  const onCancelRef = useRef(options.onCancel);
  onCancelRef.current = options.onCancel;

  const onNew = useCallback(async (message: AppendMessage) => {
    const text = extractText(message);
    await onSendRef.current({ text, attachments: message.attachments });
  }, []);

  const onCancel = useCallback(async () => {
    onCancelRef.current?.();
  }, []);

  const hasCancel = Boolean(options.onCancel);

  const adapter = useMemo(
    () => ({
      isRunning: options.isRunning,
      isDisabled: options.isDisabled,
      messages,
      convertMessage: identityConvert,
      onNew,
      onCancel: hasCancel ? onCancel : undefined,
    }),
    [options.isRunning, options.isDisabled, messages, onNew, onCancel, hasCancel],
  );

  return useExternalStoreRuntime(adapter);
}

function extractText(message: AppendMessage): string {
  if (typeof message.content === 'string') return message.content;
  if (!Array.isArray(message.content)) return '';
  return message.content
    .map((part) => {
      if (part && typeof part === 'object' && 'type' in part && part.type === 'text') {
        return String((part as { text?: string }).text || '');
      }
      return '';
    })
    .join('')
    .trim();
}
