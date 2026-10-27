import React from 'react';
import { Card } from '../../design-system';
import type { WorkPlan } from '../../types';

export interface ComposerHandoffContract {
  version: number;
  session_id: string;
  artifact_id: string;
  artifact_version: number;
  artifact_hash: string;
  goal: string;
  approved_answers?: Record<string, string>;
  assumptions?: Array<{ value?: string; status?: string }>;
  recommendations?: Array<{ value?: string; status?: string }>;
  unresolved?: Array<{ question?: string; severity?: string; status?: string }>;
  acceptance?: string[];
  risks?: string[];
  route?: { result?: string; reasons?: string[]; signals?: string[] };
}

/** Treat stored JSON as untrusted input. Never convert malformed payloads into
 * an implied approval, and never fetch or execute anything from the panel. */
export function readComposerHandoff(plan: WorkPlan | null): ComposerHandoffContract | null {
  const raw = plan?.structured_facts?.['composer.handoff.v1'];
  if (!raw) return null;
  try {
    const value: unknown = JSON.parse(raw);
    if (!value || typeof value !== 'object') return null;
    const contract = value as Partial<ComposerHandoffContract>;
    if (
      contract.version !== 1 ||
      typeof contract.artifact_id !== 'string' ||
      typeof contract.goal !== 'string'
    )
      return null;
    return contract as ComposerHandoffContract;
  } catch {
    return null;
  }
}

export function countComposerBlockingDecisions(handoff: ComposerHandoffContract): number {
  return (Array.isArray(handoff.unresolved) ? handoff.unresolved : []).filter(
    (item) => item?.status === 'OPEN' && item?.severity === 'BLOCKING',
  ).length;
}

/** Shared between Flow Draft and Mission Run: always a projection of WorkPlan
 * facts, never a second mutable copy of the original Composer session. */
export const ComposerHandoffSummary: React.FC<{
  plan: WorkPlan | null;
  execution?: boolean;
}> = ({ plan, execution = false }) => {
  const handoff = readComposerHandoff(plan);
  if (!handoff) return null;
  const pending = countComposerBlockingDecisions(handoff);
  const approved = Object.entries(handoff.approved_answers || {});
  const assumptions = (Array.isArray(handoff.assumptions) ? handoff.assumptions : []).filter(
    (item) => Boolean(item?.value),
  );
  const recommendations = (
    Array.isArray(handoff.recommendations) ? handoff.recommendations : []
  ).filter((item) => Boolean(item?.value) && item?.status !== 'DISMISSED');
  const routeExplanation = (
    Array.isArray(handoff.route?.reasons) ? handoff.route.reasons : []
  ).join(' · ');

  return (
    <Card>
      <section aria-label="Contrato de planejamento do Composer">
        <strong>Plano originado no Composer · versão {handoff.artifact_version}</strong>
        <p>{handoff.goal}</p>
        {routeExplanation && <p>Estratégia sugerida: {routeExplanation}</p>}
        {pending > 0 && (
          <p role="alert">
            {pending} decisão(ões) bloqueante(s) ainda não resolvida(s). A execução exige revisão do
            Composer e nova aprovação do plano.
          </p>
        )}
        <details>
          <summary>Ver decisões, hipóteses e critérios do planejamento</summary>
          {approved.length > 0 && (
            <div>
              <strong>Decisões respondidas pelo usuário</strong>
              <ul>
                {approved.map(([key, value]) => (
                  <li key={key}>
                    {key}: {value}
                  </li>
                ))}
              </ul>
            </div>
          )}
          {assumptions.length > 0 && (
            <div>
              <strong>Hipóteses — não são fatos confirmados</strong>
              <ul>
                {assumptions.map((item, index) => (
                  <li key={index}>{item.value}</li>
                ))}
              </ul>
            </div>
          )}
          {recommendations.length > 0 && (
            <div>
              <strong>Opções propostas pela IA — não são decisões aprovadas</strong>
              <ul>
                {recommendations.map((item, index) => (
                  <li key={index}>{item.value}</li>
                ))}
              </ul>
            </div>
          )}
          {(handoff.acceptance || []).length > 0 && (
            <div>
              <strong>Condições de aceite</strong>
              <ul>
                {handoff.acceptance?.map((item, index) => (
                  <li key={index}>{item}</li>
                ))}
              </ul>
            </div>
          )}
          <p>
            {execution
              ? 'Esta Mission utiliza uma revisão congelada; ajustes posteriores devem gerar novo plano.'
              : 'Alterar o Flow não altera silenciosamente as decisões aprovadas no Composer.'}
          </p>
        </details>
      </section>
    </Card>
  );
};
