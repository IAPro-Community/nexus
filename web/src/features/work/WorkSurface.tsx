import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { RefreshCw } from 'lucide-react';
import { Button } from '../../design-system';
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
import styles from './WorkSurface.module.scss';

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
  onOpenIntelligenceSettings?: () => void;
}> = ({ project, onOpenFlowDrafts, onOpenIntelligenceSettings }) => {
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
  const intelligenceLabel = intelligence?.available
    ? intelligence.provider || t('work.intelligenceReady', 'READY')
    : t('work.intelligenceOff', 'OFF');

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

  // Composer is a model-assisted planning experience, not a static form.
  // Avoid showing a misleading "not configured" state while the status loads.
  if (intelligence === null) {
    return (
      <div className={`nx-surface-scroll nx-composer-surface ${styles.workbench}`}>
        <p role="status" aria-live="polite">
          {t('work.composer.checkingIntelligence', 'Verificando Nexus Intelligence…')}
        </p>
      </div>
    );
  }
  // Do not render editable sessions or creation until Intelligence is configured.
  if (!intelligence.available) {
    return (
      <div className={`nx-surface-scroll nx-composer-surface ${styles.workbench}`}>
        <div className={styles.contextFix} role="status" aria-live="polite">
          <h2>{t('work.composer.intelligenceRequiredTitle', 'Configure uma IA para usar o Composer')}</h2>
          <p>
            {t(
              'work.composer.intelligenceRequiredDescription',
              'O Composer utiliza a IA configurada em Nexus Intelligence para compreender sua ideia, investigar alternativas e preparar um plano. Sem ela, não inicia entrevistas por checklist.',
            )}
          </p>
          {intelligence?.error ? <p className="nx-inline-error">{intelligence.error}</p> : null}
          <div className={styles.contextFix}>
            <Button tone="brand" onClick={() => onOpenIntelligenceSettings?.()} disabled={!onOpenIntelligenceSettings}>
              {t('work.composer.configureIntelligence', 'Abrir Nexus Intelligence')}
            </Button>
            <Button onClick={() => void refreshIntelligence()}>
              <RefreshCw size={13} /> {t('work.composer.retryIntelligence', 'Verificar novamente')}
            </Button>
          </div>
        </div>
      </div>
    );
  }

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
    <div
      className={`nx-surface-scroll nx-composer-surface nx-composer-workbench ${styles.workbench}`}
    >
      {!gate.canMaterialize && gate.action !== 'WAIT' ? (
        <div className={styles.contextFix}>
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
        </div>
      ) : null}

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
        className={`nx-composer-flow-region ${styles.flowRegion}`}
        data-gate={gate.canMaterialize ? 'ready' : 'destinations-blocked'}
      >
        <ComposerSurface
          project={project}
          destinationGate={gate}
          contextState={state}
          intelligenceReady={Boolean(intelligence?.available)}
          intelligenceLabel={intelligenceLabel}
          memoryCount={memoryCount}
          contextRefreshing={readinessBusy}
          onRefreshContext={() =>
            void prepareContext(
              gate.action === 'PREPARE' ||
                Boolean(readiness?.error?.includes('durable project context is missing')),
            )
          }
          onTransformFlow={(artifact) => void materializeFlow(artifact)}
        />
      </div>
    </div>
  );
};

export { FLOW_DRAFT_KEY };
