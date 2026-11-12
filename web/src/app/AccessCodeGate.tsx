import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { pairWithAccessCode } from '../api';
import styles from './AccessCodeGate.module.scss';

export interface AccessCodeGateProps {
  /** Called with the new CSRF token once the device is paired. */
  onPaired: (csrfToken?: string) => void;
}

/**
 * Remote pairing form for devices that opened the tunnel URL without the QR
 * bootstrap fragment. The short code is minted by the main Nexus device in
 * Settings → Remote Access and inherits the invitation's single-use, TTL,
 * host-binding and brute-force protections on the server.
 */
export const AccessCodeGate: React.FC<AccessCodeGateProps> = ({ onPaired }) => {
  const { t } = useTranslation();
  const [code, setCode] = useState('');
  const [error, setError] = useState('');
  const [pairing, setPairing] = useState(false);

  const submit = (event: React.FormEvent) => {
    event.preventDefault();
    const trimmed = code.trim();
    if (pairing || !trimmed) return;
    setPairing(true);
    setError('');
    void (async () => {
      try {
        const session = await pairWithAccessCode(trimmed);
        if (session?.authenticated) {
          onPaired(session.csrf_token);
          return;
        }
        setError(
          t('auth.accessCodeInvalid', {
            defaultValue:
              'Código inválido ou expirado. Códigos são de uso único e revogados após falhas repetidas — gere um novo convite no Nexus principal.',
          }),
        );
      } catch {
        setError(
          t('auth.accessCodeInvalid', {
            defaultValue:
              'Código inválido ou expirado. Códigos são de uso único e revogados após falhas repetidas — gere um novo convite no Nexus principal.',
          }),
        );
      } finally {
        setPairing(false);
      }
    })();
  };

  return (
    <form className={styles.form} onSubmit={submit}>
      <label className={styles.label} htmlFor="access-code-input">
        {t('auth.accessCodeTitle', { defaultValue: 'Parear por Código de Acesso' })}
      </label>
      <p className={styles.hint}>
        {t('auth.accessCodeHint', {
          defaultValue: 'Digite o código de acesso exibido ao lado do QR code no Nexus principal.',
        })}
      </p>
      <input
        id="access-code-input"
        className={styles.input}
        type="text"
        name="access-code"
        autoComplete="one-time-code"
        autoCapitalize="characters"
        autoCorrect="off"
        spellCheck={false}
        maxLength={12}
        placeholder={t('auth.accessCodePlaceholder', { defaultValue: 'Código de 8 caracteres' })}
        value={code}
        onChange={(event) => setCode(event.target.value)}
        disabled={pairing}
      />
      {error && (
        <p className={styles.error} role="alert">
          {error}
        </p>
      )}
      <button
        type="submit"
        className="nx-button"
        data-tone="brand"
        disabled={pairing || !code.trim()}
      >
        {pairing
          ? t('auth.accessCodeWorking', { defaultValue: 'Pareando…' })
          : t('auth.accessCodeSubmit', { defaultValue: 'Parear Dispositivo' })}
      </button>
    </form>
  );
};
