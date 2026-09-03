import { sanitizeAttentionText } from '../components/attentionText';

export function notificationTitle(projectName: string | undefined, reason: string, dynamicTitle?: string): string {
  const project = projectName?.trim() || 'Projeto';
  const dynamic = sanitizeAttentionText(dynamicTitle, '');
  const unsafe = /\b(?:transcript|credentials?|password|token|secret|authorization|bearer|api[-_ ]?key)\b/i.test(dynamic);
  if (dynamic && !unsafe) return `[Nexus | ${project}] ${dynamic}`;
  if (reason === 'QUESTION') return `[Nexus | ${project}] ❓ Pergunta Requer Atenção`;
  if (reason === 'APPROVAL') return `[Nexus | ${project}] ⚠️ Aprovação Necessária`;
  if (reason === 'TASK_COMPLETED') return `[Nexus | ${project}] ✅ Tarefa Concluída!`;
  return `[Nexus | ${project}] Notificação`;
}
