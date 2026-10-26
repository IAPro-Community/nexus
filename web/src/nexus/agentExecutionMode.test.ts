import { describe, expect, it } from 'vitest';
import { executionModeFromOptions, withAgentExecutionMode } from './agentExecutionMode';

describe('agent execution mode', () => {
  it('reads YOLO from the persisted UI mode or a legacy provider bypass argument', () => {
    expect(executionModeFromOptions({ mode: 'YOLO' })).toBe('YOLO');
    expect(executionModeFromOptions({ extra_args: ['--yolo'] })).toBe('YOLO');
    expect(
      executionModeFromOptions({ extra_args: ['--dangerously-bypass-approvals-and-sandbox'] }),
    ).toBe('YOLO');
    expect(executionModeFromOptions({ mode: 'Safe', extra_args: [] })).toBe('Safe');
    expect(executionModeFromOptions({ mode: 'Safe', extra_args: ['--plan'] })).toBe('Safe');
  });

  it('replaces prior execution-mode arguments while preserving unrelated provider arguments', () => {
    const previous = {
      mode: 'YOLO',
      extra_args: ['--plan', '--yolo', '--sandbox', 'workspace-write'],
      origin: 'native',
    };

    expect(withAgentExecutionMode(previous, 'Safe')).toEqual({
      mode: 'Safe',
      extra_args: ['--sandbox', 'workspace-write'],
      origin: 'native',
    });
    expect(withAgentExecutionMode(previous, 'YOLO')).toEqual({
      mode: 'YOLO',
      extra_args: ['--sandbox', 'workspace-write', '--yolo'],
      origin: 'native',
    });
  });
});
