import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { RefreshCw, Sparkles } from 'lucide-react';
import { Badge, Button } from '../../design-system';
import type {
  Agent,
  ContextReadiness,
  ContextReadinessState,
  MissionRun,
  Project,
  PromptArtifact,
  WorkPlan,
} from '../../types';
import { nexus } from '../../nexus/api';
import { asArray } from '../../lib/safeArray';
import { ComposerSurface } from './ComposerSurface';
import { composerGateForReadiness } from './composerModel';
import { ProjectIntelligenceInspector } from './ProjectIntelligenceInspector';
import styles from './WorkSurface.module.scss';

const readinessTone = (state: ContextReadinessState) =>
  state === 'READY'
    ? 'success'
    : state === 'FAILED'
      ? 'danger'
      : state === 'STALE'
        ? 'warning'
        : 'default';

const FLOW_DRAFT_KEY = (projectId: string) => `iapro:nexus:flow-draft:${projectId}`;

/**
 * Composer is the prompt refinery. Flow Studio (missions) receives the handoff draft.
 */
export const WorkSurface: React.FC<{
  project: Project;
  agents: Agent[];
  onDirect: (agent: Agent) => void;
  onFlowRun?: (run: MissionRun) => void;
  onOpenFlowDrafts?: (plan: WorkPlan) => void;
}> = ({ project, onOpenFlowDrafts }) => {
  const { t } = useTranslation();
  const [readiness, setReadiness] = useState<ContextReadiness | null>(null);
  const [readinessBusy, setReadinessBusy] = useState(false);
  const [error, setError] = useState('');
  const [intelligence, setIntelligence] = useState<{
    available: boolean;
    provider?: string;
    error?: string;
  } | null>(null);
  const [memoryCount, setMemoryCount] = useState(0);
  const [flowError, setFlowError] = useState('');

  const refreshContext = useCallback(async () => {
    try {
      setReadiness(await nexus.getContextReadiness(project.id));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [project.id]);

  const refreshIntelligence = useCallback(async () => {
    try {
      const status = await nexus.getIntelligence(project.id);
      setIntelligence({
        available: status.available,
        provider: status.provider || status.mode,
        error: status.error,
      });
    } catch (err) {
      setIntelligence({
        available: false,
        error: err instanceof Error ? err.message : String(err),
      });
    }
  }, [project.id]);

  const refreshMemories = useCallback(async () => {
    try {
      const items = await nexus.listIntelligenceMemories(project.id);
      setMemoryCount(asArray(items).length);
    } catch {
      setMemoryCount(0);
    }
  }, [project.id]);

  useEffect(() => {
    void refreshContext();
    void refreshIntelligence();
    void refreshMemories();
  }, [refreshContext, refreshIntelligence, refreshMemories]);

  const state: ContextReadinessState = readiness?.state ?? 'MISSING';
  const gate = composerGateForReadiness(state);
  const intelligenceTone = intelligence?.available ? 'success' : 'warning';

  const prepareContext = async (createContext = false) => {
    setReadinessBusy(true);
    setError('');
    try {
      setReadiness(await nexus.prepareContext(project.id, createContext));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setReadinessBusy(false);
    }
  };

  const materializeFlow = async (artifact: PromptArtifact) => {
    setFlowError('');
    try {
      const plan = await nexus.materializePromptArtifact(artifact.id);
      try {
        window.sessionStorage.setItem(FLOW_DRAFT_KEY(project.id), plan.id);
      } catch {
        /* ignore */
      }
      onOpenFlowDrafts?.(plan);
    } catch (err) {
      setFlowError(err instanceof Error ? err.message : String(err));
    }
  };

  return (
    <div className="nx-surface-scroll nx-composer-surface nx-composer-workbench">
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">
            <Sparkles size={13} /> {t('work.composerEyebrow', 'Composer')}
          </span>
          <h1>{t('work.composerTitle', 'Elaboração de intenção')}</h1>
          <p>
            {t(
              'work.composerIntro',
              'Uma decisão por rodada. Depois: Copiar, enviar ao agente ou transformar em Flow.',
            )}
          </p>
        </div>
        <div className={`nx-composer-header-actions ${styles.headerChips}`}>
          <Badge tone={readinessTone(state)}>
            {t('work.contextChip', 'Contexto')} {t(`work.gateState.${state}`, state)}
          </Badge>
          <Badge tone={intelligenceTone}>
            {t('work.intelligenceChip', 'Intelligence')}{' '}
            {intelligence?.available
              ? `${t('work.intelligenceReady', 'READY')}${
                  intelligence.provider ? ` · ${intelligence.provider}` : ''
                }`
              : t('work.intelligenceOff', 'OFF')}
          </Badge>
          <Badge tone={memoryCount > 0 ? 'success' : 'default'}>
            {t('work.remembersChip', 'Nexus remembers')} · {memoryCount}
          </Badge>
          {!gate.canMaterialize && gate.action !== 'WAIT' && (
            <Button
              size="sm"
              tone="brand"
              disabled={readinessBusy}
              onClick={() =>
                void prepareContext(
                  gate.action === 'PREPARE' ||
                    Boolean(readiness?.error?.includes('durable project context is missing')),
                )
              }
            >
              <RefreshCw size={13} />{' '}
              {readinessBusy
                ? t('work.checking', 'Verificando…')
                : gate.action === 'PREPARE'
                  ? t('work.createBaseContext', 'Criar contexto base')
                  : t('work.refreshContext', 'Atualizar contexto')}
            </Button>
          )}
        </div>
      </div>

      {!gate.canMaterialize && (
        <p className={styles.gateHint} data-state={state}>
          {t(gate.reasonKey, gate.reason)}
          {state === 'FAILED' && readiness?.error ? ` · ${readiness.error}` : ''}
        </p>
      )}

      {error && <div className="nx-inline-error">{error}</div>}
      {flowError && (
        <div className="nx-inline-error">
          {t('work.flowMaterializeFailed', 'Falha ao materializar Flow')}: {flowError}
        </div>
      )}

      <div
        className="nx-composer-flow-region"
        data-gate={gate.canMaterialize ? 'ready' : 'destinations-blocked'}
      >
        <ComposerSurface
          project={project}
          destinationGate={gate}
          onTransformFlow={(artifact) => void materializeFlow(artifact)}
        />
      </div>

      <ProjectIntelligenceInspector projectId={project.id} />
    </div>
  );
};

export { FLOW_DRAFT_KEY };
