import React, { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  AssistantRuntimeProvider,
  ComposerPrimitive,
  MessagePrimitive,
  ThreadPrimitive,
  type AssistantRuntime,
} from '@assistant-ui/react';
import { Paperclip, Send, Square, X } from 'lucide-react';
import { Button, ContextDrawer, IconButton, Spinner } from '../../design-system';
import { AgentActivity } from './AgentActivity';
import { ToolCard } from './ToolCard';
import { asActivityPayload, asToolPayload } from './nexusConversationMapper';
import type { NexusAttachment } from './types';
import { isAllowedImageMime } from './types';
import styles from './ConversationSurface.module.scss';

export type ConversationInspectorVariant = 'split' | 'drawer' | 'responsive';

export type ConversationSurfaceProps = {
  runtime: AssistantRuntime;
  emptyLabel?: string;
  inspector?: React.ReactNode;
  inspectorOpen?: boolean;
  onToggleInspector?: () => void;
  hideInspectorChrome?: boolean;
  inspectorVariant?: ConversationInspectorVariant;
  inspectorTitle?: string;
  pendingAttachments?: NexusAttachment[];
  onAddFiles?: (files: File[]) => void;
  onRemoveAttachment?: (id: string) => void;
  header?: React.ReactNode;
  footer?: React.ReactNode;
  threadEnd?: React.ReactNode;
  composerLeading?: React.ReactNode;
  composerTrailing?: React.ReactNode;
  error?: string;
  loading?: boolean;
};

function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(() =>
    typeof window === 'undefined' ? false : window.matchMedia(query).matches,
  );
  useEffect(() => {
    const media = window.matchMedia(query);
    const onChange = () => setMatches(media.matches);
    onChange();
    media.addEventListener('change', onChange);
    return () => media.removeEventListener('change', onChange);
  }, [query]);
  return matches;
}

/**
 * Central conversational layout. Domain-agnostic — Composer/Work inject inspector slots.
 */
export const ConversationSurface: React.FC<ConversationSurfaceProps> = ({
  runtime,
  emptyLabel,
  inspector,
  inspectorOpen = true,
  onToggleInspector,
  hideInspectorChrome = false,
  inspectorVariant = 'responsive',
  inspectorTitle,
  pendingAttachments = [],
  onAddFiles,
  onRemoveAttachment,
  header,
  footer,
  threadEnd,
  composerLeading,
  composerTrailing,
  error,
  loading,
}) => {
  const { t } = useTranslation();
  const fileRef = useRef<HTMLInputElement>(null);
  const [dragging, setDragging] = useState(false);
  const compactViewport = useMediaQuery('(max-width: 767px)');
  const useDrawer =
    Boolean(inspector) &&
    (inspectorVariant === 'drawer' || (inspectorVariant === 'responsive' && compactViewport));

  const acceptFiles = useCallback(
    (list: FileList | File[] | null) => {
      if (!onAddFiles || !list) return;
      const files = Array.from(list).filter((f) => isAllowedImageMime(f.type));
      if (files.length) onAddFiles(files);
    },
    [onAddFiles],
  );

  const inspectorLabel = inspectorTitle || t('conversation.inspector', 'Details');

  return (
    <AssistantRuntimeProvider runtime={runtime}>
      <div
        className={styles.shell}
        data-inspector={!inspector ? 'none' : inspectorOpen && !useDrawer ? 'open' : 'closed'}
        onDragOver={(e) => {
          if (!onAddFiles) return;
          e.preventDefault();
          setDragging(true);
        }}
        onDragLeave={() => setDragging(false)}
        onDrop={(e) => {
          if (!onAddFiles) return;
          e.preventDefault();
          setDragging(false);
          acceptFiles(e.dataTransfer.files);
        }}
      >
        <div className={styles.main}>
          {header}
          {loading ? (
            <div className={styles.loading}>
              <Spinner label={t('common.loading', 'Loading…')} />
            </div>
          ) : null}
          <ThreadPrimitive.Root className={styles.threadRoot}>
            <ThreadPrimitive.Viewport className={styles.viewport} autoScroll>
              <ThreadPrimitive.Empty>
                <p className={styles.empty}>
                  {emptyLabel || t('conversation.empty', 'Start the conversation.')}
                </p>
              </ThreadPrimitive.Empty>
              <ThreadPrimitive.Messages
                components={{
                  UserMessage: UserMessage,
                  AssistantMessage: AssistantMessage,
                  SystemMessage: SystemMessage,
                }}
              />
            </ThreadPrimitive.Viewport>

            {threadEnd ? <div className={styles.threadEnd}>{threadEnd}</div> : null}

            {error ? <div className={styles.error}>{error}</div> : null}

            {pendingAttachments.length > 0 ? (
              <ul
                className={styles.pendingList}
                aria-label={t('conversation.attachments', 'Attachments')}
              >
                {pendingAttachments.map((att) => (
                  <li key={att.id} className={styles.pendingItem}>
                    {att.previewUrl ? (
                      <img src={att.previewUrl} alt={att.name} className={styles.pendingThumb} />
                    ) : (
                      <span>{att.name}</span>
                    )}
                    {onRemoveAttachment ? (
                      <IconButton
                        label={t('conversation.removeAttachment', 'Remove attachment')}
                        onClick={() => onRemoveAttachment(att.id)}
                      >
                        <X size={14} />
                      </IconButton>
                    ) : null}
                  </li>
                ))}
              </ul>
            ) : null}

            <div className={styles.composerDock}>
              <ComposerPrimitive.Root
                className={`${styles.composer} ${dragging ? styles.dragging : ''}`.trim()}
              >
                <div className={styles.composerTools}>
                  {onAddFiles ? (
                    <>
                      <input
                        ref={fileRef}
                        type="file"
                        accept="image/png,image/jpeg,image/webp"
                        multiple
                        className={styles.fileInput}
                        onChange={(e) => {
                          acceptFiles(e.target.files);
                          e.target.value = '';
                        }}
                      />
                      <IconButton
                        label={t('conversation.attachImage', 'Attach image')}
                        onClick={() => fileRef.current?.click()}
                      >
                        <Paperclip size={16} />
                      </IconButton>
                    </>
                  ) : null}
                  {composerLeading}
                </div>
                <ComposerPrimitive.Input
                  className={styles.input}
                  rows={1}
                  placeholder={t('conversation.placeholder', 'Write a message…')}
                  submitOnEnter
                  onPaste={(e) => {
                    if (!onAddFiles) return;
                    const files = Array.from(e.clipboardData.files).filter((f) =>
                      isAllowedImageMime(f.type),
                    );
                    if (files.length) {
                      e.preventDefault();
                      onAddFiles(files);
                    }
                  }}
                />
                <div className={styles.composerActions}>
                  {composerTrailing}
                  <ComposerPrimitive.Cancel asChild>
                    <Button tone="ghost" size="sm" className={styles.cancelBtn}>
                      <Square size={14} /> {t('conversation.stop', 'Stop')}
                    </Button>
                  </ComposerPrimitive.Cancel>
                  <ComposerPrimitive.Send asChild>
                    <IconButton label={t('conversation.send', 'Send')} tone="brand">
                      <Send size={16} />
                    </IconButton>
                  </ComposerPrimitive.Send>
                </div>
              </ComposerPrimitive.Root>
            </div>
            {footer}
          </ThreadPrimitive.Root>
        </div>

        {inspector && !useDrawer ? (
          <aside className={styles.inspector} aria-label={inspectorLabel}>
            {!hideInspectorChrome && onToggleInspector ? (
              <div className={styles.inspectorToggle}>
                <Button size="sm" onClick={onToggleInspector}>
                  {inspectorOpen
                    ? t('conversation.hideDetails', 'Hide details')
                    : t('conversation.showDetails', 'Show details')}
                </Button>
              </div>
            ) : null}
            {inspectorOpen ? <div className={styles.inspectorBody}>{inspector}</div> : null}
          </aside>
        ) : null}

        {inspector && useDrawer ? (
          <ContextDrawer
            open={inspectorOpen}
            onClose={() => onToggleInspector?.()}
            title={inspectorLabel}
            width="min(100vw, 26.5rem)"
          >
            {inspector}
          </ContextDrawer>
        ) : null}
      </div>
    </AssistantRuntimeProvider>
  );
};

