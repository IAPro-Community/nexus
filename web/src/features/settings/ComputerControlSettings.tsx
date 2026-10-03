import React, { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Camera,
  Focus,
  Keyboard,
  Monitor,
  MousePointer2,
  RefreshCw,
  Search,
} from 'lucide-react';
import {
  Badge,
  Button,
  Card,
  InlineAlert,
  Select,
  Spinner,
  Switch,
  Textarea,
} from '../../design-system';
import { nexus } from '../../nexus/api';
import type {
  AutonomyContract,
  ComputerActionKind,
  ComputerAppInfo,
  ComputerElement,
  ComputerSnapshot,
  ComputerUseStatus,
  ComputerWindowInfo,
} from '../../types';
import { defaultMissionAutonomyContract } from '../work/missionAutonomyModel';
import { useTranslation } from 'react-i18next';
import styles from './ComputerControlSettings.module.scss';

type ComputerPermissionKey =
  | 'allow_screen_capture'
  | 'allow_keyboard_input'
  | 'allow_pointer_input'
  | 'allow_window_management';

function windowKey(window: ComputerWindowInfo): string {
  if (window.id) return `id:${window.id}`;
  return `index:${window.index ?? 0}`;
}

function elementKey(element: ComputerElement): string {
  return element.id ? `id:${element.id}` : `index:${element.index}`;
}

