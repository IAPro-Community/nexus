export type AgentExecutionMode = 'Safe' | 'YOLO';

const EXECUTION_MODE_ARGUMENTS = new Set([
  '--plan',
  '--yolo',
  '-y',
  '--dangerously-skip-permissions',
  '--dangerously-bypass-approvals-and-sandbox',
]);
const YOLO_ARGUMENTS = new Set([
  '--yolo',
  '-y',
  '--dangerously-skip-permissions',
  '--dangerously-bypass-approvals-and-sandbox',
]);

export function executionModeFromOptions(
  options: Record<string, unknown> | undefined,
): AgentExecutionMode {
  const extraArgs = Array.isArray(options?.extra_args) ? options.extra_args : [];
  if (
    extraArgs.some((arg) => typeof arg === 'string' && YOLO_ARGUMENTS.has(arg.trim().toLowerCase()))
  ) {
    return 'YOLO';
  }
  return options?.mode === 'YOLO' ? 'YOLO' : 'Safe';
}

export function withAgentExecutionMode(
  options: Record<string, unknown> | undefined,
  mode: AgentExecutionMode,
): Record<string, unknown> {
  const currentArgs = Array.isArray(options?.extra_args) ? options.extra_args : [];
  const extraArgs = currentArgs.filter(
    (arg) => typeof arg !== 'string' || !EXECUTION_MODE_ARGUMENTS.has(arg.trim().toLowerCase()),
  );
  if (mode === 'YOLO') extraArgs.push('--yolo');

  return {
    ...options,
    mode,
    extra_args: extraArgs,
  };
}
