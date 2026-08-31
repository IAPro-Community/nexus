export type WorkspaceLauncherAction = 'NEW_AGENT' | 'NEW_TERMINAL' | 'OPEN_COMPOSER' | 'OPEN_ACTIVE_FLOW_RUN';

export const workspaceLauncherActions: Array<{ action: WorkspaceLauncherAction; label: string }> = [
  { action: 'NEW_AGENT', label: 'Nova sessão de Agent' },
  { action: 'NEW_TERMINAL', label: 'Novo terminal' },
  { action: 'OPEN_COMPOSER', label: 'Abrir Composer' },
  { action: 'OPEN_ACTIVE_FLOW_RUN', label: 'Abrir Flow Run ativo' },
];

export function launcherActionTarget(action: WorkspaceLauncherAction): string {
  switch (action) {
    case 'NEW_AGENT': return 'agents';
    // Until the native Project Shell surface is introduced, route this to the
    // existing runtime launcher instead of opening an unsupported surface.
    case 'NEW_TERMINAL': return 'legacy-runtimes';
    case 'OPEN_COMPOSER': return 'work';
    case 'OPEN_ACTIVE_FLOW_RUN': return 'legacy-runtimes';
  }
}
