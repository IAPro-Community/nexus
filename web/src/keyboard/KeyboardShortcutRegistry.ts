export type ShortcutScope = 'global' | 'workspace' | 'terminal' | 'chat' | 'flow' | 'dialog';

export interface ShortcutDefinition {
  id: string;
  key: string; // e.g. 'k', 'p', 'Enter', 'Escape'
  ctrlOrMeta?: boolean;
  shift?: boolean;
  alt?: boolean;
  scope: ShortcutScope;
  description: string;
  action: (event: KeyboardEvent) => void;
  /** If true, default browser action is prevented. */
  preventDefault?: boolean;
}

export class KeyboardShortcutRegistry {
  private static instance: KeyboardShortcutRegistry;
  private shortcuts: Map<string, ShortcutDefinition> = new Map();
  private activeScope: ShortcutScope = 'global';
  private listenerAttached = false;

  private constructor() {
    this.handleKeyDown = this.handleKeyDown.bind(this);
  }

  public static getInstance(): KeyboardShortcutRegistry {
    if (!KeyboardShortcutRegistry.instance) {
      KeyboardShortcutRegistry.instance = new KeyboardShortcutRegistry();
    }
    return KeyboardShortcutRegistry.instance;
  }

  public setScope(scope: ShortcutScope): void {
    this.activeScope = scope;
  }

  public getScope(): ShortcutScope {
    return this.activeScope;
  }

  public register(shortcut: ShortcutDefinition): () => void {
    this.shortcuts.set(shortcut.id, shortcut);
    this.ensureListener();
    return () => {
      this.shortcuts.delete(shortcut.id);
    };
  }

  public clear(): void {
    this.shortcuts.clear();
  }

  private ensureListener(): void {
    if (this.listenerAttached || typeof window === 'undefined') return;
    window.addEventListener('keydown', this.handleKeyDown, { capture: true });
    this.listenerAttached = true;
  }

  public handleKeyDown(event: KeyboardEvent): void {
    // Check if user is typing IME composition
    if (event.isComposing || event.keyCode === 229) {
      return;
    }

    const target = event.target as HTMLElement | null;
    const isInputOrTextarea =
      target?.tagName === 'INPUT' || target?.tagName === 'TEXTAREA' || target?.isContentEditable;
    const isTerminalTarget = Boolean(
      target?.closest('.xterm') || target?.classList.contains('xterm-helper-textarea'),
    );

    // If focus is inside terminal, NEVER hijack standard shell / terminal keys
    // Only allow safe IDE triggers (Palette, Zen Focus, Rail Drawer, Tab Switch, New/Close Terminal)
    if (isTerminalTarget && this.activeScope !== 'dialog') {
      const keyLower = event.key.toLowerCase();
      const isCtrlOrMeta = event.ctrlKey || event.metaKey;
      const isPalette = isCtrlOrMeta && (keyLower === 'k' || (event.shiftKey && keyLower === 'p'));
      const isZenKey = event.key === 'F11' || (isCtrlOrMeta && event.shiftKey && keyLower === 'f');
      const isRailToggle = isCtrlOrMeta && keyLower === 'b';
      const isAltTab =
        event.altKey && !event.ctrlKey && !event.metaKey && /^[1-9]$/.test(event.key);
      const isTerminalMgmt =
        isCtrlOrMeta && event.shiftKey && (keyLower === 't' || keyLower === 'w');

      if (!isPalette && !isZenKey && !isRailToggle && !isAltTab && !isTerminalMgmt) {
        return;
      }
    }

    // Match candidate shortcuts
    for (const shortcut of this.shortcuts.values()) {
      // Scope validation: 'global' applies everywhere unless in dialog where dialog has priority
      if (shortcut.scope !== 'global' && shortcut.scope !== this.activeScope) {
        continue;
      }

      const matchKey = event.key.toLowerCase() === shortcut.key.toLowerCase();
      const matchCtrl = Boolean(shortcut.ctrlOrMeta) === (event.ctrlKey || event.metaKey);
      const matchShift = Boolean(shortcut.shift) === event.shiftKey;
      const matchAlt = Boolean(shortcut.alt) === event.altKey;

      if (matchKey && matchCtrl && matchShift && matchAlt) {
        // If typing in input, don't intercept plain navigation keys without modifier
        if (
          isInputOrTextarea &&
          !shortcut.ctrlOrMeta &&
          !shortcut.alt &&
          shortcut.key !== 'Escape'
        ) {
          continue;
        }

        if (shortcut.preventDefault) {
          event.preventDefault();
        }
        shortcut.action(event);
        break;
      }
    }
  }
}
