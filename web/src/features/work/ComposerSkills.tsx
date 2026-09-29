import React, { useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, Card, Input } from '../../design-system';
import type { BalanceEntity, ComposerSkillCatalogItem, ComposerSkillProposal } from '../../types';
import { asStringArray } from '../../lib/safeArray';
import { nexus } from '../../nexus/api';
import styles from './ComposerSurface.module.scss';

function catalogMatches(item: ComposerSkillCatalogItem, query: string): boolean {
  const q = query.trim().toLowerCase();
  if (q.length < 2) return false;
  return [
    item.id,
    item.name,
    item.description,
    item.source,
    ...asStringArray(item.capabilities),
    ...asStringArray(item.triggers),
  ]
    .filter(Boolean)
    .some((value) => String(value).toLowerCase().includes(q));
}

/**
 * Source-agnostic Nexus Skill Catalog. Recommended proposals remain durable
 * session state; catalog search only exposes safe metadata and requires an
 * explicit Use action before a skill becomes ACCEPTED.
 */
export const ComposerSkills: React.FC<{
  skills: ComposerSkillProposal[];
  catalog: ComposerSkillCatalogItem[];
  busy: boolean;
  onUpdateSkill: (skillId: string, state: 'ACCEPTED' | 'REJECTED') => void;
  onSearchCatalog: (query: string) => Promise<void>;
  onSelectCatalogSkill: (skillId: string) => void;
  compact?: boolean;
}> = ({
  skills,
  catalog,
  busy,
  onUpdateSkill,
  onSearchCatalog,
  onSelectCatalogSkill,
  compact = false,
}) => {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const [balanceEntities, setBalanceEntities] = useState<BalanceEntity[]>([]);
  const [balanceLoaded, setBalanceLoaded] = useState(false);

  useEffect(() => {
    let active = true;
    void nexus
      .listResources()
      .then((payload) => {
        if (!active) return;
        setBalanceEntities(payload.balance_entities || []);
      })
      .catch(() => {
        if (!active) return;
        setBalanceEntities([]);
      })
      .finally(() => {
        if (active) setBalanceLoaded(true);
      });
    return () => {
      active = false;
    };
  }, []);

  useEffect(() => {
    const normalized = query.trim();
    if (normalized.length < 2) return undefined;
    const timer = window.setTimeout(() => {
      void onSearchCatalog(normalized);
    }, 180);
    return () => window.clearTimeout(timer);
  }, [onSearchCatalog, query]);

  const visibleProposals = useMemo(() => {
    const q = query.trim().toLowerCase();
    return [...skills]
      .sort((left, right) => {
        const statePriority = (state: ComposerSkillProposal['state']) =>
          state === 'APPLIED' ? 3 : state === 'ACCEPTED' ? 2 : state === 'SUGGESTED' ? 1 : 0;
        return (
          statePriority(right.state) - statePriority(left.state) ||
          Number(right.recommendation_score || 0) - Number(left.recommendation_score || 0) ||
          left.skill_id.localeCompare(right.skill_id)
        );
      })
      .filter((skill) => {
        if (!q) return true;
        return [skill.skill_id, skill.source, skill.reason, skill.applicability]
          .filter(Boolean)
          .some((value) => String(value).toLowerCase().includes(q));
      });
  }, [query, skills]);

  const catalogResults = useMemo(() => {
    const proposalIDs = new Set(skills.map((skill) => skill.skill_id));
    return catalog
      .filter((skill) => !proposalIDs.has(skill.id))
      .filter((skill) => catalogMatches(skill, query))
      .slice(0, 8);
  }, [catalog, query, skills]);

  if (skills.length === 0 && catalog.length === 0) return null;

  const proposals = visibleProposals.map((skill) => (
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

  const discovered = catalogResults.map((skill) => {
    const available = skill.availability === 'AVAILABLE';
    return (
      <div key={skill.id} className={styles.skillRow}>
        <div className={styles.skillInfo}>
          <strong>{skill.name || skill.id}</strong>
          <small className={styles.skillReason}>{skill.description || skill.id}</small>
          <small className={styles.skillReason}>
            {skill.source}
            {skill.version ? ` · v${skill.version}` : ''}
            {skill.id !== skill.name ? ` · ${skill.id}` : ''}
          </small>
        </div>
        <Badge tone={available ? 'default' : 'warning'}>{skill.availability}</Badge>
        <div className={styles.skillActions}>
          <Button
            size="sm"
            tone="brand"
            disabled={busy || !available}
            onClick={() => onSelectCatalogSkill(skill.id)}
          >
            {t('work.composer.useSkill', 'Use')}
          </Button>
        </div>
      </div>
    );
  });

  const content = (
    <>
      <div className={styles.skillStack}>
        <small className={styles.skillReason}>
          {t(
            'work.composer.balanceExecutors',
            'Registered AI executors with measurable limits',
          )}{' '}
          ({balanceEntities.length})
        </small>
        {balanceEntities.length > 0 ? (
          <div className={styles.skillActions}>
            {balanceEntities.map((entity) => (
              <Badge
                key={entity.id}
                tone={entity.available ? 'success' : 'warning'}
                title={`${entity.usage_status} · ${entity.health}`}
              >
                {entity.provider}:{entity.profile} · {Math.round(entity.quota_remaining * 100)}%
              </Badge>
            ))}
          </div>
        ) : balanceLoaded ? (
          <small className={styles.skillReason}>
            {t(
              'work.composer.noBalanceExecutors',
              'No registered AI currently has measurable quota evidence for automatic balancing.',
            )}
          </small>
        ) : null}
      </div>
      <Input
        value={query}
        onChange={setQuery}
        placeholder={t('work.composer.searchSkills', 'Search skills')}
        aria-label={t('work.composer.searchSkills', 'Search skills')}
      />
      {proposals.length ? <div className={styles.skillStack}>{proposals}</div> : null}
      {query.trim().length >= 2 ? (
        <>
          {catalogResults.length ? (
            <>
              <small className={styles.skillReason}>
                {t('work.composer.catalogMatches', 'More skills from the Nexus catalog')}
              </small>
              <div className={styles.skillStack}>{discovered}</div>
            </>
          ) : proposals.length === 0 ? (
            <small className={styles.skillReason}>
              {t('work.composer.noMatchingSkills', 'No matching skills.')}
            </small>
          ) : null}
        </>
      ) : skills.length === 0 ? (
        <small className={styles.skillReason}>
          {t(
            'work.composer.searchCatalogHint',
            'Search the Nexus catalog to add a skill to this session.',
          )}
        </small>
      ) : null}
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
