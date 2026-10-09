import React from 'react';
import { TerminalPane } from '../../components/TerminalPane';
import { api } from '../../api';
import { useWorkspacePresentation } from '../../workspace/WorkspacePresentationProvider';

export const ProjectShellSurface: React.FC<{
  runtimeId: string;
  title?: string;
  liveTitleKey?: string;
  onRuntimeChanged?: (title: string) => void | Promise<void>;
  onRestart?: () => void | Promise<void>;
  initialInput?: string;
  onInitialInputConsumed?: () => void;
}> = ({
  runtimeId,
  title = 'Project Shell',
  liveTitleKey,
  onRuntimeChanged,
  onRestart,
  initialInput,
  onInitialInputConsumed,
}) => {
  const presentation = useWorkspacePresentation();
  const hideHeader = presentation.state.mode === 'DESKTOP' || presentation.state.mode === 'MOSAIC';
  return (
    <div
      className="nx-agent-terminal-surface nx-project-shell-surface"
      data-runtime-id={runtimeId}
      data-chrome={hideHeader ? 'window' : 'full'}
    >
      <TerminalPane
        runtimeId={runtimeId}
        title={title}
        provider="shell"
        profile="local"
        hideHeader={hideHeader}
        liveTitleKey={liveTitleKey}
        onUpdateTitle={async (id, nextTitle) => {
          await api.updateRuntimeTitle(id, nextTitle);
          await onRuntimeChanged?.(nextTitle);
        }}
        onRestart={onRestart}
        initialInput={initialInput}
        onInitialInputConsumed={onInitialInputConsumed}
      />
    </div>
  );
};
