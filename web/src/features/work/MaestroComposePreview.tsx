import { useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, ConfirmDialog, InlineAlert } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import type { MaestroComposePreviewResult, Project, WorkPlan } from '../../types';
import { readComposerHandoff } from './ComposerHandoffSummary';

// Composition is a paid, read-only external-model operation. Consent belongs
// to this single request; review and draft replacement are independent steps.
export function MaestroComposePreview({
  project,
  plan,
  onApply,
}: {
  project: Project;
  plan: WorkPlan;
  onApply: (preview: MaestroComposePreviewResult) => void;
}) {
  const { t } = useTranslation();
  const artifactId = plan.structured_facts?.['nexus.source_artifact_id'] || '';
  const [consent, setConsent] = useState(false);
  const [loading, setLoading] = useState(false);
  const [preview, setPreview] = useState<MaestroComposePreviewResult | null>(null);
  const [error, setError] = useState('');
  const [confirmOpen, setConfirmOpen] = useState(false);
  const requestId = useRef(0);

  useEffect(() => {
    requestId.current += 1;
    setConsent(false);
    setLoading(false);
    setPreview(null);
    setError('');
    setConfirmOpen(false);
    return () => {
      // Invalidate any paid request still in flight after switching plans/unmount.
      requestId.current += 1;
    };
  }, [project.id, plan.id, artifactId, plan.current_revision]);

  const generate = async () => {
    if (!consent || !artifactId || loading) return;
    const current = ++requestId.current;
    setLoading(true);
    setError('');
    setPreview(null);
    setConsent(false);
    try {
      const result = await nexusApi.previewMaestroComposition(project.id, artifactId);
      if (requestId.current === current) setPreview(result);
    } catch (reason) {
      if (requestId.current === current) {
        setError(
          reason instanceof Error
            ? reason.message
            : t('flow.maestroComposeError', 'Não foi possível consultar a composição do Maestro.'),
        );
      }
    } finally {
      if (requestId.current === current) setLoading(false);
    }
  };

  const handoffRaw = plan.structured_facts?.['composer.handoff.v1'];
  const handoff = readComposerHandoff(plan);
  const sourceRevision = plan.structured_facts?.['nexus.source_artifact_revision'];
  const matchesRevision = !sourceRevision || sourceRevision === String(preview?.artifact.version);
  const matchesHandoff =
    !handoffRaw ||
    (handoff?.artifact_id === preview?.artifact.id &&
      handoff?.artifact_version === preview?.artifact.version &&
      typeof handoff?.artifact_hash === 'string' &&
      handoff.artifact_hash.toLowerCase() === preview?.artifact.sha256.toLowerCase());
  const provenanceMatches = Boolean(
    preview?.artifact.id === artifactId && matchesRevision && matchesHandoff,
  );
  const hasTasks = Boolean(preview?.tasks?.length);
  const validPlan = preview?.validation.valid === true;
  const canApply = provenanceMatches && validPlan && hasTasks && preview?.executable === false;

  return (
    <Card>
      <details>
        <summary>{t('flow.maestroComposeTitle', 'Composição semântica · Maestro')}</summary>
        <p className="nx-muted-copy">
          {t(
            'flow.maestroComposeDescription',
            'Gerar uma proposta de etapas a partir do PromptArtifact finalizado. Codex é usado somente para planejar em modo read-only. Nenhum Agent será iniciado.',
          )}
        </p>
        {!artifactId ? (
          <InlineAlert
            tone="warning"
            title={t('flow.maestroComposeNoArtifact', 'PromptArtifact necessário')}
          >
            {t(
              'flow.maestroComposeUseComposer',
              'Finalize um artefato no Composer e envie-o para o Flow primeiro.',
            )}
          </InlineAlert>
        ) : (
          <>
            <label>
              <input
                type="checkbox"
                checked={consent}
                onChange={(event) => setConsent(event.target.checked)}
              />{' '}
              {t(
                'flow.maestroComposeConsent',
                'Autorizo uma chamada do Codex para planejamento; ela pode consumir créditos do meu provider.',
              )}
            </label>
            <div>
              <Button disabled={!consent || loading} onClick={() => void generate()}>
                {loading
                  ? t('flow.maestroComposeLoading', 'Compondo etapas…')
                  : t('flow.maestroComposeGenerate', 'Gerar prévia no Maestro')}
              </Button>
            </div>
          </>
        )}
        {error && (
          <InlineAlert
            tone="danger"
            title={t('flow.maestroComposeFailed', 'Composição indisponível')}
          >
            {error}
          </InlineAlert>
        )}
        {preview && !provenanceMatches && (
          <InlineAlert
            tone="danger"
            title={t('flow.maestroComposeMismatchTitle', 'Proveniência divergente')}
          >
            {t(
              'flow.maestroComposeMismatch',
              'A composição não corresponde à revisão aprovada do Composer. Gere uma nova prévia antes de substituir o rascunho.',
            )}
          </InlineAlert>
        )}
        {preview && (
          <div aria-live="polite">
            <p>
              <strong>
                {preview.validation.valid
                  ? t('flow.maestroComposeReady', 'Plano proposto · requer revisão')
                  : t('flow.maestroComposeBlocked', 'Plano rejeitado pelo validador')}
              </strong>
            </p>
            <ol>
              {(preview.tasks || []).map((task) => (
                <li key={task.id}>
                  <strong>{task.title}</strong> — {task.objective}
                  {task.dependsOn.length > 0 && (
                    <small>
                      {' '}
                      · {t('flow.dependencies', 'Depende de')}: {task.dependsOn.join(', ')}
                    </small>
                  )}
                </li>
              ))}
            </ol>
            {preview.validation.blockers?.map((blocker, index) => (
              <p key={index} role="alert">
                {blocker.code || 'BLOCKED'}: {blocker.message}
              </p>
            ))}
            <p className="nx-muted-copy">
              {t(
                'flow.maestroComposeReviewOnly',
                'A proposta não foi executada. Importá-la substituirá o rascunho atual; revise papéis, provedores, dependências e critérios antes de salvar.',
              )}
            </p>
            <Button disabled={!canApply} onClick={() => setConfirmOpen(true)}>
              {t('flow.maestroComposeApply', 'Usar etapas como rascunho')}
            </Button>
          </div>
        )}
      </details>
      <ConfirmDialog
        open={confirmOpen}
        title={t('flow.maestroComposeReplaceTitle', 'Substituir o rascunho do Flow?')}
        description={t(
          'flow.maestroComposeReplaceDescription',
          'As etapas atuais serão substituídas localmente pelas etapas do Maestro. Nenhuma tarefa será iniciada; salve e revise o Flow antes de executar.',
        )}
        confirmLabel={t('flow.maestroComposeReplaceConfirm', 'Substituir rascunho')}
        onCancel={() => setConfirmOpen(false)}
        onConfirm={() => {
          setConfirmOpen(false);
          if (preview && canApply) onApply(preview);
        }}
      />
    </Card>
  );
}
