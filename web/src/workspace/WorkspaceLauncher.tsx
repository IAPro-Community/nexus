import React from 'react';
import { Plus } from 'lucide-react';
import { launcherActionTarget, workspaceLauncherActions, type WorkspaceLauncherAction } from './launcherModel';

export const WorkspaceLauncher: React.FC<{ onOpen: (target: string) => void }> = ({ onOpen }) => (
  <div className="nx-workspace-launcher">
    <button type="button" className="nx-workspace-launcher__trigger" aria-label="Novo" title="Novo" onClick={() => onOpen('agents')}>
      <Plus size={13} /> <span>Novo</span>
    </button>
    <div className="nx-workspace-launcher__menu" role="menu">
      {workspaceLauncherActions.map(({ action, label }) => <button key={action} type="button" role="menuitem" onClick={() => onOpen(launcherActionTarget(action as WorkspaceLauncherAction))}>{label}</button>)}
    </div>
  </div>
);
