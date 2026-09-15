import React, { useCallback, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  ConversationSurface,
  useNexusAttachments,
  useNexusConversationRuntime,
  type NexusConversationEvent,
} from '../features/conversation';
import { Button, Card } from '../design-system';
import styles from './ConversationSpikePage.module.scss';

/**
 * Temporary integrated spike for assistant-ui + Nexus conversation foundation.
 * Route: /dev/conversation-spike
 */
export const ConversationSpikePage: React.FC = () => {
  const { t } = useTranslation();
  const seq = useRef(1);
  const [events, setEvents] = useState<NexusConversationEvent[]>([]);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState('');
  const attachments = useNexusAttachments({
    sessionId: 'spike',
    onInvalid: () =>
      setError(t('conversation.invalidAttachment', 'Unsupported or too large file.')),
  });

  const nextId = () => `spike_${seq.current++}`;

  const append = useCallback(
    (
      partial: Omit<NexusConversationEvent, 'sessionId' | 'sequence' | 'createdAt'> & {
        sequence?: number;
      },
    ) => {
      setEvents((prev) => [
        ...prev,
        {
          sessionId: 'spike',
          sequence: partial.sequence ?? prev.length + 1,
          createdAt: new Date().toISOString(),
          ...partial,
        },
      ]);
    },
    [],
  );

  const onSend = useCallback(
    async ({ text }: { text: string }) => {
      setError('');
      const imagePayload = attachments.pending.map((att) => ({
        id: att.id,
        name: att.name,
        mimeType: att.mimeType,
        size: att.size,
        url: att.previewUrl,
      }));
      append({
        id: nextId(),
        kind: 'user',
        content: text,
        payload: imagePayload.length ? { attachments: imagePayload } : undefined,
      });
      attachments.clear();
      setRunning(true);

      const activityId = nextId();
      append({
        id: activityId,
        kind: 'activity',
        status: 'running',
        payload: {
          title: t('conversation.spike.working', 'Working…'),
          currentId: 'analyze',
          items: [
            {
              id: 'analyze',
              label: t('conversation.spike.analyze', 'Analyzing context'),
              status: 'running',
              startedAt: new Date().toISOString(),
            },
            {
              id: 'tool',
              label: t('conversation.spike.tool', 'Running fake tool'),
              status: 'pending',
            },
            {
              id: 'done',
              label: t('conversation.spike.finish', 'Completing'),
              status: 'pending',
            },
          ],
          expandable: true,
        },
      });

      const chunks = [
        t('conversation.spike.stream1', 'Here is a streamed reply.'),
        ' ',
        t('conversation.spike.stream2', 'Activity and tools are rendered as structured parts.'),
      ];
      let body = '';
      const assistantId = nextId();
      for (const chunk of chunks) {
        body += chunk;
        await Promise.resolve();
        setEvents((prev) => {
          const without = prev.filter((e) => e.id !== assistantId);
          return [
            ...without,
            {
              id: assistantId,
              sessionId: 'spike',
              sequence: without.length + 1,
              kind: 'assistant' as const,
              content: body,
              createdAt: new Date().toISOString(),
              status: 'running' as const,
            },
          ];
        });
      }

      append({
        id: nextId(),
        kind: 'tool_call',
        payload: {
          toolName: 'fake.shell',
          status: 'success',
          summary: 'echo ok',
          detail: 'stdout:\nok\n',
        },
      });

      setEvents((prev) =>
        prev.map((e) =>
          e.id === activityId
            ? {
                ...e,
                status: 'complete' as const,
                payload: {
                  title: t('conversation.spike.working', 'Working…'),
                  currentId: 'done',
                  items: [
                    {
                      id: 'analyze',
                      label: t('conversation.spike.analyze', 'Analyzing context'),
                      status: 'done',
                    },
                    {
                      id: 'tool',
                      label: t('conversation.spike.tool', 'Running fake tool'),
                      status: 'done',
                    },
                    {
                      id: 'done',
                      label: t('conversation.spike.finish', 'Completing'),
                      status: 'done',
                    },
                  ],
                },
              }
            : e.id === assistantId
              ? { ...e, status: 'complete' as const }
              : e,
        ),
      );

      if (text.toLowerCase().includes('error')) {
        append({
          id: nextId(),
          kind: 'error',
          content: t('conversation.spike.errorDemo', 'Simulated failure for the spike.'),
          payload: { detail: 'spike_error' },
        });
        setError(t('conversation.spike.errorDemo', 'Simulated failure for the spike.'));
      } else {
        append({
          id: nextId(),
          kind: 'completion',
          content: t('conversation.spike.complete', 'Done.'),
        });
      }
      setRunning(false);
    },
    [append, attachments, t],
  );

  const runtime = useNexusConversationRuntime({
    events,
    isRunning: running,
    onSend,
    onCancel: () => setRunning(false),
  });

  const inspector = useMemo(
    () => (
      <Card>
        <strong>{t('conversation.spike.inspector', 'Spike inspector')}</strong>
        <p>{t('conversation.spike.inspectorHint', 'Domain details stay in the side panel.')}</p>
        <Button size="sm" onClick={() => setEvents([])}>
          {t('conversation.spike.reset', 'Reset')}
        </Button>
      </Card>
    ),
    [t],
  );

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <strong>{t('conversation.spike.title', 'Conversation foundation spike')}</strong>
      </header>
      <div className={styles.body}>
        <ConversationSurface
          runtime={runtime}
          inspector={inspector}
          pendingAttachments={attachments.pending}
          onAddFiles={attachments.addFiles}
          onRemoveAttachment={attachments.remove}
          error={error}
          emptyLabel={t(
            'conversation.spike.empty',
            'Send a message. Include “error” to demo failure. Attach PNG/JPEG/WEBP.',
          )}
        />
      </div>
    </div>
  );
};
