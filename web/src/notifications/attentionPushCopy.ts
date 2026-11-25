export type AttentionPushReason =
  'QUESTION' | 'APPROVAL' | 'AUTH' | 'ERROR' | 'TASK_COMPLETED' | string;

function truncate(text: string, max = 160): string {
  const clean = text.replace(/\s+/g, ' ').trim();
  if (clean.length <= max) return clean;
  return `${clean.slice(0, max - 1)}…`;
}

export function formatAttentionPushBody(input: {
  reason: AttentionPushReason;
  context: string;
  promptKind?: string;
  agentName?: string;
  projectName?: string;
  /** Cross-project or hidden-tab copy includes project name explicitly. */
  rich?: boolean;
}): string {
  const context = truncate(input.context || '');
  if (!context) return '';

  const agent = (input.agentName || '').trim();
  const project = (input.projectName || '').trim();
  const who = input.rich && project && agent ? `${project} · ${agent}` : agent || project;

  const lead =
    input.reason === 'ERROR'
      ? who
        ? `${who} reportou um erro`
        : 'Um agente reportou um erro'
      : input.reason === 'TASK_COMPLETED'
        ? who
          ? `${who} concluiu uma tarefa`
          : 'Um agente concluiu uma tarefa'
        : input.reason === 'APPROVAL'
          ? who
            ? `${who} pede aprovação`
            : 'Um agente pede aprovação'
          : input.reason === 'AUTH'
            ? who
              ? `${who} aguarda autenticação ou permissão`
              : 'Um agente aguarda autenticação ou permissão'
            : input.promptKind === 'yn'
              ? who
                ? `${who} pede confirmação (Sim/Não)`
                : 'Um agente pede confirmação (Sim/Não)'
              : who
                ? `${who} espera sua resposta`
                : 'Um agente espera sua resposta';

  return `${lead}: ${context} — Abra o Nexus e responda no terminal.`;
}

/** Browser push is only useful when the tab is not already in front. */
export function shouldSendBrowserAttentionPush(opts: {
  permission: NotificationPermission | string;
  documentHidden?: boolean;
  reason: AttentionPushReason;
  context: string;
  promptKind?: string;
  notificationsEnabled?: boolean;
}): boolean {
  if (opts.notificationsEnabled === false) return false;
  if (opts.permission !== 'granted') return false;
  if (opts.documentHidden === false) return false;
  if (
    opts.reason !== 'QUESTION' &&
    opts.reason !== 'APPROVAL' &&
    opts.reason !== 'AUTH' &&
    opts.reason !== 'ERROR' &&
    opts.reason !== 'TASK_COMPLETED'
  ) {
    return false;
  }
  const context = (opts.context || '').trim();
  if (!context) return false;
  if (opts.promptKind === 'none') return false;
  if (/o agente requer atenção/i.test(context) || /atenção necessária no terminal/i.test(context)) {
    return false;
  }
  return true;
}
