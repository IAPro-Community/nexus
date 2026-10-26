export interface ExecutionPreviewReasonInput {
  agentReason?: string;
  resourceReason?: string;
  blockers?: readonly string[] | null;
}

const canonicalReason = (reason: string): string =>
  reason
    .trim()
    .toLowerCase()
    .replace(/[.!?]+$/u, '')
    .replace(/\s+/gu, ' ');

/** Merge overlapping preview explanations and localize the stable no-provider case. */
export function executionPreviewDetails(
  input: ExecutionPreviewReasonInput,
  noProviderAccountsMessage: string,
): string[] {
  const seen = new Set<string>();
  const details: string[] = [];
  const messages = [input.agentReason, input.resourceReason, ...(input.blockers || [])];

  for (const message of messages) {
    const text = message?.trim();
    if (!text) continue;

    const key = canonicalReason(text);
    if (seen.has(key)) continue;
    seen.add(key);
    details.push(key === 'no provider accounts discovered' ? noProviderAccountsMessage : text);
  }

  return details;
}

export function localizeExecutionPreviewMessage(
  message: string,
  noLocalCostBudgetMessage: string,
): string {
  return canonicalReason(message) === 'no local cost budget is configured'
    ? noLocalCostBudgetMessage
    : message;
}
