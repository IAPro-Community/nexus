import React from 'react';
import { useTranslation } from 'react-i18next';
import { ChevronRight } from 'lucide-react';
import { Badge } from '../../design-system';
import type { ComposerContextItem, ComposerSessionView, Project } from '../../types';
import { asStringArray } from '../../lib/safeArray';
import {
  composerContextBudgetPercent,
  composerContextItems,
  composerReferencedFiles,
} from './composerModel';
import { ComposerContextChips } from './ComposerContextChips';
import { ProjectIntelligenceInspector } from './ProjectIntelligenceInspector';
import styles from './ComposerInspector.module.scss';

function shortContextSummary(item: ComposerContextItem): string {
  const compact = String(item.summary || '')
    .replace(/\s+/g, ' ')
    .trim();
  if (compact.length <= 180) return compact;
  return `${compact.slice(0, 177)}…`;
}

export const ComposerContextTab: React.FC<{
  view: ComposerSessionView;
  project: Project;
  busy: boolean;
  memoryCount: number;
  intelligenceLabel: string;
  onAttach: (path: string) => void;
  onDetach: (path: string) => void;
}> = ({ view, project, busy, memoryCount, intelligenceLabel, onAttach, onDetach }) => {
  const { t } = useTranslation();
  const files = composerReferencedFiles(view);
  const autoContext = asStringArray(
    view.brief.context &&
      typeof view.brief.context === 'object' &&
      !Array.isArray(view.brief.context)
      ? (view.brief.context as { existing_state?: unknown }).existing_state
      : view.brief.context,
  );
  const manifest = view.context_manifest;
  const kept = composerContextItems(view, 'KEEP');
  const referenced = composerContextItems(view, 'REFERENCE_ONLY');
  const dropped = composerContextItems(view, 'DROP');
  const budgetPercent = composerContextBudgetPercent(view);

  return (
    <div className={styles.tabStack}>
      <section className={styles.group}>
        <h3 className={styles.groupTitle}>
          {t('work.composer.contextSources.project', 'Project')}
        </h3>
        <p className={styles.summary}>{project.name || project.canonical_path}</p>
        <p className={styles.meta}>{project.canonical_path}</p>
      </section>

      <section
        className={styles.group}
        aria-label={t('work.composer.contextDiscovery', 'Context discovery')}
      >
        <h3 className={styles.groupTitle}>
          {t('work.composer.contextDiscovery', 'Context discovery')}
        </h3>
        {manifest?.fingerprint ? (
          <>
            <p className={styles.summary}>
              {t('work.composer.contextBudget', 'Context budget')}:&nbsp;
              {manifest.budget.used_estimated_tokens} / {manifest.budget.max_estimated_tokens}{' '}
              {t('work.composer.estimatedTokens', 'estimated tokens')} · {budgetPercent}%
            </p>
            <p className={styles.meta}>
              KEEP {manifest.budget.kept_items} · REFERENCE {manifest.budget.referenced_items} ·
              DROP {manifest.budget.dropped_items}
            </p>
            {kept.length ? (
              <ul className={styles.plainList}>
                {kept.slice(0, 8).map((item) => (
                  <li key={item.id}>
                    <Badge tone="success">KEEP</Badge> <strong>{item.kind}</strong> ·{' '}
                    {shortContextSummary(item)}
                    <small className={styles.meta}>
                      {item.source}
                      {item.locator ? ` · ${item.locator}` : ''} · {item.reason}
                    </small>
                  </li>
                ))}
              </ul>
            ) : (
              <p className={styles.summary}>
                {t('work.composer.contextSources.empty', 'None captured yet.')}
              </p>
            )}
          </>
        ) : (
          <p className={styles.summary}>
            {t(
              'work.composer.contextManifestUnavailable',
              'No context manifest has been captured yet.',
            )}
          </p>
        )}
      </section>

      {referenced.length ? (
        <details className={styles.collapse}>
          <summary>
            <span>
              {t('work.composer.contextReferences', 'References')} · {referenced.length}
            </span>
            <ChevronRight size={14} aria-hidden />
          </summary>
          <ul className={styles.plainList}>
            {referenced.map((item) => (
              <li key={item.id}>
                <Badge>REFERENCE</Badge> <strong>{item.kind}</strong> · {shortContextSummary(item)}
                <small className={styles.meta}>{item.reason}</small>
              </li>
            ))}
          </ul>
        </details>
      ) : null}

      {dropped.length ? (
        <details className={styles.collapse}>
          <summary>
            <span>
              {t('work.composer.contextExcluded', 'Excluded from model context')} · {dropped.length}
            </span>
            <ChevronRight size={14} aria-hidden />
          </summary>
          <ul className={styles.plainList}>
            {dropped.map((item) => (
              <li key={item.id}>
                <Badge tone="warning">DROP</Badge> <strong>{item.kind}</strong> ·{' '}
                {shortContextSummary(item)}
                <small className={styles.meta}>{item.reason}</small>
              </li>
            ))}
          </ul>
        </details>
      ) : null}

      {(manifest?.warnings?.length ?? 0) > 0 ? (
        <details className={styles.collapse}>
          <summary>
            <span>
              {t('work.composer.contextWarnings', 'Discovery warnings')} ·{' '}
              {manifest?.warnings?.length}
            </span>
            <ChevronRight size={14} aria-hidden />
          </summary>
          <ul className={styles.plainList}>
            {manifest?.warnings?.map((warning) => (
              <li key={warning}>{warning}</li>
            ))}
          </ul>
        </details>
      ) : null}

      <section className={styles.group}>
        <h3 className={styles.groupTitle}>
          {t('work.composer.contextSources.automatic', 'Automatic context')}
        </h3>
        {autoContext.length ? (
          <ul className={styles.plainList}>
            {autoContext.map((item) => (
              <li key={item}>{item}</li>
            ))}
          </ul>
        ) : (
          <p className={styles.summary}>
            {t('work.composer.contextSources.empty', 'None captured yet.')}
          </p>
        )}
      </section>

      <ComposerContextChips
        projectPath={project.canonical_path}
        files={files}
        busy={busy}
        onAttach={onAttach}
        onDetach={onDetach}
      />

      <section className={styles.group}>
        <h3 className={styles.groupTitle}>
          {t('work.composer.contextSources.memory', 'Memories used')}
        </h3>
        <p className={styles.summary}>
          {memoryCount} · {intelligenceLabel}
        </p>
      </section>

      <details className={styles.collapse}>
        <summary>
          <span>{t('work.projectIntelligence.title', 'Project Intelligence')}</span>
          <ChevronRight size={14} aria-hidden />
        </summary>
        <ProjectIntelligenceInspector projectId={project.id} />
      </details>
    </div>
  );
};
