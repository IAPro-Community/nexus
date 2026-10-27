import React from 'react';
import { useTranslation } from 'react-i18next';
import { ArrowRight, Plus, Sparkles } from 'lucide-react';
import { Button, Card, InlineAlert, Select } from '../../design-system';
import type { Agent, MissionRun, Project, WorkPlan } from '../../types';
import { unresolvedBlocking } from './clarificationModel';
import { ClarificationPrompt } from './ClarificationPrompt';
import { FlowCanvas } from './FlowCanvas';
import { FlowGoalBar } from './FlowGoalBar';
import { FeatureTour } from './FeatureTour';
import { ComposerHandoffSummary } from './ComposerHandoffSummary';
import { FlowPipeline } from './FlowPipeline';
import { FlowStepInspector } from './FlowStepInspector';
import type { FlowDraftModel } from './flowModel';
import { useFlowDraftEditor } from './useFlowDraftEditor';
import styles from './FlowDraftEditor.module.scss';

/**
 * Compact single-goal Flow draft editor: goal bar, leader policy, pipeline/graph
 * canvas and step inspector over the existing WorkPlan/Mission engine. Draft edits
 * never execute work until Approve & Run.
 */
export const FlowDraftEditor: React.FC<{
  project: Project;
  agents: Agent[];
  onRunCreated?: (run: MissionRun) => void;
  onPlanDeleted?: () => void;
  initialGoal?: string;
  initialPlan?: WorkPlan | null;
}> = ({ project, agents, onRunCreated, onPlanDeleted, initialGoal = '', initialPlan = null }) => {
  const { t } = useTranslation();
  const controller = useFlowDraftEditor({
    project,
    onRunCreated,
    onPlanDeleted,
    initialGoal,
    initialPlan,
  });
  const {
    clarification,
    clarificationAnswers,
    setClarificationAnswers,
    planError,
    flowNotice,
    flowErrors,
    flowDraft,
    setFlowDraft,
    selectedStep,
    selectedStepId,
    setSelectedStepId,
    stepComparison,
    studioView,
    setStudioView,
    generating,
    createManualPlan,
    resolveClarification,
    changeFlowStep,
    runStepAction,
    connectStep,
  } = controller;

  return (
    <div className="nx-composer-flow-clean">
      <FeatureTour projectId={project.id} surface="flow-editor" />
      <div data-tour="flow-goal">
        <FlowGoalBar controller={controller} agents={agents} />
      </div>
      <div data-tour="flow-handoff">
        <ComposerHandoffSummary plan={controller.selectedPlan} />
      </div>

      {planError && (
        <InlineAlert
          tone="danger"
          title={t('flow.planErrorTitle', 'Nexus Intelligence could not generate the plan')}
        >
          {planError}
        </InlineAlert>
      )}

      {clarification && (
        <Card className={styles.clarificationCard}>
          <div className={styles.clarificationHeader}>
            <strong>
              {t('flow.clarificationTitle', 'Clarification required before planning')}
            </strong>
            <p className={`nx-muted-copy ${styles.clarificationSubtitle}`}>
              {t(
                'flow.clarificationSubtitle',
                'Nexus stopped instead of guessing. Answers become durable facts.',
              )}
            </p>
          </div>
          <div className={styles.clarificationGrid}>
            {unresolvedBlocking(clarification).map((item) => (
              <ClarificationPrompt
                key={item.key}
                item={{
                  id: item.key,
                  question: item.question,
                  rationale: item.rationale,
                  severity: item.level,
                  suggested_options: item.suggested_options,
                  default_choice: item.default_choice,
                }}
                busy={generating}
                onAnswer={(answer) =>
                  setClarificationAnswers((prev) => ({ ...prev, [item.key]: answer }))
                }
              />
            ))}
          </div>
          <div className={styles.clarificationActions}>
            <Button
              tone="brand"
              disabled={
                generating ||
                unresolvedBlocking(clarification).some(
                  (item) => !(clarificationAnswers[item.key] || '').trim(),
                )
              }
              onClick={() => void resolveClarification()}
            >
              <ArrowRight size={14} /> {t('flow.continuePlanning', 'Continue planning')}
            </Button>
          </div>
        </Card>
      )}

      {flowNotice && (
        <InlineAlert tone="success" title={t('flow.draftTitle', 'Flow Draft')}>
          {flowNotice}
        </InlineAlert>
      )}
      {flowErrors.length > 0 && (
        <InlineAlert tone="warning" title={t('flow.needsAttention', 'Flow needs attention')}>
          {flowErrors.join(' ')}
        </InlineAlert>
      )}

      <div className="nx-composer-flow-body" data-has-draft={flowDraft ? 'true' : 'false'}>
        <div className="nx-composer-flow-canvas-pane" data-tour="flow-canvas">
          {flowDraft ? (
            <div className={styles.studioToolbar} data-tour="flow-views">
              <div
                className={styles.viewToggle}
                role="tablist"
                aria-label={t('flow.viewPipeline', 'Pipeline')}
              >
                <button
                  type="button"
                  role="tab"
                  aria-selected={studioView === 'pipeline'}
                  data-active={studioView === 'pipeline' ? 'true' : undefined}
                  onClick={() => setStudioView('pipeline')}
                >
                  {t('flow.viewPipeline', 'Pipeline')}
                </button>
                <button
                  type="button"
                  role="tab"
                  aria-selected={studioView === 'graph'}
                  data-active={studioView === 'graph' ? 'true' : undefined}
                  onClick={() => setStudioView('graph')}
                >
                  {t('flow.viewGraph', 'Graph')}
                </button>
              </div>
            </div>
          ) : null}
          {!flowDraft && (
            <Card className="nx-composer-empty-canvas">
              <Sparkles size={22} className={styles.emptyCanvasIcon} />
              <strong>{t('flow.emptyCanvasTitle', 'Empty pipeline')}</strong>
              <p>
                {t(
                  'flow.emptyCanvasHint',
                  'Generate a draft with AI or create a base structure manually.',
                )}
              </p>
              <div className={styles.emptyCanvasActions}>
                <Button
                  size="sm"
                  tone="default"
                  disabled={generating}
                  onClick={() => void createManualPlan()}
                >
                  <Plus size={13} /> {t('flow.newFlow', 'New Flow')}
                </Button>
              </div>
            </Card>
          )}
          {flowDraft && studioView === 'pipeline' && (
            <FlowPipeline
              flow={flowDraft}
              selectedId={selectedStepId}
              onSelect={setSelectedStepId}
            />
          )}
          {flowDraft && studioView === 'graph' && (
            <>
              <div className="nx-flow-editor-toolbar">
                <div className={styles.autonomyToolbar}>
                  <span className={styles.autonomyLabel}>
                    {t('flow.autonomyPolicy', 'Autonomy policy:')}
                  </span>
                  <Select
                    value={flowDraft.policy}
                    onChange={(val) =>
                      setFlowDraft({
                        ...flowDraft,
                        policy: val as FlowDraftModel['policy'],
                        policyStored: true,
                      })
                    }
                    options={[
                      { value: 'GUIDED', label: t('flow.policyGuided', 'Guided') },
                      { value: 'AUTONOMOUS', label: t('flow.policyAutonomous', 'Autonomous') },
                    ]}
                    selectStyle={{ minWidth: 140 }}
                  />
                </div>
              </div>
              <FlowCanvas
                flow={flowDraft}
                selectedId={selectedStepId}
                onSelect={setSelectedStepId}
                onConnect={connectStep}
              />
            </>
          )}
        </div>
        {flowDraft && selectedStep && (
          <div className="nx-composer-flow-inspector-pane">
            <FlowStepInspector
              flow={flowDraft}
              step={selectedStep}
              agents={agents}
              onChange={changeFlowStep}
              onAction={runStepAction}
            />
            {stepComparison && (
              <Card className={styles.stepCompare}>
                <pre>{stepComparison}</pre>
              </Card>
            )}
          </div>
        )}
      </div>
    </div>
  );
};
