import React, { useMemo, useRef, useState } from 'react';
import { Bot, CircleCheck, CircleDot, Link2, Minus, Network, Plus, Scan } from 'lucide-react';
import { Badge, Button } from '../../design-system';
import { executionWaves, type FlowDraftModel } from './flowModel';

const NODE_WIDTH = 190;
const NODE_HEIGHT = 122;
const COLUMN_GAP = 90;
const ROW_GAP = 24;

export const FlowCanvas: React.FC<{
  flow: FlowDraftModel;
  selectedId?: string;
  onSelect: (stepId: string) => void;
  onConnect?: (from: string, to: string) => void;
}> = ({ flow, selectedId, onSelect, onConnect }) => {
  const [zoom, setZoom] = useState(1);
  const [pan] = useState({ x: 24, y: 24 });
  const [connectFrom, setConnectFrom] = useState<string | null>(null);
  const [positions, setPositions] = useState<Record<string, { x: number; y: number }>>({});
  const dragRef = useRef<{ id: string; x: number; y: number } | null>(null);
  const graph = useMemo(() => {
    try {
      const next: Record<string, { x: number; y: number }> = {};
      executionWaves(flow).forEach((wave, column) =>
        wave.forEach((id, row) => {
          next[id] = { x: column * (NODE_WIDTH + COLUMN_GAP), y: row * (NODE_HEIGHT + ROW_GAP) };
        }),
      );
      return { error: '', positions: next };
    } catch (error) {
      return { error: error instanceof Error ? error.message : String(error), positions: {} };
    }
  }, [flow]);
  const byId = useMemo(
    () => new Map((flow.steps || []).map((step) => [step.id, step])),
    [flow.steps],
  );
  const nodePositions = useMemo(
    () => ({ ...graph.positions, ...positions }),
    [graph.positions, positions],
  );
  const edges = useMemo(
    () =>
      (flow.steps || []).flatMap((step) =>
        (step.dependencies || []).map((from) => ({ from, to: step.id })),
      ),
    [flow.steps],
  );
  const width = Math.max(
    700,
    ...Object.values(nodePositions).map((position) => position.x + NODE_WIDTH + 40),
  );
  const height = Math.max(
    280,
    ...Object.values(nodePositions).map((position) => position.y + NODE_HEIGHT + 40),
  );
  const pointerDown = (event: React.PointerEvent, id: string) => {
    if (connectFrom) {
      if (connectFrom !== id) onConnect?.(connectFrom, id);
      setConnectFrom(null);
      return;
    }
    onSelect(id);
    const position = nodePositions[id];
    if (position)
      dragRef.current = { id, x: event.clientX - position.x, y: event.clientY - position.y };
  };
  const pointerMove = (event: React.PointerEvent) => {
    const drag = dragRef.current;
    if (drag)
      setPositions((current) => ({
        ...current,
        [drag.id]: {
          x: Math.max(0, event.clientX - drag.x),
          y: Math.max(0, event.clientY - drag.y),
        },
      }));
  };
  const pointerUp = () => {
    dragRef.current = null;
  };
  if (graph.error)
    return (
      <div className="nx-flow-canvas nx-flow-canvas--invalid">
        <Network size={18} />
        <strong>Invalid Flow</strong>
        <span>{graph.error}</span>
      </div>
    );
  return (
    <div
      className="nx-flow-canvas nx-flow-canvas--dag"
      aria-label="Flow dependency graph"
      onPointerMove={pointerMove}
      onPointerUp={pointerUp}
      onPointerLeave={pointerUp}
      onWheel={(event) => {
        setZoom((current) => Math.min(1.6, Math.max(0.55, current - event.deltaY * 0.001)));
      }}
    >
      <div className="nx-flow-canvas__toolbar" role="toolbar" aria-label="DAG controls">
        <Button
          size="sm"
          tone={connectFrom ? 'brand' : 'default'}
          onClick={() => setConnectFrom(connectFrom ? null : selectedId || null)}
        >
          <Link2 size={12} />{' '}
          {connectFrom
            ? `Connect from ${byId.get(connectFrom)?.title || connectFrom}`
            : 'Connect selected'}
        </Button>
        <Button size="sm" onClick={() => setZoom((current) => Math.min(1.6, current + 0.1))}>
          <Plus size={12} /> Zoom
        </Button>
        <Button size="sm" onClick={() => setZoom((current) => Math.max(0.55, current - 0.1))}>
          <Minus size={12} /> Zoom
        </Button>
        <Button
          size="sm"
          onClick={() => {
            setZoom(1);
            setPositions({});
          }}
        >
          <Scan size={12} /> Fit view
        </Button>
        <small>
          {(flow.steps || []).length} nodes · {edges.length} edges
          {connectFrom ? ' · select target' : ''}
        </small>
      </div>
      <div
        className="nx-flow-canvas__viewport"
        style={{ minWidth: width * zoom + 48, minHeight: height * zoom + 48 }}
      >
        <div
          className="nx-flow-canvas__world"
          style={{ width, height, transform: `translate(${pan.x}px, ${pan.y}px) scale(${zoom})` }}
        >
          <svg className="nx-flow-canvas__edges" width={width} height={height} aria-hidden="true">
            <defs>
              <marker
                id="nx-flow-arrow"
                markerWidth="8"
                markerHeight="8"
                refX="7"
                refY="4"
                orient="auto"
              >
                <path d="M0,0 L8,4 L0,8 z" fill="currentColor" />
              </marker>
            </defs>
            {edges.map(({ from, to }) => {
              const start = nodePositions[from];
              const end = nodePositions[to];
              if (!start || !end) return null;
              const x1 = start.x + NODE_WIDTH;
              const y1 = start.y + NODE_HEIGHT / 2;
              const x2 = end.x;
              const y2 = end.y + NODE_HEIGHT / 2;
              const bend = Math.max(30, (x2 - x1) / 2);
              return (
                <path
                  key={`${from}-${to}`}
                  d={`M ${x1} ${y1} C ${x1 + bend} ${y1}, ${x2 - bend} ${y2}, ${x2} ${y2}`}
                  fill="none"
                  stroke="currentColor"
                  markerEnd="url(#nx-flow-arrow)"
                />
              );
            })}
          </svg>
          {(flow.steps || []).map((step) => {
            const position = nodePositions[step.id] || { x: 0, y: 0 };
            return (
              <button
                type="button"
                key={step.id}
                className="nx-flow-node nx-flow-node--dag"
                style={{
                  left: position.x,
                  top: position.y,
                  width: NODE_WIDTH,
                  minHeight: NODE_HEIGHT,
                }}
                data-selected={selectedId === step.id ? 'true' : 'false'}
                onPointerDown={(event) => pointerDown(event, step.id)}
                onClick={() => onSelect(step.id)}
                aria-label={`Flow node ${step.title || step.id}`}
              >
                <div className="nx-flow-node__title">
                  {step.status === 'VERIFIED' ? <CircleCheck size={13} /> : <CircleDot size={13} />}
                  <strong>{step.title || step.id}</strong>
                </div>
                <div className="nx-flow-node__meta">
                  <Badge tone="default">{step.assignmentStrategy}</Badge>
                  {step.parallelGroup && <Badge tone="brand">{step.parallelGroup}</Badge>}
                </div>
                <small>
                  {(step.dependencies || []).length
                    ? `after ${(step.dependencies || []).join(', ')}`
                    : 'entry node'}
                </small>
                <span className="nx-flow-node__agent">
                  <Bot size={11} />
                  {step.agentId || step.role || 'Auto resource'}
                </span>
              </button>
            );
          })}
        </div>
      </div>
    </div>
  );
};
