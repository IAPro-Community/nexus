import { useMemo } from 'react';
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

/**
 * Bridges Nexus-owned conversation events into assistant-ui ExternalStoreRuntime.
 * Persistence and provider calls stay outside this hook.
 */
export function useNexusConversationRuntime(options: UseNexusConversationRuntimeOptions) {
  const messages = useMemo(() => {
    const mapped = nexusEventsToThreadMessages(options.events);
    return options.extras?.length ? [...mapped, ...options.extras] : mapped;
  }, [options.events, options.extras]);

  return useExternalStoreRuntime({
    isRunning: options.isRunning,
    isDisabled: options.isDisabled,
    messages,
    convertMessage: (message: ThreadMessageLike) => message,
    onNew: async (message: AppendMessage) => {
      const text = extractText(message);
      await options.onSend({ text, attachments: message.attachments });
    },
    onCancel: options.onCancel
      ? async () => {
          options.onCancel?.();
        }
      : undefined,
  });
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
