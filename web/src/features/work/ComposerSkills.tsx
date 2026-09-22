import React, { useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Card, Input } from '../../design-system';
import type { ComposerSkillProposal } from '../../types';
import styles from './ComposerSurface.module.scss';

/**
 * Source-agnostic Nexus Skill Catalog recommendations. Ranking/provenance are
 * backend-owned; this surface only searches and applies durable proposals.
 */
export const ComposerSkills: React.FC<{
  skills: ComposerSkillProposal[];
  busy: boolean;
  onUpdateSkill: (skillId: string, state: 'ACCEPTED' | 'REJECTED') => void;
  compact?: boolean;
}> = ({ skills, busy, onUpdateSkill, compact = false }) => {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return [...skills]
      .sort(
        (left, right) =>
          Number(right.recommendation_score || 0) - Number(left.recommendation_score || 0) ||
          left.skill_id.localeCompare(right.skill_id),
      )
      .filter((skill) => {
        if (!q) return true;
        return [skill.skill_id, skill.source, skill.reason, skill.applicability]
          .filter(Boolean)
          .some((value) => String(value).toLowerCase().includes(q));
      });
  }, [query, skills]);

  if (skills.length === 0) return null;

  const body = visible.map((skill) => (
    <div key={skill.skill_id} className={styles.skillRow}>
      <div className={styles.skillInfo}>
        <strong>{skill.skill_id}</strong>
        <small className={styles.skillReason}>{skill.reason || skill.applicability}</small>
        <small className={styles.skillReason}>
          {skill.source || 'nexus'}
          {skill.version ? ` · v${skill.version}` : ''}
          {skill.recommendation_score ? ` · score ${skill.recommendation_score}` : ''}
        </small>
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

  const content = (
    <>
      {skills.length > 4 ? (
        <Input
          value={query}
          onChange={setQuery}
          placeholder={t('work.composer.searchSkills', 'Search recommended skills')}
          aria-label={t('work.composer.searchSkills', 'Search recommended skills')}
        />
      ) : null}
      {visible.length ? (
        <div className={styles.skillStack}>{body}</div>
      ) : (
        <small className={styles.skillReason}>
          {t('work.composer.noMatchingSkills', 'No matching recommended skills.')}
        </small>
      )}
    </>
  );

  if (compact) return content;

  return (
    <Card className={styles.artifactCard}>
      <strong>{t('work.composer.skills', 'Skills')}</strong>
      {content}
    </Card>
  );
};
