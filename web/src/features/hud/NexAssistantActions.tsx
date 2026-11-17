import React from 'react';
import { nexus } from '../../nexus/api';
import type { MissionSchedule, WorkPlan } from '../../types';
import styles from './NexAssistantPanel.module.scss';

interface NexAssistantActionsProps {
  active: boolean;
  projectId?: string;
  onResult: (message: string) => void;
}

/** Explicit Nexus capabilities, never instructions silently executed by an LLM. */
export const NexAssistantActions: React.FC<NexAssistantActionsProps> = ({
  active,
  projectId,
  onResult,
}) => {
  const [busy, setBusy] = React.useState(false);
  const [plans, setPlans] = React.useState<WorkPlan[]>([]);
  const [schedules, setSchedules] = React.useState<MissionSchedule[]>([]);
  const [scheduleOpen, setScheduleOpen] = React.useState(false);
  const [planId, setPlanId] = React.useState('');
  const [when, setWhen] = React.useState('');
  const [confirmPlan, setConfirmPlan] = React.useState(false);
  const [confirmCancel, setConfirmCancel] = React.useState<string | null>(null);
  const busyRef = React.useRef(false);

  React.useEffect(() => {
    setPlans([]);
    setSchedules([]);
    setScheduleOpen(false);
    setPlanId('');
    setWhen('');
    setConfirmPlan(false);
  }, [projectId]);

  if (!active) return null;

  const perform = async (run: () => Promise<void>) => {
    if (busyRef.current) return;
    busyRef.current = true;
    setBusy(true);
    try {
      await run();
    } catch {
      onResult(
        'Não consegui concluir essa consulta ou operação. Verifique a conexão e a área correspondente no Nexus.',
      );
    } finally {
      busyRef.current = false;
      setBusy(false);
    }
  };

  const resumeProject = () =>
    perform(async () => {
      if (!projectId) return;
      const [events, sessions] = await Promise.all([
        nexus.listProjectEvents(projectId, 20),
        nexus.listComposerSessions(projectId),
      ]);
      const recent = [...events].sort((a, b) => b.timestamp.localeCompare(a.timestamp))[0];
      const composer = [...sessions].sort((a, b) => b.updated_at.localeCompare(a.updated_at))[0];
      const lines = ['Retomada baseada nos registros disponíveis do Nexus:'];
      if (recent) {
        lines.push(
          `Última atividade (${new Date(recent.timestamp).toLocaleString()}): ${recent.summary.slice(0, 250)}.`,
        );
      } else {
        lines.push('Não há eventos recentes registrados para esse projeto.');
      }
      if (composer) {
        lines.push(`Composer mais recente: ${composer.title} — estado ${composer.state}.`);
      } else {
        lines.push('Não há sessões do Composer disponíveis no histórico consultado.');
      }
      lines.push(
        'Para continuar, abra a atividade ou a sessão correspondente no Nexus. Não iniciei nenhuma execução.',
      );
      onResult(lines.join('\n'));
    });

  const diagnose = () =>
    perform(async () => {
      const report = await nexus.getSystemDoctor();
      const problems = report.checks.filter(
        (check) => check.status === 'FAIL' || check.status === 'WARN',
      );
      if (!problems.length) {
        onResult(
          'O diagnóstico atual não retornou verificações FAIL ou WARN. Isso não substitui a homologação das plataformas.',
        );
        return;
      }
      const details = problems
        .slice(0, 4)
        .map(
          (check) =>
            `[${check.status}] ${check.summary}${check.remediation ? ` — ${check.remediation}` : ''}`,
        );
      onResult(`Diagnóstico do Nexus (${problems.length} avisos/falhas):\n${details.join('\n')}`);
    });

  const scan = () =>
    perform(async () => {
      if (!projectId) return;
      const result = await nexus.requestProjectIntelligenceScan(projectId);
      onResult(
        `Análise estática solicitada para o projeto atual. Estado inicial: ${result.state}. ID: ${result.id}. Consulte Inteligência do Projeto para acompanhar; o resultado ainda não foi avaliado pelo Nex.`,
      );
    });

  const refreshResources = () =>
    perform(async () => {
      const result = await nexus.refreshResources();
      onResult(
        `Atualização de recursos solicitada ao Nexus. ${result.accounts.length} conta(s) retornadas. Cotas não verificadas permanecem desconhecidas.`,
      );
    });

  const openSchedules = () =>
    perform(async () => {
      if (!projectId) return;
      const [allPlans, allSchedules] = await Promise.all([
        nexus.getPlans(projectId),
        nexus.getSchedules(projectId),
      ]);
      setPlans(allPlans.filter((plan) => plan.status === 'READY' && plan.current_revision > 0));
      setSchedules(allSchedules);
      setScheduleOpen(true);
      setConfirmPlan(false);
    });

  const schedule = () =>
    perform(async () => {
      if (!planId || !when || !projectId) return;
      const whenDate = new Date(when);
      if (!Number.isFinite(whenDate.getTime()) || whenDate.getTime() <= Date.now()) {
        onResult('Selecione uma data e hora futura válida antes de agendar.');
        return;
      }
      const result = await nexus.schedulePlan(planId, 'AT', {
        scheduledFor: whenDate.toISOString(),
      });
      setSchedules((current) => [...current, result]);
      setConfirmPlan(false);
      onResult(
        `Agendamento registrado no scheduler do Nexus: ${result.id}, estado ${result.status}. A execução futura continua sujeita aos controles e verificações do plano.`,
      );
    });

  const cancelSchedule = (scheduleId: string) =>
    perform(async () => {
      await nexus.cancelSchedule(scheduleId);
      setSchedules((current) =>
        current.map((row) =>
          row.id === scheduleId ? { ...row, status: 'CANCELED' as const } : row,
        ),
      );
      setConfirmCancel(null);
      onResult(`Solicitação de cancelamento registrada para o agendamento ${scheduleId}.`);
    });

  return (
    <div className={styles.actions} aria-label="Ações explícitas do Nex">
      <div className={styles.actionGrid}>
        <button
          type="button"
          className={styles.send}
          disabled={busy || !projectId}
          onClick={resumeProject}
        >
          Retomar projeto
        </button>
        <button type="button" className={styles.send} disabled={busy} onClick={diagnose}>
          Diagnosticar Nexus
        </button>
        <button type="button" className={styles.send} disabled={busy || !projectId} onClick={scan}>
          Analisar projeto
        </button>
        <button type="button" className={styles.send} disabled={busy} onClick={refreshResources}>
          Atualizar recursos
        </button>
        <button
          type="button"
          className={styles.send}
          disabled={busy || !projectId}
          onClick={openSchedules}
        >
          Agendamentos
        </button>
      </div>
      {scheduleOpen && (
        <div className={styles.scheduleBox}>
          <strong>Agendamentos existentes</strong>
          {schedules.length === 0 && <span>Nenhum agendamento registrado neste projeto.</span>}
          {schedules.map((row) => (
            <div className={styles.scheduleItem} key={row.id}>
              <span>
                {row.mode} · {row.status}
              </span>
              {row.scheduled_for && <small>{new Date(row.scheduled_for).toLocaleString()}</small>}
              {row.status === 'PENDING' &&
                (confirmCancel === row.id ? (
                  <button
                    type="button"
                    className={styles.send}
                    disabled={busy}
                    onClick={() => cancelSchedule(row.id)}
                  >
                    Confirmar cancelamento
                  </button>
                ) : (
                  <button
                    type="button"
                    className={styles.send}
                    disabled={busy}
                    onClick={() => setConfirmCancel(row.id)}
                  >
                    Cancelar
                  </button>
                ))}
            </div>
          ))}
          <strong>Agendar plano pronto</strong>
          {plans.length === 0 ? (
            <span>
              Nenhum plano READY com revisão disponível. Prepare e aprove um plano no Composer/Flow.
            </span>
          ) : (
            <>
              <label className={styles.setting}>
                Plano
                <select
                  className={styles.input}
                  value={planId}
                  onChange={(event) => {
                    setPlanId(event.target.value);
                    setConfirmPlan(false);
                  }}
                >
                  <option value="">Selecione...</option>
                  {plans.map((plan) => (
                    <option value={plan.id} key={plan.id}>
                      {plan.title}
                    </option>
                  ))}
                </select>
              </label>
              <label className={styles.setting}>
                Quando
                <input
                  className={styles.input}
                  type="datetime-local"
                  value={when}
                  onChange={(event) => {
                    setWhen(event.target.value);
                    setConfirmPlan(false);
                  }}
                />
              </label>
              {confirmPlan ? (
                <div className={styles.scheduleBox}>
                  <span>
                    Confirma agendar a execução deste plano na data selecionada? O Nexus aplicará os
                    gates de execução.
                  </span>
                  <button type="button" className={styles.send} disabled={busy} onClick={schedule}>
                    Confirmar agendamento
                  </button>
                  <button
                    type="button"
                    className={styles.send}
                    onClick={() => setConfirmPlan(false)}
                  >
                    Voltar
                  </button>
                </div>
              ) : (
                <button
                  type="button"
                  className={styles.send}
                  disabled={busy || !planId || !when}
                  onClick={() => setConfirmPlan(true)}
                >
                  Revisar agendamento
                </button>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
};
