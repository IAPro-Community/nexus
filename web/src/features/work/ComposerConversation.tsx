import React from 'react';
import { useTranslation } from 'react-i18next';
import { Send } from 'lucide-react';
import { Button, Card } from '../../design-system';
import type { ComposerSessionView, ContextFileRef } from '../../types';
import { ClarificationPrompt } from './ClarificationPrompt';
import { ComposerContextChips } from './ComposerContextChips';
import styles from './ComposerSurface.module.scss';

type BriefUnknown = NonNullable<ComposerSessionView['brief']['unknowns']>[number];

/**
 * Elaboration conversation column: existing turns, the active clarification
 * prompt (when the session is not finalized), attached context files and the
 * message composer.
 */
export const ComposerConversation: React.FC<{
  view: ComposerSessionView;
  nextUnknown: BriefUnknown | null;
  busy: boolean;
  message: string;
  onMessageChange: (value: string) => void;
  onSend: () => void;
  onResolveUnknown: (unknownId: string, status: string, answer?: string) => void;
  onRemember: (unknown: { id: string; field?: string }, answer: string) => void;
  onDismiss: (unknownId: string) => void;
  projectPath?: string;
  contextFiles?: ContextFileRef[];
  onAttachContext?: (path: string) => void;
  onDetachContext?: (path: string) => void;
}> = ({
  view,
  nextUnknown,
  busy,
  message,
  onMessageChange,
  onSend,
  onResolveUnknown,
  onRemember,
  onDismiss,
  projectPath,
  contextFiles,
  onAttachContext,
  onDetachContext,
}) => {
  const { t } = useTranslation();
  const turns = view.turns || [];
  const isFinalized = view.session.state === 'FINALIZED';
  const showContext = !isFinalized && Boolean(projectPath && onAttachContext && onDetachContext);

  return (
    <Card className={styles.conversationCard}>
      <strong>{t('work.composer.conversation', 'Elaboration conversation')}</strong>
      <div className={styles.turns}>
        {turns.length === 0 ? (
          <p className={styles.mutedCopy}>
            {t(
              'work.composer.emptyTurns',
              'Describe context, desired outcome and constraints. Composer keeps this elaboration.',
            )}
          </p>
        ) : (
          turns.map((turn) => (
            <div
              key={turn.id}
              className={`${styles.turn} ${turn.role === 'USER' ? styles.turnUser : ''}`.trim()}
            >
              <small>
                {turn.role === 'USER'
                  ? t('work.composer.you', 'You')
                  : t('work.composer.composerRole', 'Composer')}
              </small>
              <p>{turn.content}</p>
            </div>
          ))
        )}
        {nextUnknown && !isFinalized ? (
          <ClarificationPrompt
            item={{
              id: nextUnknown.id,
              question: nextUnknown.question,
              rationale: nextUnknown.rationale,
              severity: nextUnknown.severity,
              suggested_options: nextUnknown.suggested_options,
              default_choice: nextUnknown.default_choice,
            }}
            busy={busy}
            allowDismiss={nextUnknown.severity !== 'BLOCKING'}
            showRemember
            onAnswer={(answer) => onResolveUnknown(nextUnknown.id, 'ANSWERED', answer)}
            onDismiss={() => onDismiss(nextUnknown.id)}
            onRemember={(answer) => onRemember(nextUnknown, answer)}
          />
        ) : null}
      </div>
      {showContext && (
        <ComposerContextChips
          projectPath={projectPath as string}
          files={contextFiles || []}
          busy={busy}
          onAttach={onAttachContext as (path: string) => void}
          onDetach={onDetachContext as (path: string) => void}
        />
      )}
      {!isFinalized && (
        <div className={`${styles.goalBarRow} ${styles.flexRowGap8mt}`}>
          <textarea
            className={`${styles.textarea} ${styles.composerTextarea}`}
            value={message}
            onChange={(e) => onMessageChange(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey && !e.nativeEvent.isComposing) {
                e.preventDefault();
                onSend();
              }
            }}
            placeholder={t(
              'work.composer.messagePlaceholder',
              'Add a requirement, decision or gap answer… (Enter sends, Shift+Enter new line)',
            )}
            rows={2}
            disabled={busy}
          />
          <Button
            tone="brand"
            disabled={!message.trim() || busy}
            onClick={onSend}
            className={styles.sendButton}
          >
            <Send size={14} />{' '}
            {busy ? t('work.composer.sending', 'Sending…') : t('work.composer.send', 'Send')}
          </Button>
        </div>
      )}
    </Card>
  );
};
