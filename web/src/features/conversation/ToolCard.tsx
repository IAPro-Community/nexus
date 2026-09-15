import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  CircleAlert,
  LoaderCircle,
  Wrench,
} from 'lucide-react';
import { Badge, IconButton } from '../../design-system';
import type { NexusToolPayload } from './types';
import styles from './ToolCard.module.scss';

const toneFor = (status: NexusToolPayload['status']) => {
  if (status === 'success') return 'success' as const;
  if (status === 'failure') return 'danger' as const;
  if (status === 'running') return 'info' as const;
  return 'default' as const;
};

/**
 * Extensible tool result card. Stdout/detail stays collapsed by default.
 */
export const ToolCard: React.FC<{ tool: NexusToolPayload }> = ({ tool }) => {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const Icon =
    tool.status === 'running'
      ? LoaderCircle
      : tool.status === 'success'
        ? CheckCircle2
        : tool.status === 'failure'
          ? CircleAlert
          : Wrench;

  return (
    <article className={styles.root} data-status={tool.status}>
      <header className={styles.header}>
        <div className={styles.titleRow}>
          <Icon
            size={15}
            className={tool.status === 'running' ? styles.spin : undefined}
            aria-hidden
          />
          <strong>{tool.toolName}</strong>
          <Badge tone={toneFor(tool.status)}>
            {t(`conversation.tool.${tool.status}`, tool.status)}
          </Badge>
        </div>
        {tool.detail ? (
          <IconButton
            label={
              open ? t('conversation.collapse', 'Collapse') : t('conversation.expand', 'Expand')
            }
            onClick={() => setOpen((v) => !v)}
          >
            {open ? <ChevronDown size={16} /> : <ChevronRight size={16} />}
          </IconButton>
        ) : null}
      </header>
      {tool.summary ? <p className={styles.summary}>{tool.summary}</p> : null}
      {open && tool.detail ? <pre className={styles.detail}>{tool.detail}</pre> : null}
    </article>
  );
};
