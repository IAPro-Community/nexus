import React from 'react';
import { useTranslation } from 'react-i18next';
import { ChevronRight } from 'lucide-react';
import type { ComposerSessionView, Project } from '../../types';
import { asStringArray } from '../../lib/safeArray';
import { composerReferencedFiles } from './composerModel';
import { ComposerContextChips } from './ComposerContextChips';
import { ProjectIntelligenceInspector } from './ProjectIntelligenceInspector';
import styles from './ComposerInspector.module.scss';

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

  return (
    <div className={styles.tabStack}>
      <section className={styles.group}>
        <h3 className={styles.groupTitle}>
          {t('work.composer.contextSources.project', 'Project')}
        </h3>
        <p className={styles.summary}>{project.name || project.canonical_path}</p>
        <p className={styles.meta}>{project.canonical_path}</p>
      </section>
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
