import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Button, Card, InlineAlert } from '../../design-system';
import { nexusApi } from '../../nexus/api';
import type { MaestroSubmissionSnapshot, MaestroSubmissionStatusRow, Project } from '../../types';
import { asArray } from '../../lib/safeArray';
import styles from './MaestroMissionPanel.module.scss';

const REFRESH_MS = 4000;
const ACTIVE_STATUSES = new Set(['PENDING_WORKER', 'CLAIMED', 'DISPATCHED', 'CANCEL_REQUESTED']);

// Monitor of Maestro-owned durable submissions for this project. It is an
// observation surface only: the Nexus never redispatches, reschedules or
// fabricates state. Maestro unavailability renders a distinct state without
// any local fallback execution.
export function MaestroMissionPanel({ project }: { project: Project }) {
  const { t } = useTranslation();
  const [snapshot, setSnapshot] = useState<MaestroSubmissionSnapshot | null>(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [cancelling, setCancelling] = useState<string | null>(null);
  const [cancelError, setCancelError] = useState('');
  const version = useRef(0);

  const load = useCallback(
    async (quiet = false) => {
      const current = ++version.current;
      if (quiet) setRefreshing(true);
      else setLoading(true);
      if (!quiet) setError('');
      try {
        const result = await nexusApi.listMaestroSubmissions(project.id, { limit: 20 });
        if (version.current === current) setSnapshot(result);
      } catch (reason) {
        if (version.current === current) {
          setSnapshot(null);
          setError(
            reason instanceof Error
              ? reason.message
              : t('flow.maestroMissionUnavailable', 'Maestro indisponível para observação.'),
          );
        }
      } finally {
        if (version.current === current) {
          setLoading(false);
          setRefreshing(false);
        }
      }
    },
    [project.id, t],
  );

  useEffect(() => {
    void load();
    const timer = window.setInterval(() => {
      if (!document.hidden) void load(true);
    }, REFRESH_MS);
    const onVisible = () => {
      if (!document.hidden) void load(true);
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      version.current += 1;
      window.clearInterval(timer);
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [load]);

  const cancel = async (row: MaestroSubmissionStatusRow) => {
    if (!ACTIVE_STATUSES.has(row.status) || cancelling) return;
    setCancelling(row.submissionId);
    setCancelError('');
    try {
      await nexusApi.cancelMaestroSubmission(project.id, row.submissionId);
      await load(true);
    } catch (reason) {
      setCancelError(
        reason instanceof Error
          ? reason.message
          : t('flow.maestroMissionCancelFailed', 'Não foi possível solicitar o cancelamento.'),
      );
    } finally {
      setCancelling(null);
    }
  };

  const rows = asArray<MaestroSubmissionStatusRow>(snapshot?.submissions);

  return (
    <Card className={styles.panel}>
      <details>
        <summary className={styles.summary}>
          {t('flow.maestroMissionTitle', 'Missões do Maestro · fila durável')}
        </summary>
        <p className={styles.description}>
          {t(
            'flow.maestroMissionDescription',
            'Submissões duráveis Maestro-owned deste projeto. O Nexus apenas observa: o despacho e os receipts pertencem ao Maestro.',
          )}
        </p>
        {loading ? (
          <p className={styles.hint}>{t('flow.maestroMissionLoading', 'Carregando submissões…')}</p>
        ) : error ? (
          <InlineAlert
            tone="warning"
            title={t('flow.maestroMissionUnavailableTitle', 'Maestro indisponível')}
          >
            {t(
              'flow.maestroMissionUnavailableText',
              'As submissões duráveis existentes continuam no Maestro. Não há execução alternativa no Nexus.',
            )}
          </InlineAlert>
        ) : rows.length === 0 ? (
          <p className={styles.hint}>
            {t(
              'flow.maestroMissionEmpty',
              'Nenhuma submissão durável ainda. Admita uma missão a partir de um pré-validação pronta.',
            )}
          </p>
        ) : (
          <ul className={styles.rows}>
            {rows.map((row) => (
              <li key={row.submissionId} className={styles.row}>
                <div className={styles.identity}>
                  <strong>{row.submissionId}</strong>
                  <span className={styles.status}>{row.status}</span>
                </div>
                <small className={styles.meta}>
                  {t('flow.maestroMissionTasks', 'Tarefas')}: {row.taskCount} ·{' '}
                  {row.providers.join(', ')}
                </small>
                {ACTIVE_STATUSES.has(row.status) && (
                  <Button
                    className={styles.cancel}
                    disabled={cancelling === row.submissionId}
                    onClick={() => void cancel(row)}
                  >
                    {cancelling === row.submissionId
                      ? t('flow.maestroMissionCancelling', 'Solicitando…')
                      : t('flow.maestroMissionCancel', 'Cancelar')}
                  </Button>
                )}
              </li>
            ))}
          </ul>
        )}
        {cancelError && (
          <InlineAlert
            tone="danger"
            title={t('flow.maestroMissionCancelFailedTitle', 'Cancelamento não confirmado')}
          >
            {t(
              'flow.maestroMissionCancelFailedText',
              'A solicitação foi registrada ou rejeitada pelo Maestro; o estado real permanece na fila.',
            )}
          </InlineAlert>
        )}
        {refreshing && (
          <p className={styles.hint} aria-live="polite">
            {t('flow.maestroMissionRefreshing', 'Atualizando…')}
          </p>
        )}
      </details>
    </Card>
  );
}
