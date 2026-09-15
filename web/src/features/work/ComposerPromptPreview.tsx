import React from 'react';
import { useTranslation } from 'react-i18next';
import { RefreshCw } from 'lucide-react';
import { Badge, Button, Card, Input } from '../../design-system';
import type { PromptArtifact } from '../../types';
import styles from './ComposerSurface.module.scss';

/**
 * Canonical prompt surface. Renders the live preview from the brief while the
 * session is still being elaborated and the immutable finalized artifact plus
 * its refine controls once an artifact exists.
 */
export const ComposerPromptPreview: React.FC<{
  preview?: string;
  artifact: PromptArtifact | null;
  busy: boolean;
  showRefineInput: boolean;
  refineText: string;
  onToggleRefine: () => void;
  onRefineTextChange: (value: string) => void;
  onCancelRefine: () => void;
  onRefine: () => void;
}> = ({
  preview,
  artifact,
  busy,
  showRefineInput,
  refineText,
  onToggleRefine,
  onRefineTextChange,
  onCancelRefine,
  onRefine,
}) => {
  const { t } = useTranslation();

  if (!artifact) {
    if (!preview) return null;
    return (
      <Card className={styles.artifactCard}>
        <div className={styles.artifactHeader}>
          <strong>{t('work.composer.livePreview', 'Live prompt preview')}</strong>
          <Badge tone="brand">{t('work.composer.draftPreview', 'Draft')}</Badge>
        </div>
        <pre className={styles.preWrap}>{preview}</pre>
      </Card>
    );
  }

  return (
    <Card className={styles.artifactCard}>
      <div className={styles.artifactHeader}>
        <strong>
          {t('work.composer.canonicalPrompt', 'Canonical prompt')} · v{artifact.version}
        </strong>
        <Badge tone="success">
          {t('work.composer.immutableVersion', 'Immutable version')} #{artifact.version}
        </Badge>
      </div>
      <pre className={styles.preWrap}>{artifact.content}</pre>
      <div className={`${styles.headerActions} ${styles.artifactActions}`}>
        <Button disabled={busy} onClick={onToggleRefine}>
          <RefreshCw size={14} /> {t('work.refine', 'Refine')} (v{artifact.version + 1})
        </Button>
      </div>

      {showRefineInput && (
        <div className={styles.refinementPanel}>
          <small className={styles.refinementLabel}>
            {t('work.composer.refineHint', 'Optional refinement instruction:')}
          </small>
          <Input
            value={refineText}
            onChange={onRefineTextChange}
            placeholder={t(
              'work.composer.refinePlaceholder',
              "E.g. 'Add PostgreSQL support', 'Focus only on the REST API'…",
            )}
            disabled={busy}
          />
          <div className={styles.refinementActions}>
            <Button size="sm" disabled={busy} onClick={onCancelRefine}>
              {t('common.cancel', 'Cancel')}
            </Button>
            <Button size="sm" tone="brand" disabled={busy} onClick={onRefine}>
              {busy
                ? t('work.composer.generatingVersion', 'Generating v{{version}}…', {
                    version: artifact.version + 1,
                  })
                : t('work.composer.generateRevision', 'Generate new revision (v{{version}})', {
                    version: artifact.version + 1,
                  })}
            </Button>
          </div>
        </div>
      )}
    </Card>
  );
};
