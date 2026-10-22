import React, { useEffect, useMemo, useState } from 'react';
import { Command, CornerDownLeft } from 'lucide-react';
import { Dialog, SearchInput } from '../../design-system';
import { filterCommands, type NexusCommand } from './registry';
import styles from './CommandPalette.module.scss';
import { useTranslation } from 'react-i18next';

export const CommandPalette: React.FC<{
  open: boolean;
  onClose: () => void;
  commands: NexusCommand[];
}> = ({ open, onClose, commands }) => {
  const { t } = useTranslation();
  const [query, setQuery] = useState('');
  const [active, setActive] = useState(0);
  const filtered = useMemo(() => filterCommands(commands, query), [commands, query]);
  useEffect(() => {
    if (open) {
      setQuery('');
      setActive(0);
    }
  }, [open]);
  useEffect(() => setActive(0), [query]);
  const execute = (command: NexusCommand | undefined) => {
    if (!command) return;
    command.run();
    onClose();
  };
  return (
    <Dialog open={open} onClose={onClose} title={t('commands.title')} wide>
      <div
        className="nx-command-palette"
        onKeyDown={(event) => {
          if (event.key === 'ArrowDown') {
            event.preventDefault();
            setActive((value) => Math.min(Math.max(0, filtered.length - 1), value + 1));
          } else if (event.key === 'ArrowUp') {
            event.preventDefault();
            setActive((value) => Math.max(0, value - 1));
          } else if (event.key === 'Enter') {
            event.preventDefault();
            execute(filtered[active]);
          }
        }}
      >
        <SearchInput
          value={query}
          onChange={setQuery}
          placeholder={t('commands.placeholder')}
          autoFocus
        />
        <div className={styles.hints} aria-label={t('commands.searchScopes')}>
          <span>
            <kbd>#</kbd> {t('commands.scopeProjects')}
          </span>
          <span>
            <kbd>@</kbd> {t('commands.scopeAgents')}
          </span>
          <span>
            <kbd>$</kbd> {t('commands.scopeResources')}
          </span>
          <span>
            <kbd>&gt;</kbd> {t('commands.scopeActions')}
          </span>
        </div>
        <div className="nx-command-list" role="listbox" aria-label={t('commands.available')}>
          {filtered.map((command, index) => (
            <button
              type="button"
              role="option"
              aria-selected={index === active}
              data-active={index === active ? 'true' : 'false'}
              key={command.id}
              onMouseEnter={() => setActive(index)}
              onClick={() => execute(command)}
            >
              <span className="nx-command-icon">
                <Command size={15} />
              </span>
              <span className="nx-command-copy">
                <strong>{command.label}</strong>
                <small>{command.description || command.group}</small>
              </span>
              <span className="nx-command-meta">
                {command.shortcut || command.group}
                {index === active && <CornerDownLeft size={12} />}
              </span>
            </button>
          ))}
          {filtered.length === 0 && <p className="nx-command-empty">{t('commands.none')}</p>}
        </div>
      </div>
    </Dialog>
  );
};
