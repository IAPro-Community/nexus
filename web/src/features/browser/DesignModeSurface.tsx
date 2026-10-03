import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Crosshair, ExternalLink, Send, Square, Copy } from 'lucide-react';
import { useTranslation } from 'react-i18next';
import { nexus } from '../../nexus/api';
import type { Agent, DesignModeStatus, UIContextArtifact } from '../../types';
import styles from './DesignModeSurface.module.scss';

function artifactText(artifact: UIContextArtifact): string {
  return [
    'Nexus UI Context Artifact',
    `URL: ${artifact.url}`,
    `Selector: ${artifact.selector}`,
    artifact.text ? `Text: ${artifact.text}` : '',
    artifact.source_hints
      ? `Source hints: ${JSON.stringify(artifact.source_hints)}`
      : '',
    `Computed styles: ${JSON.stringify(artifact.computed_styles || {})}`,
    'HTML:',
    artifact.html,
  ]
    .filter(Boolean)
    .join('\n');
}

export const DesignModeSurface: React.FC<{
  projectId: string;
  agents: Agent[];
}> = ({ projectId, agents }) => {
  const { t } = useTranslation();
  const [status, setStatus] = useState<DesignModeStatus | null>(null);
  const [url, setURL] = useState('http://localhost:3000');
  const [selector, setSelector] = useState('');
  const [artifact, setArtifact] = useState<UIContextArtifact | null>(null);
  const [busy, setBusy] = useState(false);
  const [picking, setPicking] = useState(false);
  const [error, setError] = useState('');
  const [feedback, setFeedback] = useState('');
  const [selectedAgentId, setSelectedAgentId] = useState('');

  const selectedAgent = useMemo(
    () => agents.find((agent) => agent.id === selectedAgentId),
    [agents, selectedAgentId],
  );

  useEffect(() => {
    if (!selectedAgentId && agents[0]?.id) setSelectedAgentId(agents[0].id);
  }, [agents, selectedAgentId]);

  useEffect(() => {
    let mounted = true;
    void nexus
      .getDesignModeStatus(projectId)
      .then((next) => {
        if (!mounted) return;
        setStatus(next);
        if (next.url) setURL(next.url);
      })
      .catch((err) => {
        if (mounted) setError(err instanceof Error ? err.message : String(err));
      });
    return () => {
      mounted = false;
    };
  }, [projectId]);

  const inspect = useCallback(
    async (nextSelector: string) => {
      const clean = nextSelector.trim();
      if (!clean) return;
      setBusy(true);
      setError('');
      setFeedback('');
      try {
        const next = await nexus.inspectDesignModeSelector(projectId, clean);
        setSelector(next.selector);
        setArtifact(next);
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setBusy(false);
      }
    },
    [projectId],
  );

  useEffect(() => {
    if (!picking) return;
    let cancelled = false;
    let timer: number | undefined;

    const poll = async () => {
      try {
        const next = await nexus.getDesignModePicker(projectId);
        if (cancelled) return;
        if (next.selector) {
          setPicking(false);
          setSelector(next.selector);
          await inspect(next.selector);
          return;
        }
        if (next.cancelled || !next.active) {
          setPicking(false);
          return;
        }
      } catch (err) {
        if (!cancelled) {
          setPicking(false);
          setError(err instanceof Error ? err.message : String(err));
        }
        return;
      }
      if (!cancelled) timer = window.setTimeout(poll, 450);
    };

    timer = window.setTimeout(poll, 250);
    return () => {
      cancelled = true;
      if (timer !== undefined) window.clearTimeout(timer);
    };
  }, [picking, projectId, inspect]);

  const start = async () => {
    setBusy(true);
    setError('');
    setFeedback('');
    try {
      const next = await nexus.startDesignMode(projectId, url.trim());
      setStatus((current) => ({
        enabled: current?.enabled ?? true,
        active: next.active,
        url: next.url,
      }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const stop = async () => {
    setBusy(true);
    setError('');
    try {
      await nexus.stopDesignMode(projectId);
      setPicking(false);
      setArtifact(null);
      setStatus((current) => ({ enabled: current?.enabled ?? true, active: false }));
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  const beginPicker = async () => {
    setError('');
    setFeedback('');
    try {
      await nexus.beginDesignModePicker(projectId);
      setPicking(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    }
  };

  const copyArtifact = async () => {
    if (!artifact) return;
    try {
      await navigator.clipboard.writeText(artifactText(artifact));
      setFeedback(t('designMode.copied', { defaultValue: 'Context copied.' }));
    } catch {
      setError(t('designMode.copyFailed', { defaultValue: 'Could not copy context.' }));
    }
  };

  const sendToAgent = async () => {
    if (!artifact || !selectedAgent) return;
    setBusy(true);
    setError('');
    setFeedback('');
    try {
      await nexus.askAgent(
        selectedAgent.id,
        [
          'Use this Nexus Design Mode context to inspect the referenced UI element.',
          'Do not infer files that are not supported by source hints; verify before editing.',
          '',
          artifactText(artifact),
        ].join('\n'),
        true,
      );
      setFeedback(
        t('designMode.sent', {
          defaultValue: 'Context sent to {{name}}.',
          name: selectedAgent.name,
        }),
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  };

  if (status && !status.enabled) {
    return (
      <div className={styles.root}>
        <div className={styles.disabled}>
          <h2 className={styles.title}>
            {t('designMode.title', { defaultValue: 'Design Mode' })}
          </h2>
          <p className={styles.help}>
            {t('designMode.disabled', {
              defaultValue:
                'Design Mode is behind the browser_design_mode feature flag. Enable NEXUS_ENABLE_BROWSER_DESIGN_MODE=1 for this 1.1 development branch.',
            })}
          </p>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.root}>
      <header className={styles.header}>
        <div>
          <h2 className={styles.title}>
            {t('designMode.title', { defaultValue: 'Design Mode' })}
          </h2>
          <p className={styles.subtitle}>
            {t('designMode.subtitle', {
              defaultValue:
                'Open an isolated Chromium window, pick a rendered element and turn it into bounded UI context for an Agent.',
            })}
          </p>
        </div>
        <span className={styles.status} data-active={status?.active ? 'true' : 'false'}>
          {status?.active
            ? t('designMode.active', { defaultValue: 'Browser active' })
            : t('designMode.inactive', { defaultValue: 'Browser stopped' })}
        </span>
      </header>

      <section className={styles.panel}>
        <div className={styles.row}>
          <label className={styles.field}>
            <span className={styles.label}>
              {t('designMode.url', { defaultValue: 'Application URL' })}
            </span>
            <input
              className={styles.input}
              value={url}
              onChange={(event) => setURL(event.target.value)}
              placeholder="http://localhost:3000"
              inputMode="url"
              disabled={busy}
            />
          </label>
          <button
            type="button"
            className="nx-button"
            data-tone="brand"
            onClick={() => void start()}
            disabled={busy || !url.trim()}
          >
            <ExternalLink size={14} />
            <span>{t('designMode.start', { defaultValue: 'Open browser' })}</span>
          </button>
          <button
            type="button"
            className="nx-button"
            onClick={() => void stop()}
            disabled={busy || !status?.active}
          >
            <Square size={13} />
            <span>{t('designMode.stop', { defaultValue: 'Stop' })}</span>
          </button>
        </div>

        <div className={styles.row}>
          <label className={styles.field}>
            <span className={styles.label}>
              {t('designMode.selector', { defaultValue: 'CSS selector' })}
            </span>
            <input
              className={styles.input}
              value={selector}
              onChange={(event) => setSelector(event.target.value)}
              placeholder="[data-testid=&quot;save&quot;]"
              disabled={!status?.active || busy}
            />
          </label>
          <button
            type="button"
            className="nx-button"
            data-tone={picking ? 'brand' : undefined}
            onClick={() => void beginPicker()}
            disabled={!status?.active || busy || picking}
          >
            <Crosshair size={14} />
            <span>
              {picking
                ? t('designMode.picking', { defaultValue: 'Click an element…' })
                : t('designMode.pick', { defaultValue: 'Pick element' })}
            </span>
          </button>
          <button
            type="button"
            className="nx-button"
            onClick={() => void inspect(selector)}
            disabled={!status?.active || busy || !selector.trim()}
          >
            {t('designMode.inspect', { defaultValue: 'Inspect' })}
          </button>
        </div>

        {picking && (
          <p className={styles.help}>
            {t('designMode.pickerHelp', {
              defaultValue:
                'Move over the Chromium window and click the element. Press Esc in Chromium to cancel.',
            })}
          </p>
        )}
        {error && <p className={styles.error}>{error}</p>}
        {feedback && <p className={styles.feedback}>{feedback}</p>}
      </section>

      {artifact && (
        <section className={styles.panel}>
          <div className={styles.row}>
            <strong>{artifact.selector}</strong>
            <button type="button" className="nx-button" onClick={() => void copyArtifact()}>
              <Copy size={13} />
              <span>{t('designMode.copy', { defaultValue: 'Copy context' })}</span>
            </button>
            <label className={styles.field}>
              <span className={styles.label}>
                {t('designMode.agent', { defaultValue: 'Send to Agent' })}
              </span>
              <select
                className={styles.select}
                value={selectedAgentId}
                onChange={(event) => setSelectedAgentId(event.target.value)}
              >
                <option value="">
                  {t('designMode.selectAgent', { defaultValue: 'Select an Agent' })}
                </option>
                {agents.map((agent) => (
                  <option key={agent.id} value={agent.id}>
                    {agent.name}
                  </option>
                ))}
              </select>
            </label>
            <button
              type="button"
              className="nx-button"
              data-tone="brand"
              onClick={() => void sendToAgent()}
              disabled={busy || !selectedAgent}
            >
              <Send size={13} />
              <span>{t('designMode.send', { defaultValue: 'Send context' })}</span>
            </button>
          </div>

          <div className={styles.artifactGrid}>
            <div>
              {artifact.screenshot_ref ? (
                <img
                  className={styles.preview}
                  src={nexus.designModeScreenshotURL(projectId, artifact.screenshot_ref)}
                  alt={t('designMode.screenshotAlt', {
                    defaultValue: 'Captured UI element',
                  })}
                />
              ) : (
                <p className={styles.help}>
                  {t('designMode.noScreenshot', { defaultValue: 'No screenshot was captured.' })}
                </p>
              )}
              <dl className={styles.metaList}>
                <dt>URL</dt>
                <dd>{artifact.url}</dd>
                <dt>{t('designMode.text', { defaultValue: 'Text' })}</dt>
                <dd>{artifact.text || '—'}</dd>
                <dt>{t('designMode.bounds', { defaultValue: 'Bounds' })}</dt>
                <dd>
                  {Math.round(artifact.bounds.width)} × {Math.round(artifact.bounds.height)}
                </dd>
              </dl>
            </div>

            <div>
              <span className={styles.label}>HTML</span>
              <pre className={styles.code}>{artifact.html}</pre>
              <span className={styles.label}>
                {t('designMode.computedStyles', { defaultValue: 'Computed styles' })}
              </span>
              <pre className={styles.code}>
                {JSON.stringify(artifact.computed_styles || {}, null, 2)}
              </pre>
              {artifact.source_hints && Object.keys(artifact.source_hints).length > 0 && (
                <>
                  <span className={styles.label}>
                    {t('designMode.sourceHints', { defaultValue: 'Source hints' })}
                  </span>
                  <pre className={styles.code}>
                    {JSON.stringify(artifact.source_hints, null, 2)}
                  </pre>
                </>
              )}
            </div>
          </div>
        </section>
      )}
    </div>
  );
};
