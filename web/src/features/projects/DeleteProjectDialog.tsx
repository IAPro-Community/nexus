import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { TFunction } from 'i18next';
import { toast } from 'sonner';
import { Button, Dialog, InlineAlert } from '../../design-system';
import { asArray, asStringArray } from '../../lib/safeArray';
import {
  NexusAPIError,
  nexus,
  type ProjectDeleteMode,
  type ProjectDeleteResult,
  type ProjectLiveAgentInfo,
} from '../../nexus/api';
import type { Project } from '../../types';
import styles from './DeleteProjectDialog.module.scss';

export interface DeleteProjectDialogProps {
  project: Project | null;
  onClose: () => void;
  onDeleted?: (project: Project, result: ProjectDeleteResult) => void;
  initialBlockedAgents?: ProjectLiveAgentInfo[] | null;
}

export function formatProjectDeleteSummary(
  t: TFunction,
  projectName: string,
  result: ProjectDeleteResult,
): string {
  const detached = asStringArray(result.detached_agents);
  const stopped = asStringArray(result.stopped_agents);
  if (result.mode === 'detach' && detached.length > 0) {
    return t('projectDelete.summaryDetached', {
      name: projectName,
      count: detached.length,
      agents: detached.join(', '),
    });
  }
  if (result.mode === 'stop_agents' && stopped.length > 0) {
    return t('projectDelete.summaryStopped', {
      name: projectName,
      count: stopped.length,
      agents: stopped.join(', '),
    });
  }
  return t('projectDelete.summaryClean', { name: projectName });
}

export const DeleteProjectDialog: React.FC<DeleteProjectDialogProps> = ({
  project,
  onClose,
  onDeleted,
  initialBlockedAgents = null,
}) => {
  const { t } = useTranslation();
  const [busyMode, setBusyMode] = useState<'plain' | ProjectDeleteMode | null>(null);
  const [blockedAgents, setBlockedAgents] = useState<ProjectLiveAgentInfo[] | null>(
    initialBlockedAgents,
  );
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  useEffect(() => {
    setBusyMode(null);
    setBlockedAgents(initialBlockedAgents ?? null);
    setErrorMsg(null);
  }, [project?.id, initialBlockedAgents]);

  if (!project) return null;

  const projectName = project.name || project.id;
  const runDelete = async (mode?: ProjectDeleteMode) => {
    if (busyMode) return;
    setBusyMode(mode ?? 'plain');
    setErrorMsg(null);
    try {
      const result = await nexus.deleteProject(project.id, mode ? { mode } : undefined);
      toast.success(formatProjectDeleteSummary(t, projectName, result));
      onDeleted?.(project, result);
      onClose();
    } catch (error) {
      if (
        !mode &&
        error instanceof NexusAPIError &&
        error.status === 409 &&
        error.payload?.code === 'project_has_live_agents'
      ) {
        const live = asArray<ProjectLiveAgentInfo>(error.payload.live_agents);
        setBlockedAgents(live);
        return;
      }
      const message =
        error instanceof NexusAPIError && error.payload?.error
          ? String(error.payload.error)
          : error instanceof Error
            ? error.message
            : t('rail.deleteError');
      setErrorMsg(message);
    } finally {
      setBusyMode(null);
    }
  };

  const showResolution = blockedAgents !== null;

  return (
    <Dialog open={Boolean(project)} title={t('rail.deleteTitle')} onClose={onClose}>
      <div className={styles.body} data-nex-delete-project={showResolution ? 'blocked' : 'confirm'}>
        {!showResolution ? (
          <>
            <p className={styles.description}>
              {t('rail.deleteDescription', { name: projectName })}
            </p>
            <p className={styles.hint}>{t('projectDelete.diskSafeHint')}</p>
          </>
        ) : (
          <>
            <InlineAlert tone="warning">
              {t('projectDelete.activeAgentsDetected', {
                name: projectName,
                count: blockedAgents.length,
              })}
            </InlineAlert>
            <p className={styles.hint}>{t('projectDelete.cliOriginHint')}</p>
            {blockedAgents.length > 0 && (
              <ul className={styles.agentList} data-nex-delete-live-agents>
                {blockedAgents.map((agent) => (
                  <li key={agent.id || agent.runtime_id} className={styles.agentItem}>
                    <span className={styles.agentName}>{agent.name || agent.id}</span>
                    <span className={styles.agentMeta}>
                      {[agent.provider, agent.runtime_id].filter(Boolean).join(' · ')}
                    </span>
                  </li>
                ))}
              </ul>
            )}
          </>
        )}

        {errorMsg && <InlineAlert tone="danger">{errorMsg}</InlineAlert>}

        <div className={styles.actions}>
          <Button type="button" onClick={onClose} disabled={Boolean(busyMode)}>
            {t('common.cancel')}
          </Button>
          {!showResolution ? (
            <Button
              type="button"
              tone="danger"
              disabled={Boolean(busyMode)}
              data-nex-delete-confirm
              onClick={() => void runDelete()}
            >
              {busyMode === 'plain' ? t('rail.deleting') : t('rail.delete')}
            </Button>
          ) : (
            <>
              <Button
                type="button"
                tone="brand"
                disabled={Boolean(busyMode)}
                data-nex-delete-detach
                onClick={() => void runDelete('detach')}
              >
                {busyMode === 'detach' ? t('rail.deleting') : t('projectDelete.detachBtn')}
              </Button>
              <Button
                type="button"
                tone="danger"
                disabled={Boolean(busyMode)}
                data-nex-delete-stop
                onClick={() => void runDelete('stop_agents')}
              >
                {busyMode === 'stop_agents'
                  ? t('rail.deleting')
                  : t('projectDelete.stopAndDeleteBtn')}
              </Button>
            </>
          )}
        </div>
      </div>
    </Dialog>
  );
};
