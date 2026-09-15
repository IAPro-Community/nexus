import React from 'react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Card } from '../../design-system';
import type { ComposerSkillProposal } from '../../types';
import styles from './ComposerSurface.module.scss';

/**
 * Maestro skill proposals with accept/dismiss controls. Applied and
 * unavailable skills are read-only.
 */
export const ComposerSkills: React.FC<{
  skills: ComposerSkillProposal[];
  busy: boolean;
  onUpdateSkill: (skillId: string, state: 'ACCEPTED' | 'REJECTED') => void;
  compact?: boolean;
}> = ({ skills, busy, onUpdateSkill, compact = false }) => {
  const { t } = useTranslation();
  if (skills.length === 0) return null;

  const body = skills.map((skill) => (
    <div key={skill.skill_id} className={styles.skillRow}>
      <div className={styles.skillInfo}>
        <strong>{skill.skill_id}</strong>
        <small className={styles.skillReason}>{skill.reason || skill.applicability}</small>
      </div>
      <Badge tone={skill.state === 'UNAVAILABLE' ? 'danger' : 'default'}>
        {t(`work.composer.skillState.${skill.state}`, skill.state)}
      </Badge>
      {skill.state !== 'UNAVAILABLE' && skill.state !== 'APPLIED' && (
        <div className={styles.skillActions}>
          <Button
            size="sm"
            tone="brand"
            disabled={busy}
            onClick={() => onUpdateSkill(skill.skill_id, 'ACCEPTED')}
          >
            {t('work.composer.accept', 'Accept')}
          </Button>
          <Button
            size="sm"
            disabled={busy}
            onClick={() => onUpdateSkill(skill.skill_id, 'REJECTED')}
          >
            {t('work.composer.dismiss', 'Dismiss')}
          </Button>
        </div>
      )}
    </div>
  ));

  if (compact) return <div className={styles.skillStack}>{body}</div>;

  return (
    <Card className={styles.artifactCard}>
      <strong>{t('work.composer.maestroSkills', 'Maestro skills')}</strong>
      {body}
    </Card>
  );
};