export const ComputerControlSettings: React.FC<{ projectId?: string }> = ({ projectId }) => {
  const { t } = useTranslation();
  const [contract, setContract] = useState<AutonomyContract>(() => defaultMissionAutonomyContract());
  const [status, setStatus] = useState<ComputerUseStatus | null>(null);
  const [statusLoading, setStatusLoading] = useState(false);
  const [appsLoading, setAppsLoading] = useState(false);
  const [windowsLoading, setWindowsLoading] = useState(false);
  const [operationLoading, setOperationLoading] = useState('');
  const [error, setError] = useState('');
  const [apps, setApps] = useState<ComputerAppInfo[]>([]);
  const [selectedApp, setSelectedApp] = useState('');
  const [windows, setWindows] = useState<ComputerWindowInfo[]>([]);
  const [selectedWindowKey, setSelectedWindowKey] = useState('');
  const [snapshot, setSnapshot] = useState<ComputerSnapshot | null>(null);
  const [selectedElementKey, setSelectedElementKey] = useState('');
  const [textDraft, setTextDraft] = useState('');

  const selectedWindow = useMemo(
    () => windows.find((item) => windowKey(item) === selectedWindowKey),
    [selectedWindowKey, windows],
  );
  const selectedElement = useMemo(
    () => (snapshot?.elements || []).find((item) => elementKey(item) === selectedElementKey),
    [selectedElementKey, snapshot],
  );

  const ready =
    Boolean(projectId) &&
    Boolean(status?.feature_enabled) &&
    Boolean(status?.permissions.available) &&
    contract.allow_computer_use;

  const resetTargetState = useCallback(() => {
    setApps([]);
    setSelectedApp('');
    setWindows([]);
    setSelectedWindowKey('');
    setSnapshot(null);
    setSelectedElementKey('');
    setTextDraft('');
  }, []);

  const refreshStatus = useCallback(async () => {
    if (!projectId) {
      setStatus(null);
      return;
    }
    try {
      setStatusLoading(true);
      setError('');
      setStatus(await nexus.getProjectComputerStatus(projectId));
    } catch (cause) {
      setStatus(null);
      setError(
        cause instanceof Error
          ? cause.message
          : t('settings.computer.errors.status', {
              defaultValue: 'Could not read computer-control status.',
            }),
      );
    } finally {
      setStatusLoading(false);
    }
  }, [projectId, t]);

  useEffect(() => {
    setContract(defaultMissionAutonomyContract());
    resetTargetState();
    void refreshStatus();
  }, [projectId, refreshStatus, resetTargetState]);

  const setComputerUse = (enabled: boolean) => {
    setContract((current) => ({
      ...current,
      allow_computer_use: enabled,
      ...(!enabled
        ? {
            allow_screen_capture: false,
            allow_keyboard_input: false,
            allow_pointer_input: false,
            allow_window_management: false,
          }
        : {}),
    }));
    if (!enabled) resetTargetState();
  };

  const setPermission = (key: ComputerPermissionKey, enabled: boolean) => {
    setContract((current) => ({ ...current, [key]: enabled }));
  };

  const discoverApps = async () => {
    if (!projectId) return;
    try {
      setAppsLoading(true);
      setError('');
      const result = await nexus.listProjectComputerApps(projectId, contract);
      const nextApps = result.apps || [];
      setApps(nextApps);
      if (!nextApps.some((item) => item.name === selectedApp)) {
        setSelectedApp('');
        setWindows([]);
        setSelectedWindowKey('');
        setSnapshot(null);
        setSelectedElementKey('');
      }
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setAppsLoading(false);
    }
  };

  const discoverWindows = async () => {
    if (!projectId || !selectedApp) return;
    try {
      setWindowsLoading(true);
      setError('');
      const result = await nexus.listProjectComputerWindows(projectId, contract, selectedApp);
      const nextWindows = result.windows || [];
      setWindows(nextWindows);
      setSelectedWindowKey(nextWindows.length === 1 ? windowKey(nextWindows[0]) : '');
      setSnapshot(null);
      setSelectedElementKey('');
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setWindowsLoading(false);
    }
  };

  const targetWindowFields = () => {
    if (!selectedWindow) return {};
    if (selectedWindow.id) return { window_id: selectedWindow.id };
    if (typeof selectedWindow.index === 'number') return { window_index: selectedWindow.index };
    return {};
  };

  const captureSnapshot = async () => {
    if (!projectId || !selectedApp || !selectedWindow) return;
    try {
      setOperationLoading('snapshot');
      setError('');
      const next = await nexus.captureProjectComputerSnapshot(projectId, contract, {
        app: selectedApp,
        ...targetWindowFields(),
        screenshot: contract.allow_screen_capture,
        restore_window: false,
      });
      setSnapshot(next);
      setSelectedElementKey('');
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setOperationLoading('');
    }
  };

  const performAction = async (
    kind: ComputerActionKind,
    extra: Partial<Parameters<typeof nexus.performProjectComputerAction>[2]> = {},
  ) => {
    if (!projectId || !selectedApp || !selectedWindow) return;
    try {
      setOperationLoading(kind);
      setError('');
      const result = await nexus.performProjectComputerAction(projectId, contract, {
        kind,
        app: selectedApp,
        ...targetWindowFields(),
        ...extra,
      });
      setSnapshot(result.snapshot);
      setSelectedElementKey('');
    } catch (cause) {
      setError(cause instanceof Error ? cause.message : String(cause));
    } finally {
      setOperationLoading('');
    }
  };

  const elementTarget = () => {
    if (!selectedElement) return {};
    if (selectedElement.id) return { element_id: selectedElement.id };
    return { element_index: selectedElement.index };
  };

  if (!projectId) {
    return (
      <Card className={styles.fullWidth}>
        <InlineAlert tone="info">
          {t('settings.computer.projectRequired', {
            defaultValue: 'Select a Project before configuring Computer Control.',
          })}
        </InlineAlert>
      </Card>
    );
  }

  const capabilities = status?.capabilities;
  const permissionAvailable = Boolean(status?.permissions.available);
  const featureEnabled = Boolean(status?.feature_enabled);
  const elements = (snapshot?.elements || []).slice(0, 100);
  const selectedElementIsWritable = Boolean(selectedElement && !selectedElement.sensitive);

  return (
    <>
      <Card className={styles.fullWidth}>
        <div className="nx-settings-card__title">
          <Monitor size={17} />
          <div>
            <strong>
              {t('settings.computer.title', { defaultValue: 'Computer Control' })}
            </strong>
            <small>
              {t('settings.computer.description', {
                defaultValue:
                  'Observe and control the active machine through structured, permission-gated actions.',
              })}
            </small>
          </div>
        </div>

        {statusLoading && !status ? (
          <Spinner
            label={t('settings.computer.loading', { defaultValue: 'Checking native support…' })}
          />
        ) : (
          <div className={styles.statusRow}>
            <Badge tone={featureEnabled ? 'success' : 'warning'}>
              {featureEnabled
                ? t('settings.computer.featureOn', { defaultValue: 'feature enabled' })
                : t('settings.computer.featureOff', { defaultValue: 'feature disabled' })}
            </Badge>
            <Badge tone={permissionAvailable ? 'success' : 'warning'}>
              {permissionAvailable
                ? t('settings.computer.providerReady', { defaultValue: 'native provider ready' })
                : t('settings.computer.providerUnavailable', {
                    defaultValue: 'native provider unavailable',
                  })}
            </Badge>
            {capabilities?.provider && (
              <span className={styles.statusMeta}>
                {capabilities.provider} · {capabilities.platform}
              </span>
            )}
            <Button tone="ghost" size="sm" onClick={() => void refreshStatus()} disabled={statusLoading}>
              <RefreshCw size={14} />
              {t('common.refresh')}
            </Button>
          </div>
        )}

        {!featureEnabled && status && (
          <InlineAlert tone="warning">
            {t('settings.computer.featureDisabledHint', {
              defaultValue:
                'Computer Control is disabled by feature flag. No observation or action will be sent to the native helper.',
            })}
          </InlineAlert>
        )}
        {status?.permissions.reason && (
          <InlineAlert tone={permissionAvailable ? 'info' : 'warning'}>
            {status.permissions.reason}
          </InlineAlert>
        )}
        {error && <InlineAlert tone="danger">{error}</InlineAlert>}
      </Card>

      <Card className={styles.fullWidth}>
        <div className="nx-settings-card__title">
          <Focus size={17} />
          <div>
            <strong>
              {t('settings.computer.permissionsTitle', { defaultValue: 'Session permissions' })}
            </strong>
            <small>
              {t('settings.computer.permissionsDescription', {
                defaultValue:
                  'These permissions are sent as the same AutonomyContract used by Nexus Missions.',
              })}
            </small>
          </div>
        </div>

        <Switch
          checked={contract.allow_computer_use}
          onChange={setComputerUse}
          label={t('settings.computer.allowUse', { defaultValue: 'Allow Computer Control' })}
          description={t('settings.computer.allowUseDescription', {
            defaultValue: 'Required before Nexus may inspect applications or windows.',
          })}
        />

        {contract.allow_computer_use && (
          <div className={styles.permissionGrid}>
            <Switch
              checked={contract.allow_screen_capture}
              onChange={(checked) => setPermission('allow_screen_capture', checked)}
              label={t('settings.computer.allowScreenshot', {
                defaultValue: 'Allow screen capture',
              })}
              description={t('settings.computer.allowScreenshotDescription', {
                defaultValue: 'Allows PNG capture of the selected window.',
              })}
            />
            <Switch
              checked={contract.allow_keyboard_input}
              onChange={(checked) => setPermission('allow_keyboard_input', checked)}
              label={t('settings.computer.allowKeyboard', {
                defaultValue: 'Allow keyboard/text input',
              })}
              description={t('settings.computer.allowKeyboardDescription', {
                defaultValue: 'Required for typing or replacing values in non-sensitive controls.',
              })}
            />
            <Switch
              checked={contract.allow_pointer_input}
              onChange={(checked) => setPermission('allow_pointer_input', checked)}
              label={t('settings.computer.allowPointer', {
                defaultValue: 'Allow pointer input',
              })}
              description={t('settings.computer.allowPointerDescription', {
                defaultValue: 'Required for clicks, scrolling and dragging.',
              })}
            />
            <Switch
              checked={contract.allow_window_management}
              onChange={(checked) => setPermission('allow_window_management', checked)}
              label={t('settings.computer.allowWindowManagement', {
                defaultValue: 'Allow window management',
              })}
              description={t('settings.computer.allowWindowManagementDescription', {
                defaultValue: 'Allows focus, move and resize operations on the selected window.',
              })}
            />
          </div>
        )}
      </Card>

      <Card className={styles.fullWidth}>
        <div className="nx-settings-card__title">
          <Search size={17} />
          <div>
            <strong>
              {t('settings.computer.targetTitle', { defaultValue: 'Target application' })}
            </strong>
            <small>
              {t('settings.computer.targetDescription', {
                defaultValue: 'Choose an accessible application and one of its top-level windows.',
              })}
            </small>
          </div>
        </div>

        <div className={styles.toolbar}>
          <Button
            onClick={() => void discoverApps()}
            disabled={!ready || appsLoading || operationLoading !== ''}
          >
            <RefreshCw size={14} />
            {appsLoading
              ? t('settings.computer.discoveringApps', { defaultValue: 'Discovering…' })
              : t('settings.computer.discoverApps', { defaultValue: 'Discover apps' })}
          </Button>
          {apps.length > 0 && <Badge tone="info">{apps.length} apps</Badge>}
        </div>

        <div className={styles.targetGrid}>
          <Select
            aria-label={t('settings.computer.app', { defaultValue: 'Application' })}
            value={selectedApp}
            onChange={(value) => {
              setSelectedApp(value);
              setWindows([]);
              setSelectedWindowKey('');
              setSnapshot(null);
              setSelectedElementKey('');
            }}
            disabled={!ready || apps.length === 0}
            placeholder={t('settings.computer.chooseApp', { defaultValue: 'Choose an application' })}
            options={apps.map((app) => ({
              value: app.name,
              label: `${app.name} · PID ${app.pid}`,
            }))}
          />

          <Button
            onClick={() => void discoverWindows()}
            disabled={!ready || !selectedApp || windowsLoading || operationLoading !== ''}
          >
            <Monitor size={14} />
            {windowsLoading
              ? t('settings.computer.loadingWindows', { defaultValue: 'Loading windows…' })
              : t('settings.computer.listWindows', { defaultValue: 'List windows' })}
          </Button>

          <Select
            aria-label={t('settings.computer.window', { defaultValue: 'Window' })}
            value={selectedWindowKey}
            onChange={(value) => {
              setSelectedWindowKey(value);
              setSnapshot(null);
              setSelectedElementKey('');
            }}
            disabled={!ready || windows.length === 0}
            placeholder={t('settings.computer.chooseWindow', { defaultValue: 'Choose a window' })}
            options={windows.map((window) => ({
              value: windowKey(window),
              label: window.title || `Window ${window.index ?? 0}`,
            }))}
          />
        </div>

        {selectedWindow && (
          <div className={styles.windowMeta}>
            <span>{selectedWindow.width}×{selectedWindow.height}</span>
            {typeof selectedWindow.x === 'number' && typeof selectedWindow.y === 'number' && (
              <span>@ {selectedWindow.x},{selectedWindow.y}</span>
            )}
            {selectedWindow.id && <code>{selectedWindow.id}</code>}
          </div>
        )}

        <div className={styles.toolbar}>
          <Button
            tone="brand"
            onClick={() => void captureSnapshot()}
            disabled={!ready || !selectedWindow || operationLoading !== ''}
          >
            <Camera size={14} />
            {operationLoading === 'snapshot'
              ? t('settings.computer.capturing', { defaultValue: 'Capturing…' })
              : t('settings.computer.capture', { defaultValue: 'Capture snapshot' })}
          </Button>
          <Button
            onClick={() => void performAction('FOCUS_WINDOW')}
            disabled={
              !ready ||
              !selectedWindow ||
              !contract.allow_window_management ||
              !capabilities?.window_management ||
              operationLoading !== ''
            }
          >
            <Focus size={14} />
            {t('settings.computer.focusWindow', { defaultValue: 'Focus window' })}
          </Button>
          <Button
            onClick={() => void performAction('SCROLL', { direction: 'UP' })}
            disabled={
              !ready ||
              !selectedWindow ||
              !contract.allow_pointer_input ||
              !capabilities?.scroll ||
              operationLoading !== ''
            }
          >
            ↑ {t('settings.computer.scroll', { defaultValue: 'Scroll' })}
          </Button>
          <Button
            onClick={() => void performAction('SCROLL', { direction: 'DOWN' })}
            disabled={
              !ready ||
              !selectedWindow ||
              !contract.allow_pointer_input ||
              !capabilities?.scroll ||
              operationLoading !== ''
            }
          >
            ↓ {t('settings.computer.scroll', { defaultValue: 'Scroll' })}
          </Button>
        </div>
      </Card>

      {snapshot && (
        <Card className={styles.fullWidth}>
          <div className="nx-settings-card__title">
            <Camera size={17} />
            <div>
              <strong>
                {t('settings.computer.snapshotTitle', { defaultValue: 'Current snapshot' })}
              </strong>
              <small>
                {t('settings.computer.snapshotDescription', {
                  defaultValue:
                    'Accessibility metadata is authoritative for targeting; screenshots are supplemental.',
                })}
              </small>
            </div>
          </div>

          {snapshot.screenshot_ref && (
            <div className={styles.screenshotFrame}>
              <img
                src={nexus.projectComputerScreenshotURL(projectId, snapshot.screenshot_ref)}
                alt={t('settings.computer.screenshotAlt', {
                  defaultValue: 'Selected application window snapshot',
                })}
              />
            </div>
          )}

          <div className={styles.snapshotMeta}>
            <Badge tone="info">
              {(snapshot.elements || []).length}{' '}
              {t('settings.computer.elements', { defaultValue: 'elements' })}
            </Badge>
            <span>{snapshot.window.title}</span>
            <span>{new Date(snapshot.captured_at).toLocaleString()}</span>
          </div>

          <div className={styles.elementActionGrid}>
            <div className={styles.elementColumn}>
              <Select
                aria-label={t('settings.computer.element', { defaultValue: 'Accessibility element' })}
                value={selectedElementKey}
                onChange={setSelectedElementKey}
                placeholder={t('settings.computer.chooseElement', {
                  defaultValue: 'Choose an accessibility element',
                })}
                options={elements.map((element) => ({
                  value: elementKey(element),
                  label: `#${element.index} · ${element.role || 'Element'} · ${element.name || '(unnamed)'}${element.sensitive ? ' · sensitive' : ''}`,
                  disabled: Boolean(element.sensitive),
                }))}
              />
              {(snapshot.elements || []).length > elements.length && (
                <p className={styles.muted}>
                  {t('settings.computer.elementsLimited', {
                    defaultValue:
                      'Showing the first 100 elements. Capture a narrower target when possible.',
                  })}
                </p>
              )}

              {selectedElement && (
                <div className={styles.elementDetail}>
                  <div>
                    <strong>{selectedElement.name || '(unnamed)'}</strong>
                    <Badge tone={selectedElement.sensitive ? 'warning' : 'default'}>
                      {selectedElement.role || 'Element'}
                    </Badge>
                  </div>
                  {selectedElement.value && !selectedElement.sensitive && (
                    <code>{selectedElement.value}</code>
                  )}
                  <span>
                    {selectedElement.width || 0}×{selectedElement.height || 0} @{' '}
                    {selectedElement.x || 0},{selectedElement.y || 0}
                  </span>
                </div>
              )}
            </div>

            <div className={styles.actionColumn}>
              <Button
                onClick={() => void performAction('CLICK', elementTarget())}
                disabled={
                  !ready ||
                  !selectedElement ||
                  !contract.allow_pointer_input ||
                  !capabilities?.click ||
                  operationLoading !== ''
                }
              >
                <MousePointer2 size={14} />
                {t('settings.computer.clickElement', { defaultValue: 'Click element' })}
              </Button>

              <Textarea
                value={textDraft}
                onChange={setTextDraft}
                rows={4}
                maxLength={8192}
                disabled={!selectedElementIsWritable || !contract.allow_keyboard_input}
                placeholder={t('settings.computer.textPlaceholder', {
                  defaultValue: 'Text to type into the selected non-sensitive control',
                })}
              />

              <div className={styles.toolbar}>
                <Button
                  onClick={() =>
                    void performAction('TYPE_TEXT', { ...elementTarget(), text: textDraft })
                  }
                  disabled={
                    !ready ||
                    !selectedElementIsWritable ||
                    !textDraft ||
                    !contract.allow_keyboard_input ||
                    !capabilities?.type_text ||
                    operationLoading !== ''
                  }
                >
                  <Keyboard size={14} />
                  {t('settings.computer.typeText', { defaultValue: 'Type text' })}
                </Button>
                <Button
                  onClick={() =>
                    void performAction('SET_VALUE', { ...elementTarget(), text: textDraft })
                  }
                  disabled={
                    !ready ||
                    !selectedElementIsWritable ||
                    !contract.allow_keyboard_input ||
                    !capabilities?.set_value ||
                    operationLoading !== ''
                  }
                >
                  {t('settings.computer.setValue', { defaultValue: 'Replace value' })}
                </Button>
              </div>
            </div>
          </div>
        </Card>
      )}
    </>
  );
};
