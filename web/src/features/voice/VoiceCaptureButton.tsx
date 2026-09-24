import React, { useCallback, useEffect, useRef, useState } from 'react';
import { LoaderCircle, Mic, Square } from 'lucide-react';
import { nexus } from '../../nexus/api';
import {
  extensionForVoiceMime,
  startVoiceCapture,
  supportsVoiceCapture,
  type VoiceCaptureSession,
} from './voiceCapture';
import type { VoiceSettingsView, VoiceTranscribeResult } from './voiceModel';
import styles from './VoiceCaptureButton.module.scss';

type VoiceContext = 'terminal' | 'nexus';

let cachedSettings: VoiceSettingsView | null = null;
let settingsPromise: Promise<VoiceSettingsView> | null = null;

async function loadSettings(force = false) {
  if (!force && cachedSettings) return cachedSettings;
  if (!force && settingsPromise) return settingsPromise;
  settingsPromise = nexus
    .getVoiceSettings()
    .then((settings) => {
      cachedSettings = settings;
      return settings;
    })
    .finally(() => {
      settingsPromise = null;
    });
  return settingsPromise;
}

export const invalidateVoiceSettingsCache = () => {
  cachedSettings = null;
  settingsPromise = null;
};

export const VoiceCaptureButton: React.FC<{
  context: VoiceContext;
  disabled?: boolean;
  showLabel?: boolean;
  className?: string;
  onTranscript: (text: string, result: VoiceTranscribeResult) => void | Promise<void>;
}> = ({ context, disabled = false, showLabel = false, className = '', onTranscript }) => {
  const [settings, setSettings] = useState<VoiceSettingsView | null>(cachedSettings);
  const [state, setState] = useState<'idle' | 'recording' | 'transcribing' | 'error'>('idle');
  const [detail, setDetail] = useState('');
  const captureRef = useRef<VoiceCaptureSession | null>(null);
  const timerRef = useRef<number | undefined>();

  const refreshSettings = useCallback((force = false) => {
    void loadSettings(force)
      .then(setSettings)
      .catch((error) => {
        setDetail(error instanceof Error ? error.message : String(error));
        setState('error');
      });
  }, []);

  useEffect(() => {
    refreshSettings(false);
    const changed = () => {
      invalidateVoiceSettingsCache();
      refreshSettings(true);
    };
    window.addEventListener('nexus:voice-settings', changed);
    return () => {
      window.removeEventListener('nexus:voice-settings', changed);
      if (timerRef.current !== undefined) window.clearTimeout(timerRef.current);
      captureRef.current?.cancel();
      captureRef.current = null;
    };
  }, [refreshSettings]);

  const contextEnabled =
    settings?.enabled &&
    (context === 'terminal' ? settings.terminal_enabled : settings.nexus_control_enabled);
  const hasConfiguredProvider = Boolean(
    settings?.providers.some((provider) => provider.enabled && provider.configured),
  );

  const finish = useCallback(async () => {
    const capture = captureRef.current;
    if (!capture) return;
    captureRef.current = null;
    if (timerRef.current !== undefined) {
      window.clearTimeout(timerRef.current);
      timerRef.current = undefined;
    }
    setState('transcribing');
    setDetail('Transcrevendo…');
    try {
      const recording = await capture.stop();
      if (recording.blob.size === 0) throw new Error('Nenhum áudio foi capturado.');
      const form = new FormData();
      form.append(
        'audio',
        recording.blob,
        `nexus-voice.${extensionForVoiceMime(recording.mimeType)}`,
      );
      form.append('duration_ms', String(recording.durationMs));
      if (settings?.language) form.append('language', settings.language);
      const result = await nexus.transcribeVoice(form);
      const text = result.text.trim();
      if (!text) throw new Error('O provedor não retornou texto.');
      await onTranscript(text, result);
      setDetail(`${result.provider} · ${result.model}`);
      setState('idle');
    } catch (error) {
      setDetail(error instanceof Error ? error.message : String(error));
      setState('error');
    }
  }, [onTranscript, settings?.language]);

  const start = useCallback(async () => {
    if (!settings || !contextEnabled || !hasConfiguredProvider || disabled) return;
    if (!supportsVoiceCapture()) {
      setDetail('Microfone indisponível neste navegador/WebView.');
      setState('error');
      return;
    }
    setDetail('Solicitando microfone…');
    try {
      const capture = await startVoiceCapture();
      captureRef.current = capture;
      setState('recording');
      setDetail('Ouvindo… clique novamente para enviar.');
      timerRef.current = window.setTimeout(
        () => {
          void finish();
        },
        Math.max(5, settings.max_capture_seconds || 60) * 1000,
      );
    } catch (error) {
      setDetail(error instanceof Error ? error.message : String(error));
      setState('error');
    }
  }, [contextEnabled, disabled, finish, hasConfiguredProvider, settings]);

  if (!settings || !contextEnabled) return null;

  const unavailable = !hasConfiguredProvider || !supportsVoiceCapture();
  const busy = state === 'transcribing';
  const recording = state === 'recording';
  const label = recording
    ? 'Parar e transcrever'
    : busy
      ? 'Transcrevendo voz'
      : context === 'terminal'
        ? 'Ditado por voz no terminal'
        : 'Controle do Nexus por voz';

  return (
    <span className={`${styles.wrap} ${className}`} title={detail || label}>
      <button
        type="button"
        className={styles.button}
        data-state={state}
        aria-label={label}
        aria-pressed={recording}
        disabled={disabled || unavailable || busy}
        onClick={() => {
          if (recording) void finish();
          else void start();
        }}
      >
        {busy ? (
          <LoaderCircle size={14} className={styles.spin} />
        ) : recording ? (
          <Square size={12} />
        ) : (
          <Mic size={14} />
        )}
        {showLabel && <span>{recording ? 'Parar' : busy ? 'Transcrevendo…' : 'Voz'}</span>}
      </button>
      <span className={styles.srStatus} role="status" aria-live="polite">
        {detail}
      </span>
    </span>
  );
};
