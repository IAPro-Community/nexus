import React from 'react';
import { useTranslation } from 'react-i18next';
import { History, Plus, Sparkles, Trash2 } from 'lucide-react';
import { Badge, Button, ConfirmDialog, Select } from '../../design-system';
import type { ComposerSessionView, ContextReadinessState } from '../../types';
import {
  composerSessionStateKey,
  formatComposerRelativeTime,
  formatComposerSessionTitle,
} from './composerModel';
import styles from './ComposerWorkspaceHeader.module.scss';

export type ComposerWorkspaceHeaderProps = {
  view: ComposerSessionView;
  sessionOptions: { value: string; label: string }[];
  showSwitcher: boolean;
  busy: boolean;
  onSelectSession: (id: string) => void;
  onNewSession: () => void;
  onDeleteSession: (id: string, expectedRevision: number) => void;
  onToggleHistory?: () => void;
  historyOpen?: boolean;
  contextState: ContextReadinessState;
  intelligenceLabel: string;
  intelligenceReady: boolean;
  memoryCount: number;
};

export const ComposerWorkspaceHeader: React.FC<ComposerWorkspaceHeaderProps> = ({
  view,
  sessionOptions,
  showSwitcher,
  busy,
  onSelectSession,
  onNewSession,
  onDeleteSession,
  onToggleHistory,
  historyOpen,
  contextState,
  intelligenceLabel,
  intelligenceReady,
  memoryCount,
}) => {
  const { t, i18n } = useTranslation();
  const [deleteOpen, setDeleteOpen] = React.useState(false);
  const title = formatComposerSessionTitle(
    view.session.title,
    view.session.id,
    view.session.created_at,
  );
  const updated = formatComposerRelativeTime(view.session.updated_at, Date.now(), i18n.language);
  const messageCount = (view.turns || []).length;
  const active = view.session.state !== 'FINALIZED';

  return (
    <header className={styles.root} data-tour="composer-session">
      <div className={styles.top}>
        <div className={styles.brand}>
          <span className={styles.icon} aria-hidden>
            <Sparkles size={16} />
          </span>
          <div>
            <h1 className={styles.product}>{t('work.composerEyebrow', 'Composer')}</h1>
            <p className={styles.lede}>
              {t(
                'work.composerWorkspaceIntro',
                'Refine ideas into executable plans. Composer understands the project context and prepares Agent or Flow execution.',
              )}
            </p>
          </div>
        </div>
        <div className={styles.actions}>
          {showSwitcher ? (
            <Select
              value={view.session.id}
              onChange={(id) => {
                if (id && id !== view.session.id) onSelectSession(id);
              }}
              options={sessionOptions}
              aria-label={t('work.composer.previousSessions', 'Previous elaborations')}
            />
          ) : (
            <Button
              size="sm"
              onClick={onToggleHistory}
              aria-pressed={historyOpen}
              disabled={!onToggleHistory}
            >
              <History size={14} /> {t('work.composer.history', 'History')}
            </Button>
          )}
          <Button
            size="sm"
            tone="danger"
            disabled={busy}
            onClick={() => setDeleteOpen(true)}
            title={t('work.composer.deleteElaboration', 'Excluir elaboração')}
          >
            <Trash2 size={14} /> {t('common.delete', 'Excluir')}
          </Button>
          <Button size="sm" tone="brand" disabled={busy} onClick={onNewSession}>
            <Plus size={14} /> {t('work.composer.newElaboration', 'New elaboration')}
          </Button>
        </div>
      </div>

      <ConfirmDialog
        open={deleteOpen}
        title={t('work.composer.deleteElaboration', 'Excluir elaboração')}
        description={t(
          'work.composer.deleteElaborationConfirm',
          'Excluir "{{title}}" e suas mensagens e artefatos? Esta ação não pode ser desfeita. Elaborações vinculadas a um Flow ou entrega não podem ser excluídas.',
          { title },
        )}
        confirmLabel={t('work.composer.deleteElaboration', 'Excluir elaboração')}
        onCancel={() => setDeleteOpen(false)}
        onConfirm={() => {
          setDeleteOpen(false);
          onDeleteSession(view.session.id, view.session.revision);
        }}
      />
      <div className={styles.sessionRow}>
        <div className={styles.sessionMeta}>
          <div className={styles.titleRow}>
            <h2 className={styles.sessionTitle}>{title}</h2>
            {active ? (
              <Badge tone="success">
                {t('work.composer.activeElaboration', 'Active elaboration')}
              </Badge>
            ) : (
              <Badge tone="default">
                {t(composerSessionStateKey(view.session.state), view.session.state)}
              </Badge>
            )}
          </div>
          <p className={styles.metaLine}>
            {updated
              ? t('work.composer.updatedAgo', 'Updated {{time}}', { time: updated })
              : t('work.composer.updatedUnknown', 'Recently updated')}
            <span aria-hidden> · </span>
            {t('work.composer.messageCount', '{{count}} messages', { count: messageCount })}
          </p>
        </div>
        <ul className={styles.statusList}>
          <li>
            <span>{t('work.contextChip', 'Context')}</span>
            <strong data-tone={contextState === 'READY' ? 'ready' : 'warn'}>
              {t(`work.gateState.${contextState}`, contextState)}
            </strong>
          </li>
          <li>
            <span>{t('work.intelligenceChip', 'Intelligence')}</span>
            <strong data-tone={intelligenceReady ? 'ready' : 'warn'}>{intelligenceLabel}</strong>
          </li>
          <li>
            <span>{t('work.memoriesChip', 'Memories')}</span>
            <strong>{memoryCount}</strong>
          </li>
        </ul>
      </div>
    </header>
  );
};
