import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AlertCircle, HelpCircle, Send, Terminal, X, Check } from 'lucide-react';
import { api } from '../api';
import type { RuntimeSession } from '../types';
import { sanitizeAttentionText } from './attentionText';
import { isHonestNeedsUser } from '../app/documentTitle';

export interface AttentionNotificationCardProps {
  runtime: RuntimeSession;
  onFocusRuntime: (runtimeId: string) => void;
  onDismiss: (runtimeId: string) => void;
}

export function shouldRenderAttentionCard(runtime: RuntimeSession): boolean {
  return isHonestNeedsUser(runtime);
}

/** Pure UI contract: Yes/No only for honest yn waits. */
export function attentionCardActions(runtime: RuntimeSession): {
  showYesNo: boolean;
  showTextInput: boolean;
  showOpenTerminal: boolean;
} {
  if (!shouldRenderAttentionCard(runtime)) {
    return { showYesNo: false, showTextInput: false, showOpenTerminal: false };
  }
  const promptKind = runtime.prompt_kind || 'free_text';
  return {
    showYesNo: promptKind === 'yn',
    showTextInput: promptKind === 'yn' || promptKind === 'free_text' || promptKind === 'choice',
    showOpenTerminal: true,
  };
}

export const AttentionNotificationCard: React.FC<AttentionNotificationCardProps> = ({
  runtime,
  onFocusRuntime,
  onDismiss,
}) => {
  const { t } = useTranslation();
  const [inputText, setInputText] = useState('');
  const [sending, setSending] = useState(false);

  const projName = sanitizeAttentionText(runtime.project_name, t('rail.projects', 'Project'));
  const promptKind = runtime.prompt_kind || 'free_text';
  const isApproval = runtime.attention_reason === 'APPROVAL';
  const contextText = sanitizeAttentionText(runtime.attention_context, '');
  const runtimeLabel =
    runtime.runtime_id.length > 16 ? `${runtime.runtime_id.slice(0, 16)}…` : runtime.runtime_id;
  const actions = attentionCardActions(runtime);

  if (!contextText) {
    return null;
  }

  const handleSend = async (text: string) => {
    if (!text.trim() || sending) return;
    setSending(true);
    try {
      await api.respondRuntime(runtime.runtime_id, text);
      setInputText('');
      onDismiss(runtime.runtime_id);
    } catch (err) {
      console.error('[AttentionNotification] Falha ao enviar resposta:', err);
    } finally {
      setSending(false);
    }
  };

  const dialogLabel = isApproval
    ? t('attention.approvalRequired', {
        project: projName,
        defaultValue: `Approval required in ${projName}`,
      })
    : t('attention.agentWaiting', {
        project: projName,
        defaultValue: `Agent waiting in ${projName}`,
      });

  return (
    <article
      className="nx-uui-notification"
      role="alertdialog"
      aria-live="assertive"
      aria-label={dialogLabel}
    >
      <div className="nx-uui-notification__container">
        <div
          className={`nx-uui-notification__icon-wrap nx-uui-notification__icon-wrap--${isApproval ? 'rose' : 'amber'}`}
        >
          {isApproval ? (
            <AlertCircle className="nx-uui-notification__icon text-rose-400" size={20} />
          ) : (
            <HelpCircle className="nx-uui-notification__icon text-amber-400" size={20} />
          )}
        </div>

        <div className="nx-uui-notification__content">
          <div className="nx-uui-notification__header">
            <div className="nx-uui-notification__badges">
              <span className="nx-uui-badge nx-uui-badge--brand">{projName}</span>
              <span className={`nx-uui-badge nx-uui-badge--${isApproval ? 'rose' : 'amber'}`}>
                {isApproval
                  ? t('attention.approval', 'Approval')
                  : promptKind === 'yn'
                    ? t('attention.yesNo', 'Yes/No')
                    : promptKind === 'choice'
                      ? t('attention.choice', 'Choice')
                      : t('attention.response', 'Response')}
              </span>
              <span className="nx-uui-notification__id" title={runtime.runtime_id}>
                {runtimeLabel}
              </span>
            </div>
            <button
              type="button"
              className="nx-uui-notification__close"
              onClick={() => onDismiss(runtime.runtime_id)}
              aria-label={t('attention.dismiss', 'Dismiss notification')}
              title={t('attention.dismissTitle', 'Dismiss')}
            >
              <X size={15} />
            </button>
          </div>

          <p className="nx-uui-notification__text">
            {promptKind === 'yn'
              ? t('attention.confirmationPending', {
                  context: contextText,
                  defaultValue: `Confirmation pending: ${contextText}`,
                })
              : promptKind === 'choice'
                ? t('attention.choicePending', {
                    context: contextText,
                    defaultValue: `Choice pending: ${contextText}`,
                  })
                : contextText}
          </p>
          <p
            className="nx-uui-notification__hint"
            style={{ margin: '4px 0 0', fontSize: '0.786rem', color: 'var(--nx-muted)' }}
          >
            {t('attention.hint', 'Reply here or open the agent terminal.')}
          </p>

          <div className="nx-uui-notification__footer">
            <div className="nx-uui-notification__quick-actions">
              {actions.showYesNo ? (
                <>
                  <button
                    type="button"
                    className="nx-uui-btn nx-uui-btn--primary"
                    onClick={() => handleSend('y')}
                    disabled={sending}
                  >
                    <Check size={13} />
                    <span>{t('attention.yes', 'Yes (y)')}</span>
                  </button>
                  <button
                    type="button"
                    className="nx-uui-btn nx-uui-btn--secondary"
                    onClick={() => handleSend('n')}
                    disabled={sending}
                  >
                    <span>{t('attention.no', 'No (n)')}</span>
                  </button>
                </>
              ) : null}
              {actions.showOpenTerminal ? (
                <button
                  type="button"
                  className="nx-uui-btn nx-uui-btn--tertiary"
                  onClick={() => onFocusRuntime(runtime.runtime_id)}
                >
                  <Terminal size={13} />
                  <span>{t('attention.viewTerminal', 'View Terminal')}</span>
                </button>
              ) : null}
            </div>

            {actions.showTextInput ? (
              <form
                className="nx-uui-notification__form"
                onSubmit={(e) => {
                  e.preventDefault();
                  void handleSend(inputText);
                }}
              >
                <input
                  type="text"
                  className="nx-uui-input"
                  placeholder={
                    promptKind === 'choice'
                      ? t('attention.placeholderChoice', 'Type choice number…')
                      : t('attention.placeholderText', 'Type response…')
                  }
                  value={inputText}
                  onChange={(e) => setInputText(e.target.value)}
                  disabled={sending}
                />
                <button
                  type="submit"
                  className="nx-uui-btn nx-uui-btn--icon-submit"
                  disabled={!inputText.trim() || sending}
                  aria-label={t('attention.sendResponse', 'Send response')}
                  title={t('attention.sendResponse', 'Send response')}
                >
                  <Send size={13} />
                </button>
              </form>
            ) : null}
          </div>
        </div>
      </div>
    </article>
  );
};
