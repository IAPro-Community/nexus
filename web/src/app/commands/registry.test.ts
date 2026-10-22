import { describe, expect, it } from 'vitest';
import { filterCommands, type NexusCommand } from './registry';

const commands: NexusCommand[] = [
  {
    id: 'open-agents',
    label: 'Open Agents',
    group: 'Project',
    keywords: ['team', 'workers'],
    run: () => undefined,
  },
  {
    id: 'open-resources',
    label: 'Open Resources',
    group: 'Global',
    keywords: ['quota', 'provider'],
    run: () => undefined,
  },
  {
    id: 'theme',
    label: 'Change Theme',
    group: 'Settings',
    keywords: ['dark', 'light'],
    run: () => undefined,
  },
];

describe('command registry', () => {
  it('returns all commands for empty query', () =>
    expect(filterCommands(commands, '')).toHaveLength(3));
  it('matches labels', () =>
    expect(filterCommands(commands, 'resources')[0].id).toBe('open-resources'));
  it('matches keywords', () =>
    expect(filterCommands(commands, 'quota')[0].id).toBe('open-resources'));
  it('matches group names', () => expect(filterCommands(commands, 'settings')[0].id).toBe('theme'));
});

describe('contextual quick-search prefixes', () => {
  const context: NexusCommand[] = [
    {
      id: 'project',
      kind: 'project',
      label: 'Projeto Educação',
      group: 'Projetos',
      keywords: ['/tmp/educacao'],
      run: () => undefined,
    },
    {
      id: 'agent',
      kind: 'agent',
      label: 'Agente Revisão',
      group: 'Agents',
      keywords: ['security'],
      run: () => undefined,
    },
    {
      id: 'resource',
      kind: 'resource',
      label: 'Contas e modelos',
      group: 'Recursos',
      run: () => undefined,
    },
    {
      id: 'action',
      kind: 'action',
      label: 'Abrir terminais',
      group: 'Ações',
      keywords: ['split'],
      run: () => undefined,
    },
  ];
  it('filters accent-insensitively and checks all query terms', () => {
    expect(filterCommands(context, '#educacao').map((c) => c.id)).toEqual(['project']);
    expect(filterCommands(context, '@revisao security').map((c) => c.id)).toEqual(['agent']);
    expect(filterCommands(context, 'agente projeto')).toEqual([]);
  });
  it('keeps scopes isolated and permits unscoped global results', () => {
    expect(filterCommands(context, '>terminais').map((c) => c.id)).toEqual(['action']);
    expect(filterCommands(context, '$contas').map((c) => c.id)).toEqual(['resource']);
    expect(filterCommands(context, '#').map((c) => c.id)).toEqual(['project']);
    expect(filterCommands(context, '').map((c) => c.id)).toEqual([
      'project',
      'agent',
      'resource',
      'action',
    ]);
  });
});

describe('typo tolerant quick search', () => {
  const commands: NexusCommand[] = [
    {
      id: 'review',
      kind: 'agent',
      label: 'Revisão de código',
      group: 'Agentes',
      run: () => undefined,
    },
    {
      id: 'resource',
      kind: 'resource',
      label: 'Contas de acesso',
      group: 'Recursos',
      run: () => undefined,
    },
    {
      id: 'project',
      kind: 'project',
      label: 'Educação municipal',
      group: 'Projetos',
      run: () => undefined,
    },
  ];

  it('tolerates one character omission and adjacent swaps for long words', () => {
    expect(filterCommands(commands, '@reviaso').map((entry) => entry.id)).toEqual(['review']);
    expect(filterCommands(commands, '#educaco').map((entry) => entry.id)).toEqual(['project']);
    expect(filterCommands(commands, '$conats').map((entry) => entry.id)).toEqual(['resource']);
  });

  it('preserves exact matches ahead of typo matches', () => {
    expect(
      filterCommands(
        [
          {
            id: 'typo',
            kind: 'project',
            label: 'Progeto',
            group: 'Workspace',
            run: () => undefined,
          },
          {
            id: 'exact',
            kind: 'project',
            label: 'Projeto',
            group: 'Workspace',
            run: () => undefined,
          },
        ],
        '#projeto',
      ).map((item) => item.id),
    ).toEqual(['exact', 'typo']);
  });

  it('never fuzzy matches short identifiers or scopes', () => {
    expect(filterCommands(commands, '#ex')).toEqual([]);
    expect(filterCommands(commands, '@rvs')).toEqual([]);
    expect(filterCommands(commands, '#')).toHaveLength(1);
  });
});
