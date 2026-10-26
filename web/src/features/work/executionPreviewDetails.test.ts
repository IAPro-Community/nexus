import { describe, expect, it } from 'vitest';
import {
  executionPreviewDetails,
  localizeExecutionPreviewMessage,
} from './executionPreviewDetails';

describe('executionPreviewDetails', () => {
  it('localizes and deduplicates the no-provider reason across preview fields', () => {
    const details = executionPreviewDetails(
      {
        agentReason: 'Nenhum Agent elegível atende aos requisitos da tarefa.',
        resourceReason: 'No provider accounts discovered',
        blockers: ['no provider accounts discovered'],
      },
      'Nenhuma conta de provedor foi encontrada.',
    );

    expect(details).toEqual([
      'Nenhum Agent elegível atende aos requisitos da tarefa.',
      'Nenhuma conta de provedor foi encontrada.',
    ]);
  });

  it('deduplicates repeated details while preserving distinct blockers', () => {
    const details = executionPreviewDetails(
      {
        agentReason: 'No eligible Agent.',
        resourceReason: 'No eligible Agent.',
        blockers: ['No eligible Agent!', 'Provider quota is unknown'],
      },
      'No provider accounts are available.',
    );

    expect(details).toEqual(['No eligible Agent.', 'Provider quota is unknown']);
  });

  it('localizes the stable missing-cost-budget message', () => {
    expect(
      localizeExecutionPreviewMessage(
        'No local cost budget is configured.',
        'Nenhum orçamento de custo local está configurado.',
      ),
    ).toBe('Nenhum orçamento de custo local está configurado.');
    expect(localizeExecutionPreviewMessage('Custom provider error', 'Fallback')).toBe(
      'Custom provider error',
    );
  });
});
