import { useId, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Camera } from 'lucide-react';
import {
  VISUAL_DIFF_DEFAULTS,
  buildVisualDiffRequest,
  formatDiffRatio,
  formatPixelCount,
  parseRegionInput,
  visualDiffTone,
  visualDiffVerdictKey,
  type VisualDiffRequest,
  type VisualDiffResult,
} from './visualDiffModel';
import styles from './VisualDiffPanel.module.scss';

export type VisualDiffPanelProps = {
  /** Active browser session id; absent means the panel renders idle. */
  sessionId?: string;
  /** Transport owned by the surface (auth + route). Returns the server payload. */
  run: (request: VisualDiffRequest) => Promise<VisualDiffResult>;
};

/**
 * Deterministic screenshot comparison UI (capability visual-diff). The panel
 * declares region/thresholds explicitly — the server never invents them — and
 * mirrors the measured verdict. Mode: server recheck (capture → settle →
 * capture), which never touches the page or types into it.
 */
export const VisualDiffPanel: React.FC<VisualDiffPanelProps> = ({ sessionId, run }) => {
  const { t, i18n } = useTranslation();
  const locale = i18n.language || 'en';

  const [delayMs, setDelayMs] = useState<number>(VISUAL_DIFF_DEFAULTS.recheckDelayMs);
  const [pixelThreshold, setPixelThreshold] = useState<number>(VISUAL_DIFF_DEFAULTS.pixelThreshold);
  const [maxDiffRatio, setMaxDiffRatio] = useState<number>(VISUAL_DIFF_DEFAULTS.maxDiffRatio);
  const [regionText, setRegionText] = useState('');
  const [regionError, setRegionError] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [result, setResult] = useState<VisualDiffResult | null>(null);

  const delayId = useId();
  const thresholdId = useId();
  const ratioId = useId();
  const regionId = useId();

  const onSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!sessionId || busy) return;
    const region = parseRegionInput(regionText);
    if (regionText.trim() && !region) {
      setRegionError(true);
      return;
    }
    setRegionError(false);
    setError('');
    setBusy(true);
    const request = buildVisualDiffRequest(
      { kind: 'recheck', delayMs },
      pixelThreshold,
      maxDiffRatio,
    );
    const body = region ? { ...request, region } : request;
    run(body)
      .then((next) => {
        setResult(next);
      })
      .catch((err: unknown) => {
        setError(err instanceof Error ? err.message : String(err));
      })
      .finally(() => {
        setBusy(false);
      });
  };

  const tone = visualDiffTone(result);

  return (
    <div className={styles.panel}>
      <form className={styles.form} onSubmit={onSubmit}>
        <div className={styles.fieldRow}>
          <label className={styles.field} htmlFor={delayId}>
            <span>{t('browserVisualDiff.delayLabel')}</span>
            <input
              id={delayId}
              type="number"
              min={0}
              max={5000}
              step={100}
              value={delayMs}
              onChange={(event) => setDelayMs(Number(event.target.value) || 0)}
              disabled={busy || !sessionId}
            />
          </label>
          <label className={styles.field} htmlFor={thresholdId}>
            <span>{t('browserVisualDiff.pixelThresholdLabel')}</span>
            <input
              id={thresholdId}
              type="number"
              min={0}
              max={255}
              step={1}
              value={pixelThreshold}
              onChange={(event) => setPixelThreshold(Number(event.target.value) || 0)}
              disabled={busy || !sessionId}
            />
          </label>
          <label className={styles.field} htmlFor={ratioId}>
            <span>{t('browserVisualDiff.maxDiffRatioLabel')}</span>
            <input
              id={ratioId}
              type="number"
              min={0}
              max={1}
              step={0.005}
              value={maxDiffRatio}
              onChange={(event) => setMaxDiffRatio(Number(event.target.value) || 0)}
              disabled={busy || !sessionId}
            />
          </label>
        </div>
        <label className={styles.field} htmlFor={regionId}>
          <span>{t('browserVisualDiff.regionLabel')}</span>
          <input
            id={regionId}
            type="text"
            inputMode="numeric"
            placeholder="x, y, width, height"
            value={regionText}
            aria-invalid={regionError}
            onChange={(event) => {
              setRegionText(event.target.value);
              setRegionError(false);
            }}
            disabled={busy || !sessionId}
          />
          {regionError && (
            <span className={styles.fieldError} role="alert">
              {t('browserVisualDiff.regionInvalid')}
            </span>
          )}
        </label>
        <button type="submit" className={styles.runButton} disabled={!sessionId || busy}>
          <Camera size={14} aria-hidden="true" />
          <span>{busy ? t('browserVisualDiff.running') : t('browserVisualDiff.run')}</span>
        </button>
      </form>

      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}

      {result && (
        <div className={styles.result} data-tone={tone} role="status">
          <p className={styles.verdict}>{t(visualDiffVerdictKey(result))}</p>
          <dl className={styles.metrics}>
            <div className={styles.metric}>
              <dt>{t('browserVisualDiff.metricDiffRatio')}</dt>
              <dd>{formatDiffRatio(result.diff_ratio, locale)}</dd>
            </div>
            <div className={styles.metric}>
              <dt>{t('browserVisualDiff.metricDifferentPixels')}</dt>
              <dd>{formatPixelCount(result.different_pixels, locale)}</dd>
            </div>
            <div className={styles.metric}>
              <dt>{t('browserVisualDiff.metricComparedPixels')}</dt>
              <dd>{formatPixelCount(result.compared_pixels, locale)}</dd>
            </div>
          </dl>
          {result.size_mismatch_note && <p className={styles.note}>{result.size_mismatch_note}</p>}
          {result.artifact_b64 && (
            <img
              className={styles.artifact}
              src={`data:image/png;base64,${result.artifact_b64}`}
              alt={t('browserVisualDiff.artifactAlt')}
              loading="lazy"
            />
          )}
        </div>
      )}

      {!sessionId && <p className={styles.idle}>{t('browserVisualDiff.idle')}</p>}
    </div>
  );
};
