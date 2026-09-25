import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import type { ExecutionRole, RuntimeExecutionRoleView } from '../types';
import { Button, ConfirmDialog } from '../design-system';
import { nexus } from './api';
import { executionRoleSummary, executionRoleTone } from './executionRoleModel';
import styles from './ExecutionRoleSelector.module.scss';

export const ExecutionRoleSelector: React.FC<{
  runtimeId?: string;
  disabled?: boolean;
  provider?: string;
  onChanged?: (view: RuntimeExecutionRoleView) => void;
}> = ({ runtimeId, disabled = false, provider, onChanged }) => {
  const { t } = useTranslation();
  const [roles, setRoles] = useState<ExecutionRole[]>([]);
  const [view, setView] = useState<RuntimeExecutionRoleView | null>(null);
  const [busy, setBusy] = useState(false);
  const [certifying, setCertifying] = useState(false);
  const [confirmCertification, setConfirmCertification] = useState(false);
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
  const isOpenCode = String(provider || '').toLowerCase() === 'opencode';
  const isReadOnlyRole =
    view?.role?.access_intent === 'READ_ONLY' || view?.role?.access_intent === 'VERIFY_ONLY';
  const canCertify =
    isOpenCode &&
    isReadOnlyRole &&
    view?.binding?.enforcement !== 'ENFORCED' &&
    Boolean(runtimeId);

  const refreshRole = useCallback(
    async (id: string) => {
      const refreshed = await nexus.getRuntimeExecutionRole(id);
      setView(refreshed);
      onChanged?.(refreshed);
      return refreshed;
    },
    [onChanged],
  );

  const changeRole = useCallback(async (roleId: string) => {
    const id = String(runtimeId || '').trim();
    if (!id || busy || roleId === currentRoleId) return;
    const previous = view;
    setBusy(true);
    setError('');
    try {
      const next = await nexus.setRuntimeExecutionRole(id, roleId);
      setView(next);
      onChanged?.(next);

      // OpenCode role projection is local/filesystem-only and does not invoke
      // a model. Failure must not roll back the role: Nexus overlay remains a
      // truthful fallback (for example, a VPN target without shared profile home).
      if (isOpenCode) {
        try {
          await nexus.applyAgentProjection(id);
          await refreshRole(id);
        } catch (projectionError) {
          setError(
            t('terminal.executionRoleProjectionFallback', {
              defaultValue:
                'Role changed. Native OpenCode projection is unavailable; Nexus overlay remains active. {{error}}',
              error:
                projectionError instanceof Error ? projectionError.message : String(projectionError),
            }),
          );
        }
      }
    } catch (err) {
      setView(previous);
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }, [runtimeId, busy, currentRoleId, view, isOpenCode, refreshRole, t, onChanged]);

  useEffect(() => {
    const handleExternalRoleSwitch = (event: Event) => {
      const detail = (event as CustomEvent<{ runtimeId?: string; roleId?: string }>).detail;
      if (!detail?.runtimeId || detail.runtimeId !== runtimeId || !detail.roleId) return;
      void changeRole(detail.roleId);
    };
    window.addEventListener('nexus:execution-role-switch', handleExternalRoleSwitch);
    return () =>
      window.removeEventListener('nexus:execution-role-switch', handleExternalRoleSwitch);
  }, [runtimeId, changeRole]);

  const certifyNativeReadOnly = async () => {
    const id = String(runtimeId || '').trim();
    if (!id || certifying) return;
    setConfirmCertification(false);
    setCertifying(true);
    setError('');
    try {
      const current = view ?? (await nexus.getRuntimeExecutionRole(id));
      if (!current.binding.projection_receipt_id) {
        await nexus.applyAgentProjection(id);
      }
      const afterProjection = await nexus.getRuntimeExecutionRole(id);
      if (afterProjection.binding.activation_strategy !== 'NATIVE_ON_LAUNCH') {
        await nexus.probeAgentProjectionActivation(id, true);
      }
      await nexus.probeAgentProjectionEnforcement(id, true);
      await refreshRole(id);
    } catch (certificationError) {
      setError(
        certificationError instanceof Error
          ? certificationError.message
          : String(certificationError),
      );
    } finally {
      setCertifying(false);
    }
  };

  return (
    <div className={styles.root} title={error || summary}>
      <label className={styles.label} htmlFor={runtimeId ? `execution-role-${runtimeId}` : undefined}>
        {t('terminal.executionRole', { defaultValue: 'Role' })}
      </label>
      <select
        id={runtimeId ? `execution-role-${runtimeId}` : undefined}
        className={styles.select}
        data-execution-role-selector="true"
        data-runtime-id={runtimeId || ''}
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
      {canCertify && (
        <Button
          size="sm"
          tone="warning"
          className={styles.certify}
          disabled={busy || certifying}
          onClick={() => setConfirmCertification(true)}
        >
          {certifying
            ? t('terminal.executionRoleCertifying', { defaultValue: 'Certifying…' })
            : t('terminal.executionRoleCertify', { defaultValue: 'Certify' })}
        </Button>
      )}
      <span className={styles.status} role="status" aria-live="polite">
        {error ||
          (busy
            ? t('common.saving', { defaultValue: 'Saving' })
            : certifying
              ? t('terminal.executionRoleCertifyingStatus', {
                  defaultValue: 'Running OpenCode activation and permission enforcement probes',
                })
              : summary)}
      </span>
      <ConfirmDialog
        open={confirmCertification}
        tone="warning"
        title={t('terminal.executionRoleCertifyTitle', {
          defaultValue: 'Certify native read-only enforcement?',
        })}
        description={t('terminal.executionRoleCertifyDescription', {
          defaultValue:
            'Nexus will run OpenCode with the projected agent, then attempt a controlled file write that must be denied. These probes invoke the configured model and may consume quota/cost.',
        })}
        confirmLabel={t('terminal.executionRoleCertifyConfirm', {
          defaultValue: 'Run certification',
        })}
        onConfirm={() => void certifyNativeReadOnly()}
        onCancel={() => setConfirmCertification(false)}
      />
    </div>
  );
};
