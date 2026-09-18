import React, { useState } from 'react';
import { RuntimeSession } from '../types';
import { TerminalPane } from './TerminalPane';
import { Columns2, Square, Grid2X2 } from 'lucide-react';

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
  const [splitMode, setSplitMode] = useState<'single' | 'split-h' | 'grid'>('single');
  const [openIds, setOpenIds] = useState<string[]>(() => {
    if (activeRuntimeId) return [activeRuntimeId];
    if (runtimes.length > 0) return [runtimes[0].runtime_id];
    return [];
  });

  // Ensure activeRuntimeId is tracked in openIds
  React.useEffect(() => {
    if (activeRuntimeId) {
      setOpenIds((prev) => (prev.includes(activeRuntimeId) ? prev : [...prev, activeRuntimeId]));
    }
  }, [activeRuntimeId]);

  if (runtimes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-slate-500 font-mono text-xs">
        <p>No active supervised runtimes.</p>
        <p className="mt-1 text-[11px] text-slate-600">
          Start an agent runtime from Dashboard to open terminal.
        </p>
      </div>
    );
  }

  const handleCloseTab = (id: string) => {
    setOpenIds((prev) => {
      const next = prev.filter((item) => item !== id);
      if (id === activeRuntimeId && next.length > 0) {
        onSelectRuntime(next[0]);
      }
      return next;
    });
  };

  // Determine which runtimes to render based on split mode
  const currentId = activeRuntimeId || (openIds.length > 0 ? openIds[0] : runtimes[0].runtime_id);
  const activeSession = runtimes.find((r) => r.runtime_id === currentId);

  let renderSessions: RuntimeSession[] = [];
  if (splitMode === 'single') {
    if (activeSession) renderSessions = [activeSession];
  } else if (splitMode === 'split-h') {
    renderSessions = runtimes.slice(0, 2);
  } else if (splitMode === 'grid') {
    renderSessions = runtimes.slice(0, 4);
  }

  return (
    <div className="flex flex-col h-full space-y-2">
      {/* Tab bar & Layout controls */}
      <div className="flex items-center justify-between border-b border-slate-800 pb-2 px-1">
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
                onClick={() => onSelectRuntime(r.runtime_id)}
                className={`flex items-center space-x-2 px-3 py-1.5 rounded text-xs font-mono transition border ${
                  isActive
                    ? 'bg-slate-900 text-white border-slate-700 font-semibold shadow-sm'
                    : 'text-slate-400 hover:text-slate-200 border-transparent hover:bg-slate-900/60'
                }`}
              >
                {isWaiting ? (
                  <span
                    className="w-2 h-2 rounded-full bg-amber-400 animate-ping"
                    title="Aguardando resposta"
                  ></span>
                ) : isApproval ? (
                  <span
                    className="w-2 h-2 rounded-full bg-rose-400 animate-pulse"
                    title="Aprovação necessária"
                  ></span>
                ) : isDone ? (
                  <span
                    className="w-2 h-2 rounded-full bg-emerald-400"
                    title="Tarefa concluída"
                  ></span>
                ) : (
                  <span className="w-2 h-2 rounded-full bg-sky-400" title="Em execução"></span>
                )}
                <span className="font-sans font-medium">{title}</span>
                <span className="text-slate-500 text-[10px]">[{r.runtime_id}]</span>
              </button>
            );
          })}
        </div>

        {/* Split View Mode Selector */}
        <div className="flex items-center space-x-1 bg-slate-900 border border-slate-800 rounded p-0.5">
          <button
            onClick={() => setSplitMode('single')}
            className={`p-1 rounded transition ${
              splitMode === 'single'
                ? 'bg-slate-800 text-sky-400'
                : 'text-slate-400 hover:text-white'
            }`}
            title="Single Terminal"
          >
            <Square className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={() => setSplitMode('split-h')}
            className={`p-1 rounded transition ${
              splitMode === 'split-h'
                ? 'bg-slate-800 text-sky-400'
                : 'text-slate-400 hover:text-white'
            }`}
            title="Split Side-by-Side"
          >
            <Columns2 className="w-3.5 h-3.5" />
          </button>
          <button
            onClick={() => setSplitMode('grid')}
            className={`p-1 rounded transition ${
              splitMode === 'grid' ? 'bg-slate-800 text-sky-400' : 'text-slate-400 hover:text-white'
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