const UserMessage: React.FC = () => (
  <MessagePrimitive.Root className={`${styles.message} ${styles.user}`}>
    <span className={`${styles.avatar} ${styles.avatarUser}`} aria-hidden>
      U
    </span>
    <div className={styles.bubble}>
      <MessagePrimitive.Content />
    </div>
  </MessagePrimitive.Root>
);

const SystemMessage: React.FC = () => (
  <MessagePrimitive.Root className={`${styles.message} ${styles.system}`}>
    <MessagePrimitive.Content />
  </MessagePrimitive.Root>
);

const AssistantMessage: React.FC = () => (
  <MessagePrimitive.Root className={`${styles.message} ${styles.assistant}`}>
    <span className={`${styles.avatar} ${styles.avatarAssistant}`} aria-hidden>
      N
    </span>
    <div className={styles.assistantBody}>
      <MessagePrimitive.Content
        components={{
          Text: ({ text }) => <p className={styles.textPart}>{text}</p>,
          Image: ({ image }) => <img src={image} alt="" className={styles.imagePart} />,
          tools: {
            Fallback: ({ toolName, result, status }) => (
              <ToolCard
                tool={{
                  toolName,
                  status:
                    status?.type === 'running'
                      ? 'running'
                      : status?.type === 'incomplete'
                        ? 'failure'
                        : 'success',
                  summary: typeof result === 'string' ? result : undefined,
                  detail: typeof result === 'string' ? undefined : JSON.stringify(result, null, 2),
                }}
              />
            ),
          },
          data: {
            by_name: {
              'nexus.activity': ({ data }) => renderActivityIfPresent(data),
              'nexus.artifact': ({ data }) => (
                <pre className={styles.structuredPart}>{JSON.stringify(data, null, 2)}</pre>
              ),
            },
            Fallback: ({ data, name }) => (
              <pre className={styles.structuredPart}>
                {name}: {JSON.stringify(data)}
              </pre>
            ),
          },
        }}
      />
    </div>
  </MessagePrimitive.Root>
);

export function renderActivityIfPresent(payload: unknown): React.ReactNode {
  const activity = asActivityPayload(payload);
  if (!activity) return null;
  return <AgentActivity activity={activity} />;
}

export function renderToolIfPresent(payload: unknown): React.ReactNode {
  const tool = asToolPayload(payload);
  if (!tool) return null;
  return <ToolCard tool={tool} />;
}
