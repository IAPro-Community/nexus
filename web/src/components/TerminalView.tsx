import React, { useState } from 'react';
import { RuntimeSession } from '../types';
import { TerminalPane } from './TerminalPane';
import { Columns2, Square, Grid2X2 } from 'lucide-react';
import { normalizeOpenRuntimeIds, visibleTerminalRuntimes, type TerminalSplitMode } from './terminalViewModel';

interface TerminalViewProps {
  runtimes: RuntimeSession[];
  activeRuntimeId?: string;
  onSelectRuntime: (id: string) => void;
  onUpdateTitle?: (id: string, title: string) => void;
}

export const TerminalView: React.FC<TerminalViewProps> = ({
  runtimes,
  activeRuntimeId,
  onSelectRuntime,
  onUpdateTitle,
}) => {
  const [splitMode, setSplitMode] = useState<TerminalSplitMode>('single');
  const [openIds, setOpenIds] = useState<string[]>(() => {
    if (activeRuntimeId) return [activeRuntimeId];
    if (runtimes.length > 0) return [runtimes[0].runtime_id];
    return [];
  });

  // Session selection and split membership are view-only, never a runtime launch.
  // Normalize when a runtime disappears or the host selects another session.
  React.useEffect(() => {
    setOpenIds((prev) => {
      const next = normalizeOpenRuntimeIds(prev, runtimes, activeRuntimeId);
      return next.length === prev.length && next.every((id, i) => id === prev[i]) ? prev : next;
    });
  }, [runtimes, activeRuntimeId]);

  if (runtimes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-[var(--nx-muted)] font-mono text-xs">
        <p>No active supervised runtimes.</p>
        <p className="mt-1 text-[11px] text-[var(--nx-subtle)]">
          Start an agent runtime from Dashboard to open terminal.
        </p>
      </div>
    );
  }

  const opened = normalizeOpenRuntimeIds(openIds, runtimes, activeRuntimeId);
  const currentId = activeRuntimeId && runtimes.some((r) => r.runtime_id === activeRuntimeId)
    ? activeRuntimeId
    : opened[0];
  const renderSessions = visibleTerminalRuntimes(runtimes, opened, currentId, splitMode);

  const handleSelectTab = (id: string) => {
    setOpenIds((prev) => prev.includes(id) ? prev : [...prev, id]);
    onSelectRuntime(id);
  };

  const handleCloseTab = (id: string) => {
    const remaining = opened.filter((item) => item !== id);
    // The parent controls activeRuntimeId: keep the final view instead of
    // appearing to close it and immediately reopening it on the next render.
    if (remaining.length === 0) return;
    setOpenIds(remaining);
    if (id === currentId) onSelectRuntime(remaining[0]);
  };

  return (
    <div className="flex flex-col h-full space-y-2">
      {/* Tab bar & Layout controls */}
      <div className="flex items-center justify-between border-b border-[var(--nx-border)] pb-2 px-1">
        {/* Terminal Tabs */}
        <div className="flex items-center space-x-1.5 overflow-x-auto">
          {runtimes.map((r) => {
            const isActive = r.runtime_id === currentId;
            const prov = r.provider_id || r.provider || 'AI';
            const prof = r.profile_id || r.profile || 'default';
            const title = r.dynamic_title || r.title || `${prov.toUpperCase()} (${prof})`;
            const isWaiting = r.attention_reason === 'QUESTION' || r.state === 'WAITING';
            const isApproval = r.attention_reason === 'APPROVAL' || r.state === 'APPROVAL';
            const isDone = r.attention_reason === 'TASK_COMPLETED';
            return (
              <button
                key={r.runtime_id}
                onClick={() => handleSelectTab(r.runtime_id)}
                className={`flex items-center space-x-2 px-3 py-1.5 rounded text-xs font-mono transition border ${
                  isActive
                    ? 'bg-[var(--nx-surface)] text-[var(--nx-text)] border-[var(--nx-border-strong)] font-semibold shadow-sm'
                    : 'text-[var(--nx-muted)] hover:text-[var(--nx-text-soft)] border-transparent hover:bg-[var(--nx-surface-2)]'
                }`}
              >
                {isWaiting ? (
                  <span
                    className="w-2 h-2 rounded-full bg-[var(--nx-warning)] animate-ping"
                    title="Aguardando resposta"
                  ></span>
                ) : isApproval ? (
                  <span
                    className="w-2 h-2 rounded-full bg-[var(--nx-danger)] animate-pulse"
                    title="Aprovação necessária"
                  ></span>
                ) : isDone ? (
                  <span
                    className="w-2 h-2 rounded-full bg-[var(--nx-success)]"
                    title="Tarefa concluída"
                  ></span>
                ) : (
                  <span
                    className="w-2 h-2 rounded-full bg-[var(--nx-info)]"
                    title="Em execução"
                  ></span>
                )}
                <span className="font-sans font-medium">{title}</span>
                <span className="text-[var(--nx-muted)] text-[10px]">[{r.runtime_id}]</span>
              </button>
            );
          })}
        </div>

        {/* Split View Mode Selector */}
        <div className="flex items-center space-x-1 bg-[var(--nx-surface)] border border-[var(--nx-border)] rounded p-0.5">
          <button
            onClick={() => setSplitMode('single')}
            className={`p-1 rounded transition ${
              splitMode === 'single'
                ? 'bg-[var(--nx-surface-3)] text-[var(--nx-accent-text)]'
                : 'text-[var(--nx-muted)] hover:text-[var(--nx-text)]'
            }`}
            title="Single Terminal"
          >
            <Square className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={() => setSplitMode('split-h')}
            className={`p-1 rounded transition ${
              splitMode === 'split-h'
                ? 'bg-[var(--nx-surface-3)] text-[var(--nx-accent-text)]'
                : 'text-[var(--nx-muted)] hover:text-[var(--nx-text)]'
            }`}
            title="Split Side-by-Side"
          >
            <Columns2 className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={() => setSplitMode('grid')}
            className={`p-1 rounded transition ${
              splitMode === 'grid'
                ? 'bg-[var(--nx-surface-3)] text-[var(--nx-accent-text)]'
                : 'text-[var(--nx-muted)] hover:text-[var(--nx-text)]'
            }`}
            title="2x2 Grid View"
          >
            <Grid2X2 className="w-3.5 h-3.5" />
          </button>
        </div>
      </div>

      {/* Terminal Grid */}
      <div
        className={`flex-1 grid gap-2 overflow-hidden ${
          splitMode === 'single'
            ? 'grid-cols-1 grid-rows-1'
            : splitMode === 'split-h'
              ? 'grid-cols-1 md:grid-cols-2 grid-rows-1'
              : 'grid-cols-1 md:grid-cols-2 grid-rows-2'
        }`}
      >
        {renderSessions.map((r) => (
          <TerminalPane
            key={r.runtime_id}
            runtimeId={r.runtime_id}
            title={r.title}
            provider={r.provider_id || r.provider || 'AI'}
            profile={r.profile_id || r.profile || 'default'}
            onUpdateTitle={onUpdateTitle}
            onClose={() => handleCloseTab(r.runtime_id)}
          />
        ))}
      </div>
    </div>
  );
};
