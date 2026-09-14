import React, { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Badge, Button, InlineAlert } from '../../design-system';
import { nexus } from '../../nexus/api';
import type { IntelligenceMemory } from '../../types';
import { asArray } from '../../lib/safeArray';
import styles from './IntelligenceMemoriesPanel.module.scss';

export const IntelligenceMemoriesPanel: React.FC<{ projectId?: string }> = ({ projectId }) => {
  const { t } = useTranslation();
  const [memories, setMemories] = useState<IntelligenceMemory[]>([]);
  const [error, setError] = useState('');
  const [busyId, setBusyId] = useState('');

  const refresh = useCallback(async () => {
    try {
      setError('');
      setMemories(asArray(await nexus.listIntelligenceMemories(projectId)));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  }, [projectId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const forget = async (memory: IntelligenceMemory) => {
    setBusyId(memory.id);
    try {
      await nexus.deleteIntelligenceMemory(memory.scope, memory.id, memory.project_id || projectId);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusyId('');
    }
  };

  const install = memories.filter((m) => m.scope === 'INSTALL');
  const project = memories.filter((m) => m.scope === 'PROJECT');

  return (
    <div className={styles.root}>
      <div className={styles.header}>
        <strong>{t('settings.memoriesTitle', 'Intelligence memories')}</strong>
        <p>
          {t(
            'settings.memoriesIntro',
            'Habits Nexus remembers so Composer and Flow skip repeated questions. Delete any time.',
          )}
        </p>
      </div>
      {error ? (
        <InlineAlert tone="danger" title={t('common.error', 'Error')}>
          {error}
        </InlineAlert>
      ) : null}
      <MemoryGroup
        title={t('settings.memoriesInstall', 'This installation')}
        items={install}
        busyId={busyId}
        onForget={forget}
        empty={t('settings.memoriesEmpty', 'No memories yet.')}
      />
      <MemoryGroup
        title={t('settings.memoriesProject', 'This project')}
        items={project}
        busyId={busyId}
        onForget={forget}
        empty={t('settings.memoriesEmpty', 'No memories yet.')}
      />
    </div>
  );
};

const MemoryGroup: React.FC<{
  title: string;
  items: IntelligenceMemory[];
  busyId: string;
  empty: string;
  onForget: (memory: IntelligenceMemory) => void;
}> = ({ title, items, busyId, empty, onForget }) => {
  const { t } = useTranslation();
  return (
    <section className={styles.group} aria-label={title}>
      <h3>{title}</h3>
      {items.length === 0 ? (
        <p className={styles.empty}>{empty}</p>
      ) : (
        <ul className={styles.list}>
          {items.map((memory) => (
            <li key={memory.id} className={styles.item}>
              <div>
                <div className={styles.keyRow}>
                  <strong>{memory.key}</strong>
                  {memory.source ? <Badge tone="default">{memory.source}</Badge> : null}
                </div>
                <p>{memory.value}</p>
              </div>
              <Button size="sm" disabled={busyId === memory.id} onClick={() => onForget(memory)}>
                {t('settings.memoriesDelete', 'Forget')}
              </Button>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
};
