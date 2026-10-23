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
  value
    .normalize('NFD')
    .replace(/[\u0300-\u036f]/g, '')
    .toLocaleLowerCase()
    .trim();

/** Only tolerate one edit (including swapped adjacent letters) in tokens with
 * four or more characters. Short tokens and paths keep precise matching. */
function oneEditApart(left: string, right: string): boolean {
  if (Math.abs(left.length - right.length) > 1) return false;
  let i = 0;
  let j = 0;
  let edits = 0;
  while (i < left.length && j < right.length) {
    if (left[i] === right[j]) {
      i++;
      j++;
      continue;
    }
    edits++;
    if (edits > 1) return false;
    if (
      left.length === right.length &&
      left[i] === right[j + 1] &&
      left[i + 1] === right[j]
    ) {
      i += 2;
      j += 2;
    } else if (left.length > right.length) {
      i++;
    } else if (left.length < right.length) {
      j++;
    } else {
      i++;
      j++;
    }
  }
  return edits + Number(i < left.length || j < right.length) <= 1;
}

function termMatches(term: string, haystack: string): boolean {
  if (haystack.includes(term)) return true;
  if (term.length < 4 || /[^a-z0-9]/i.test(term)) return false;
  return haystack
    .split(/[^a-z0-9]+/)
    .some((word) => word.length >= 4 && oneEditApart(term, word));
}

export function filterCommands(commands: NexusCommand[], query: string): NexusCommand[] {
  const value = query.trim();
  const prefix = value[0];
  const kind: NexusCommandKind | null =
    prefix === '#'
      ? 'project'
      : prefix === '@'
        ? 'agent'
        : prefix === '>'
          ? 'action'
          : prefix === '$'
            ? 'resource'
            : null;
  const needle = normalizeSearch(kind ? value.slice(1) : value);
  const terms = needle.split(/\s+/).filter(Boolean);
  return commands
    .filter((command) => !kind || (command.kind ?? 'action') === kind)
    .map((command, index) => {
      const label = normalizeSearch(command.label);
      const haystack = normalizeSearch(
        [command.label, command.group, command.description, ...(command.keywords ?? [])]
          .filter(Boolean)
          .join(' '),
      );
      if (!terms.every((term) => termMatches(term, haystack))) {
        return { command, index, score: 0 };
      }
      const exactTerms = terms.every((term) => haystack.includes(term));
      const score = !needle
        ? 1
        : label === needle
          ? 100
          : label.startsWith(needle)
            ? 80
            : label.includes(needle)
              ? 60
              : 30 + terms.length;
      return { command, index, score: exactTerms ? score : 10 + terms.length };
    })
    .filter((entry) => entry.score > 0)
    .sort((a, b) => b.score - a.score || a.index - b.index)
    .map((entry) => entry.command);
}
