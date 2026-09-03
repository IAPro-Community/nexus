import React, { useState, useEffect, useCallback } from 'react';

export interface AttentionEvent {
  id: string;
  projectId: string;
  agentId?: string;
  runtimeId?: string;
  flowRunId?: string;
  viewLogicalKey?: string;
  state: string;
  reason: string;
  message?: string;
  source: string;
  occurredAt: string;
  acknowledgedAt?: string;
}

export interface ProjectAttentionSummary {
  projectId: string;
  total: number;
  working: number;
  waitingInput: number;
  waitingApproval: number;
  completed: number;
  completedUnread: number;
  failed: number;
  rateLimited: number;
  recoverable: number;
  priorityState?: string;
  priorityReason?: string;
  priorityEventId?: string;
}

interface AttentionCenterProps {
  summaries: ProjectAttentionSummary[];
  onOpenProject: (projectId: string) => void;
  onMarkSeen: (projectId: string, eventId: string) => void;
}

const STATE_ICONS: Record<string, string> = {
  WORKING: '⏳',
  WAITING_INPUT: '❓',
  WAITING_APPROVAL: '⚠️',
  COMPLETED: '✅',
  FAILED: '❌',
  RATE_LIMITED: '🚫',
  RECOVERABLE: '🔄',
};

const STATE_COLORS: Record<string, string> = {
  WORKING: 'text-blue-500',
  WAITING_INPUT: 'text-amber-500',
  WAITING_APPROVAL: 'text-rose-500',
  COMPLETED: 'text-emerald-500',
  FAILED: 'text-red-500',
  RATE_LIMITED: 'text-orange-500',
  RECOVERABLE: 'text-purple-500',
};

export function AttentionCenter({ summaries, onOpenProject, onMarkSeen }: AttentionCenterProps) {
  const sorted = [...summaries].sort((a, b) => {
    const priorityOrder: Record<string, number> = {
      FAILED: 7,
      WAITING_APPROVAL: 6,
      WAITING_INPUT: 5,
      COMPLETED: 4,
      RECOVERABLE: 3,
      RATE_LIMITED: 2,
      WORKING: 1,
    };
    const aPriority = priorityOrder[a.priorityState || ''] || 0;
    const bPriority = priorityOrder[b.priorityState || ''] || 0;
    return bPriority - aPriority;
  });

  return (
    <div className="nx-attention-center">
      <h3 className="text-sm font-semibold mb-3">Attention</h3>
      {sorted.length === 0 && (
        <p className="text-xs text-muted-foreground">No pending attention.</p>
      )}
      {sorted.map((summary) => (
        <div
          key={summary.projectId}
          className="nx-attention-item flex items-center gap-2 p-2 rounded hover:bg-accent cursor-pointer"
          onClick={() => onOpenProject(summary.projectId)}
        >
          <span className="text-lg">{STATE_ICONS[summary.priorityState || ''] || '●'}</span>
          <div className="flex-1 min-w-0">
            <div className="text-sm font-medium truncate">{summary.projectId}</div>
            <div className="text-xs text-muted-foreground truncate">
              {summary.priorityReason || `${summary.total} events`}
            </div>
          </div>
          {summary.completedUnread > 0 && (
            <span className="text-xs bg-emerald-100 text-emerald-700 px-1.5 py-0.5 rounded-full">
              ✓ {summary.completedUnread}
            </span>
          )}
          {summary.waitingApproval > 0 && (
            <span className="text-xs bg-rose-100 text-rose-700 px-1.5 py-0.5 rounded-full">
              ⚠ {summary.waitingApproval}
            </span>
          )}
          {summary.failed > 0 && (
            <span className="text-xs bg-red-100 text-red-700 px-1.5 py-0.5 rounded-full">
              ❌ {summary.failed}
            </span>
          )}
        </div>
      ))}
    </div>
  );
}

interface AttentionEventListProps {
  events: AttentionEvent[];
  onMarkSeen: (eventId: string) => void;
  onFocusView?: (viewLogicalKey: string) => void;
}

export function AttentionEventList({ events, onMarkSeen, onFocusView }: AttentionEventListProps) {
  return (
    <div className="nx-attention-events space-y-2">
      {events.map((event) => (
        <div
          key={event.id}
          className="nx-attention-event flex items-start gap-2 p-2 rounded border"
        >
          <span className="text-lg mt-0.5">{STATE_ICONS[event.state] || '●'}</span>
          <div className="flex-1 min-w-0">
            <div className="flex items-center gap-2">
              <span className="text-sm font-medium">{event.reason}</span>
              <span className="text-xs text-muted-foreground">{event.source}</span>
            </div>
            {event.message && (
              <p className="text-xs text-muted-foreground mt-1 line-clamp-2">{event.message}</p>
            )}
            <div className="flex items-center gap-2 mt-1">
              <span className="text-xs text-muted-foreground">
                {new Date(event.occurredAt).toLocaleTimeString()}
              </span>
              {event.viewLogicalKey && onFocusView && (
                <button
                  className="text-xs text-blue-500 hover:underline"
                  onClick={(e) => {
                    e.stopPropagation();
                    onFocusView(event.viewLogicalKey!);
                  }}
                >
                  Open
                </button>
              )}
              {!event.acknowledgedAt && (
                <button
                  className="text-xs text-muted-foreground hover:underline"
                  onClick={(e) => {
                    e.stopPropagation();
                    onMarkSeen(event.id);
                  }}
                >
                  Mark Seen
                </button>
              )}
            </div>
          </div>
        </div>
      ))}
    </div>
  );
}
