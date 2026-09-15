import React from 'react';
import { useTranslation } from 'react-i18next';
import { Plus } from 'lucide-react';
import { Badge, Button, Select } from '../../design-system';
import type { ComposerSessionView } from '../../types';
import { composerSessionStateKey, formatComposerSessionTitle } from './composerModel';
import styles from './ComposerSurface.module.scss';

/**
 * Deliberative header: session title, archetype, state badge, the session
 * switcher (when more than one exists) and the new-session action.
 */
export const ComposerSessionHeader: React.FC<{
  view: ComposerSessionView;
  sessionOptions: { value: string; label: string }[];
  showSwitcher: boolean;
  busy: boolean;
  onSelectSession: (id: string) => void;
  onNewSession: () => void;
}> = ({ view, sessionOptions, showSwitcher, busy, onSelectSession, onNewSession }) => {
  const { t } = useTranslation();
  const archetype = view.brief.intent?.archetype;
  const archetypeLabel = archetype ? t(`work.composer.archetypes.${archetype}`, archetype) : null;

  return (
    <div className={styles.deliberativeHeader}>
      <div>
        <h2 className={styles.deliberativeTitle}>
          {formatComposerSessionTitle(view.session.title, view.session.id, view.session.created_at)}
        </h2>
      </div>
      <div className={styles.flexRowGap8Center}>
        {archetypeLabel && (
          <Badge tone="default">
            {t('work.composer.archetype', 'Archetype')}: {archetypeLabel}
          </Badge>
        )}
        {showSwitcher && (
          <Select
            value={view.session.id}
            onChange={(id) => {
              if (id && id !== view.session.id) onSelectSession(id);
            }}
            options={sessionOptions}
            selectClassName={styles.sessionSelect}
          />
        )}
        <Badge tone={view.session.state === 'FINALIZED' ? 'success' : 'brand'}>
          {t(composerSessionStateKey(view.session.state), view.session.state)}
        </Badge>
        <Button size="sm" disabled={busy} onClick={onNewSession}>
          <Plus size={13} /> {t('work.composer.newSession', 'New')}
        </Button>
      </div>
    </div>
  );
};
