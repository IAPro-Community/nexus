export type NexusCommandKind = 'action' | 'project' | 'agent' | 'resource';

export interface NexusCommand {
  id: string;
  label: string;
  group: string;
  description?: string;
  shortcut?: string;
  keywords?: string[];
  kind?: NexusCommandKind;
  run: () => void;
}

const normalizeSearch = (value: string): string =>
  value.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLocaleLowerCase().trim();

export function filterCommands(commands: NexusCommand[], query: string): NexusCommand[] {
  const value = query.trim();
  const prefix = value[0];
  const kind: NexusCommandKind | null =
    prefix === '#' ? 'project' : prefix === '@' ? 'agent' : prefix === '>' ? 'action' : prefix === '$' ? 'resource' : null;
  const needle = normalizeSearch(kind ? value.slice(1) : value);
  const terms = needle.split(/\s+/).filter(Boolean);
  return commands
    .filter((command) => !kind || (command.kind ?? 'action') === kind)
    .map((command, index) => {
      const label = normalizeSearch(command.label);
      const haystack = normalizeSearch(
        [command.label, command.group, command.description, ...(command.keywords ?? [])]
          .filter(Boolean).join(' '),
      );
      if (!terms.every((term) => haystack.includes(term))) return { command, index, score: 0 };
      const score = !needle
        ? 1
        : label === needle
          ? 100
          : label.startsWith(needle)
            ? 80
            : label.includes(needle)
              ? 60
              : 30 + terms.length;
      return { command, index, score };
    })
    .filter((entry) => entry.score > 0)
    .sort((a, b) => b.score - a.score || a.index - b.index)
    .map((entry) => entry.command);
}
