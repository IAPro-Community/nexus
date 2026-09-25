import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { ExecutionRole, RuntimeExecutionRoleView } from '../types';
import { nexus } from './api';
import { executionRoleSummary, executionRoleTone } from './executionRoleModel';
import styles from './ExecutionRoleSelector.module.scss';

export const ExecutionRoleSelector: React.FC<{
  runtimeId?: string;
  disabled?: boolean;
  onChanged?: (view: RuntimeExecutionRoleView) => void;
}> = ({ runtimeId, disabled = false, onChanged }) => {
  const { t } = useTranslation();
  const [roles, setRoles] = useState<ExecutionRole[]>([]);
  const [view, setView] = useState<RuntimeExecutionRoleView | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const requestSeq = useRef(0);

  useEffect(() => {
    let alive = true;
    void nexus
      .listExecutionRoles()
      .then((next) => {
        if (alive) setRoles(Array.isArray(next) ? next : []);
      })
      .catch((err) => {
        if (alive) setError(err instanceof Error ? err.message : String(err));
      });
    return () => {
      alive = false;
    };
  }, []);

  useEffect(() => {
    const id = String(runtimeId || '').trim();
    const seq = ++requestSeq.current;
    if (!id) {
      setView(null);
      return;
    }
    setError('');
    void nexus
      .getRuntimeExecutionRole(id)
      .then((next) => {
        if (seq === requestSeq.current) setView(next);
      })
      .catch((err) => {
        if (seq === requestSeq.current) {
          setView(null);
          setError(err instanceof Error ? err.message : String(err));
        }
      });
  }, [runtimeId]);

  const currentRoleId = view?.role?.id || 'build';
  const summary = useMemo(() => executionRoleSummary(view), [view]);
  const tone = executionRoleTone(view);

  const changeRole = async (roleId: string) => {
    const id = String(runtimeId || '').trim();
    if (!id || busy || roleId === currentRoleId) return;
    const previous = view;
    setBusy(true);
    setError('');
    try {
      const next = await nexus.setRuntimeExecutionRole(id, roleId);
      setView(next);
      onChanged?.(next);
    } catch (err) {
      setView(previous);
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className={styles.root} title={error || summary}>
      <label
        className={styles.label}
        htmlFor={runtimeId ? `execution-role-${runtimeId}` : undefined}
      >
        {t('terminal.executionRole', { defaultValue: 'Role' })}
      </label>
      <select
        id={runtimeId ? `execution-role-${runtimeId}` : undefined}
        className={styles.select}
        value={currentRoleId}
        disabled={disabled || busy || !runtimeId || roles.length === 0}
        aria-label={t('terminal.executionRoleAria', { defaultValue: 'Execution role' })}
        onChange={(event) => void changeRole(event.target.value)}
      >
        {roles.map((role) => (
          <option key={role.id} value={role.id}>
            {role.label}
          </option>
        ))}
      </select>
      {view && (
        <span
          className={styles.badge}
          data-tone={tone}
          aria-label={t('terminal.executionRoleEnforcement', {
            defaultValue: 'Role enforcement: {{enforcement}}',
            enforcement: view.binding.enforcement,
          })}
        >
          {String(view.binding.enforcement || 'UNVERIFIED').replaceAll('_', ' ')}
        </span>
      )}
      <span className={styles.status} role="status" aria-live="polite">
        {error || (busy ? t('common.saving', { defaultValue: 'Saving' }) : summary)}
      </span>
    </div>
  );
};
