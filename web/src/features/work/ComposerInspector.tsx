import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ChevronRight, Pencil } from 'lucide-react';
import { Button } from '../../design-system';
import type {
  Agent,
  ComposerSessionView,
  ComposerSkillCatalogItem,
  Project,
  PromptArtifact,
} from '../../types';
import { asArray, asStringArray } from '../../lib/safeArray';
import type { ComposerBriefUnknown, ComposerGate } from './composerModel';
import { ComposerBriefOverview } from './ComposerBriefOverview';
import { ComposerReadiness } from './ComposerReadiness';
import { ComposerSkills } from './ComposerSkills';
import { ComposerPromptPreview } from './ComposerPromptPreview';
import { ComposerContextTab } from './ComposerContextTab';
import { ComposerActionDock } from './ComposerActionDock';
import type { ComposerActionPhase } from './composerModel';
import styles from './ComposerInspector.module.scss';

export type ComposerInspectorTab = 'briefing' | 'context' | 'prompt';

export const ComposerInspector: React.FC<{
  view: ComposerSessionView;
  project: Project;
  artifact: PromptArtifact | null;
  busy: boolean;
  confirmGaps: boolean;
  phase: ComposerActionPhase;
  pendingCount: number;
  actionableGaps: ComposerBriefUnknown[];
  memoryCount: number;
  intelligenceLabel: string;
  agents: Agent[];
  selectedAgentId: string;
  onSelectAgent: (id: string) => void;
  canMaterialize: boolean;
  canExecute: boolean;
  destinationGate?: ComposerGate;
  showRefineInput: boolean;
  refineText: string;
  onToggleRefine: () => void;
  onRefineTextChange: (value: string) => void;
  onCancelRefine: () => void;
  onRefine: () => void;
  skillCatalog: ComposerSkillCatalogItem[];
  onUpdateSkill: (skillId: string, state: 'ACCEPTED' | 'REJECTED') => void;
  onSearchSkillCatalog: (query: string) => Promise<void>;
  onSelectCatalogSkill: (skillId: string) => void;
  onAttach: (path: string) => void;
  onDetach: (path: string) => void;
  onCopy: () => void;
  onTransformFlow: () => void;
  onSendToAgent: (
    prompt: string,
    skills: string[],
    startIfNeeded: boolean,
    contextFingerprintId: string,
  ) => Promise<void>;
  onReviewFinalize: () => void;
  onFinalizeWithGaps: () => void;
  onFocusThread?: () => void;
  contextRefreshing?: boolean;
  onRefreshContext?: () => void;
}> = ({
  view,
  project,
  artifact,
  busy,
  confirmGaps,
  phase,
  pendingCount,
  actionableGaps,
  memoryCount,
  intelligenceLabel,
  agents,
  selectedAgentId,
  onSelectAgent,
  canMaterialize,
  canExecute,
  destinationGate,
  showRefineInput,
  refineText,
  onToggleRefine,
  onRefineTextChange,
  onCancelRefine,
  onRefine,
  skillCatalog,
  onUpdateSkill,
  onSearchSkillCatalog,
  onSelectCatalogSkill,
  onAttach,
  onDetach,
  onCopy,
  onTransformFlow,
  onSendToAgent,
  onReviewFinalize,
  onFinalizeWithGaps,
  onFocusThread,
  contextRefreshing,
  onRefreshContext,
}) => {
  const { t } = useTranslation();
  const [tab, setTab] = useState<ComposerInspectorTab>(() =>
    phase === 'finalized' ? 'prompt' : 'briefing',
  );
  const [motivationOpen, setMotivationOpen] = useState(false);
  const assumptions = asArray<string | { value?: string }>(view.brief.assumptions)
    .map((item) => (typeof item === 'string' ? item : item.value || ''))
    .filter(Boolean);
  const motivation = [...assumptions, ...asStringArray(view.brief.alternatives)];
  const suggestedSkills = (view.skills || []).filter((skill) => skill.state === 'SUGGESTED').length;

  const tabs: Array<{ id: ComposerInspectorTab; label: string }> = [
    { id: 'briefing', label: t('work.composer.tabs.briefing', 'Briefing') },
    { id: 'context', label: t('work.composer.tabs.context', 'Context') },
    { id: 'prompt', label: t('work.composer.tabs.prompt', 'Prompt') },
  ];

  return (
    <div className={styles.root}>
      <div
        className={styles.tabs}
        role="tablist"
        aria-label={t('work.composer.inspector', 'Briefing')}
      >
        {tabs.map((item) => (
          <button
            key={item.id}
            type="button"
            role="tab"
            aria-selected={tab === item.id}
            className={styles.tab}
            onClick={() => setTab(item.id)}
          >
            {item.label}
          </button>
        ))}
      </div>

      <div className={styles.body} role="tabpanel">
        {tab === 'briefing' ? (
          <div className={styles.tabStack}>
            <div className={styles.sectionHead}>
              <h2 className={styles.panelTitle}>{t('work.composer.overview.title', 'Overview')}</h2>
              <Button size="sm" tone="ghost" onClick={() => setTab('prompt')}>
                <Pencil size={13} /> {t('work.composer.editBrief', 'Edit')}
              </Button>
            </div>
            <ComposerBriefOverview view={view} />
            <ComposerReadiness
              view={view}
              onFocusGaps={() => {
                onFocusThread?.();
              }}
            />
            {actionableGaps.length > 0 ? (
              <section className={styles.group}>
                <h3 className={styles.groupTitle}>{t('work.composer.gaps', 'Actionable gaps')}</h3>
                <ul className={styles.plainList}>
                  {actionableGaps.map((gap) => (
                    <li key={gap.id}>{gap.question}</li>
                  ))}
                </ul>
              </section>
            ) : null}
            <details className={styles.collapse} open={suggestedSkills > 0}>
              <summary>
                <span>
                  {t('work.composer.skills', 'Skills')}
                  {suggestedSkills > 0
                    ? ` · ${t('work.composer.skillsSuggested', '{{count}} suggested', { count: suggestedSkills })}`
                    : ''}
                </span>
                <ChevronRight size={14} aria-hidden />
              </summary>
              <ComposerSkills
                skills={view.skills || []}
                catalog={skillCatalog}
                busy={busy}
                compact
                onUpdateSkill={onUpdateSkill}
                onSearchCatalog={onSearchSkillCatalog}
                onSelectCatalogSkill={onSelectCatalogSkill}
              />
            </details>
            {motivation.length > 0 ? (
              <details
                className={styles.collapse}
                open={motivationOpen}
                onToggle={(e) => setMotivationOpen((e.target as HTMLDetailsElement).open)}
              >
                <summary>
                  <span>{t('work.composer.motivation', 'Motivation & context')}</span>
                  <ChevronRight size={14} aria-hidden />
                </summary>
                <ul className={styles.plainList}>
                  {motivation.map((item) => (
                    <li key={item}>{item}</li>
                  ))}
                </ul>
              </details>
            ) : null}
          </div>
        ) : null}

        {tab === 'context' ? (
          <ComposerContextTab
            view={view}
            project={project}
            busy={busy}
            memoryCount={memoryCount}
            intelligenceLabel={intelligenceLabel}
            onAttach={onAttach}
            onDetach={onDetach}
          />
        ) : null}

        {tab === 'prompt' ? (
          <ComposerPromptPreview
            preview={view.brief.preview}
            artifact={artifact}
            busy={busy}
            showRefineInput={showRefineInput}
            refineText={refineText}
            onToggleRefine={onToggleRefine}
            onRefineTextChange={onRefineTextChange}
            onCancelRefine={onCancelRefine}
            onRefine={onRefine}
          />
        ) : null}
      </div>

      <ComposerActionDock
        phase={phase}
        pendingCount={pendingCount}
        busy={busy}
        confirmGaps={confirmGaps}
        artifact={artifact}
        agents={agents}
        selectedAgentId={selectedAgentId}
        onSelectAgent={onSelectAgent}
        canMaterialize={canMaterialize}
        canExecute={canExecute}
        destinationGate={destinationGate}
        flowSuitability={view.flow_suitability}
        promptReviewMissing={asStringArray(view.prompt_review?.missing)}
        promptReviewContradictions={asStringArray(view.prompt_review?.contradictions)}
        projectId={project.id}
        contextRefreshing={contextRefreshing}
        onRefreshContext={onRefreshContext}
        onCopy={onCopy}
        onTransformFlow={onTransformFlow}
        onSendToAgent={onSendToAgent}
        onReviewFinalize={onReviewFinalize}
        onFinalizeWithGaps={onFinalizeWithGaps}
        onSeeGaps={onFocusThread || (() => undefined)}
        onFocusPending={onFocusThread}
      />
    </div>
  );
};
