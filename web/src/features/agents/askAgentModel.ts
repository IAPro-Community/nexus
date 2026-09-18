export interface AskAgentAction {
  label: 'Prepare Task' | 'Start & Send Task';
  startIfNeeded: boolean;
}

const active = new Set(['WORKING', 'WAITING', 'APPROVAL', 'HANDOFF', 'STARTING', 'RECOVERING']);

export function askActionForStatus(status: string): AskAgentAction {
  return active.has(String(status || '').toUpperCase())
    ? { label: 'Prepare Task', startIfNeeded: false }
    : { label: 'Start & Send Task', startIfNeeded: true };
}
