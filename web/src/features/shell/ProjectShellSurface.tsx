import React, { useEffect, useRef, useState } from 'react';
import { TerminalSquare } from 'lucide-react';
import { EmptyState, Spinner } from '../../design-system';
import { api } from '../../api';
import { TerminalPane } from '../../components/TerminalPane';
import type { Project, RuntimeSession } from '../../types';

export const ProjectShellSurface: React.FC<{ project: Project }> = ({ project }) => {
  const [runtime, setRuntime] = useState<RuntimeSession | null>(null);
  const runtimeRef = useRef<RuntimeSession | null>(null);
  const [error, setError] = useState('');

  useEffect(() => {
    let active = true;
    void api.startRuntime('shell', 'default', project.canonical_path, [], 'Project Shell')
      .then((started) => { if (active) { runtimeRef.current = started; setRuntime(started); } })
      .catch((err) => { if (active) setError(err instanceof Error ? err.message : String(err)); });
    return () => {
      active = false;
      if (runtimeRef.current) void api.stopRuntime(runtimeRef.current.runtime_id).catch(() => undefined);
    };
  }, [project.canonical_path]);

  if (error) return <EmptyState icon={<TerminalSquare size={22} />} title="Project Shell unavailable" hint={error} />;
  if (!runtime) return <Spinner label="Starting Project Shell…" />;
  return <div className="nx-agent-terminal-surface"><TerminalPane runtimeId={runtime.runtime_id} title="Project Shell" provider="shell" profile="default" /></div>;
};
