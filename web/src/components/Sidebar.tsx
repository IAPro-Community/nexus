import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { Workspace } from '../types';
import {
  LayoutDashboard,
  Terminal,
  Cpu,
  History,
  FolderGit2,
  ShieldCheck,
  Plus,
  Trash2,
  ExternalLink,
} from 'lucide-react';

interface SidebarProps {
  currentTab: string;
  onSelectTab: (tab: string) => void;
  workspaces: Workspace[];
  activeWorkspace: string;
  onSelectWorkspace: (path: string) => void;
  runtimeCount: number;
  onAddWorkspace?: (path: string, name?: string) => void;
  onRemoveWorkspace?: (path: string) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
  currentTab,
  onSelectTab,
  workspaces,
  activeWorkspace,
  onSelectWorkspace,
  runtimeCount,
  onAddWorkspace,
  onRemoveWorkspace,
}) => {
  const { t } = useTranslation();
  const [showAdd, setShowAdd] = useState(false);
  const [newPath, setNewPath] = useState('');
  const [newName, setNewName] = useState('');

  const handleAddSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (newPath.trim() && onAddWorkspace) {
      onAddWorkspace(newPath.trim(), newName.trim() || undefined);
      setNewPath('');
      setNewName('');
      setShowAdd(false);
    }
  };
  const navItems = [
    { id: 'dashboard', label: t('nav.overview', 'Dashboard'), icon: LayoutDashboard },
    {
      id: 'terminals',
      label: t('nav.terminals', 'Terminals'),
      icon: Terminal,
      badge: runtimeCount > 0 ? runtimeCount : undefined,
    },
    { id: 'runtimes', label: t('nav.runtimes', 'Runtimes'), icon: Cpu },
    { id: 'providers', label: t('nav.providers', 'Providers'), icon: ShieldCheck },
    { id: 'events', label: t('nav.events', 'Event Log'), icon: History },
  ];

  return (
    <aside
      className="w-64 flex flex-col h-full select-none"
      style={{
        background: 'var(--nx-bg-elevated)',
        borderRight: '1px solid var(--nx-border)',
        color: 'var(--nx-text)',
      }}
    >
      {/* Brand Header */}
      <div
        className="p-4 flex items-center space-x-3"
        style={{ borderBottom: '1px solid var(--nx-border)' }}
      >
        <div
          className="w-9 h-9 rounded-lg overflow-hidden flex items-center justify-center flex-shrink-0"
          style={{ background: 'var(--nx-surface-2)', border: '1px solid var(--nx-border)' }}
        >
          <img src="./nexus-icon.png" alt="IAPro Nexus" className="w-full h-full object-contain" />
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center space-x-1.5">
            <h1 className="text-sm font-extrabold iapro-gradient-text tracking-wide truncate">
              IAPro
            </h1>
            <span
              className="text-[10px] font-bold tracking-wider"
              style={{ color: 'var(--nx-text-soft)' }}
            >
              NEXUS
            </span>
          </div>
          <p
            className="text-[10px] font-mono font-medium truncate"
            style={{ color: 'var(--nx-accent-text)' }}
          >
            Workspace OS
          </p>
        </div>
      </div>

      {/* Navigation Links */}
      <div className="p-3 space-y-1">
        <div
          className="px-3 py-1.5 text-[10px] font-semibold font-mono tracking-wider uppercase"
          style={{ color: 'var(--nx-subtle)' }}
        >
          {t('shell.navigation', 'Navigation')}
        </div>
        {navItems.map((item) => {
          const Icon = item.icon;
          const isActive = currentTab === item.id;
          return (
            <button
              key={item.id}
              onClick={() => onSelectTab(item.id)}
              className={`w-full flex items-center justify-between px-3 py-2 rounded-md text-xs font-medium transition ${
                isActive
                  ? 'bg-[var(--nx-accent-soft)] text-[var(--nx-accent-text)] border border-[var(--nx-accent)] shadow-sm'
                  : 'hover:bg-[var(--nx-surface-2)]'
              }`}
              style={{
                color: isActive ? undefined : 'var(--nx-text-soft)',
              }}
            >
              <div className="flex items-center space-x-2.5">
                <Icon
                  className="w-4 h-4"
                  style={{ color: isActive ? 'var(--nx-accent-text)' : 'var(--nx-muted)' }}
                />
                <span className={isActive ? 'font-semibold' : ''}>{item.label}</span>
              </div>
              {item.badge !== undefined && (
                <span
                  className="px-1.5 py-0.5 text-[10px] font-mono font-bold rounded-full"
                  style={{
                    background: 'var(--nx-surface-2)',
                    color: 'var(--nx-accent-text)',
                    border: '1px solid var(--nx-border)',
                  }}
                >
                  {item.badge}
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Projects & Workspaces */}
      <div
        className="p-3 flex-1 overflow-y-auto"
        style={{ borderTop: '1px solid var(--nx-border)' }}
      >
        <div
          className="px-3 py-1.5 text-[10px] font-semibold font-mono tracking-wider uppercase flex items-center justify-between"
          style={{ color: 'var(--nx-subtle)' }}
        >
          <span>{t('rail.projects', 'Projects')}</span>
          <div className="flex items-center space-x-1">
            <button
              onClick={() => setShowAdd(!showAdd)}
              className="p-1 rounded transition hover:bg-[var(--nx-surface-2)]"
              style={{ color: 'var(--nx-text-soft)' }}
              title={t('projects.add', 'Add Project')}
              aria-label={t('projects.add', 'Add Project')}
            >
              <Plus className="w-3.5 h-3.5" />
            </button>
            <FolderGit2 className="w-3.5 h-3.5" style={{ color: 'var(--nx-muted)' }} />
          </div>
        </div>

        {showAdd && (
          <form
            onSubmit={handleAddSubmit}
            className="mb-2 p-2 rounded text-xs space-y-1.5 font-mono"
            style={{
              background: 'var(--nx-surface)',
              border: '1px solid var(--nx-border)',
            }}
          >
            <input
              type="text"
              placeholder={t('rail.path', 'Project path')}
              value={newPath}
              onChange={(e) => setNewPath(e.target.value)}
              className="w-full rounded px-2 py-1 text-[11px] focus:outline-none focus:ring-1 focus:ring-[var(--nx-accent)]"
              style={{
                background: 'var(--nx-bg)',
                border: '1px solid var(--nx-border)',
                color: 'var(--nx-text)',
              }}
              autoFocus
            />
            <input
              type="text"
              placeholder={t('rail.namePlaceholder', 'My Project')}
              value={newName}
              onChange={(e) => setNewName(e.target.value)}
              className="w-full rounded px-2 py-1 text-[11px] focus:outline-none focus:ring-1 focus:ring-[var(--nx-accent)]"
              style={{
                background: 'var(--nx-bg)',
                border: '1px solid var(--nx-border)',
                color: 'var(--nx-text)',
              }}
            />
            <div className="flex justify-end space-x-1 pt-1">
              <button
                type="button"
                onClick={() => setShowAdd(false)}
                className="px-2 py-0.5 rounded text-[10px] hover:bg-[var(--nx-surface-2)] transition"
                style={{ color: 'var(--nx-text-soft)' }}
              >
                {t('directSession.cancel', 'Cancel')}
              </button>
              <button
                type="submit"
                className="px-2.5 py-0.5 rounded text-[10px] text-[var(--nx-accent-contrast)] font-medium transition"
                style={{ background: 'var(--nx-accent)' }}
              >
                {t('agents.create', 'Add')}
              </button>
            </div>
          </form>
        )}

        <div className="mt-1 space-y-1">
          {workspaces.map((ws) => {
            const isSelected = activeWorkspace === ws.path;
            return (
              <div
                key={ws.path}
                className="group flex items-center justify-between px-3 py-2 rounded-md text-xs transition"
                style={{
                  background: isSelected ? 'var(--nx-surface-2)' : 'transparent',
                  color: isSelected ? 'var(--nx-text)' : 'var(--nx-text-soft)',
                  borderLeft: isSelected ? '2px solid var(--nx-accent)' : '2px solid transparent',
                  fontWeight: isSelected ? 600 : 400,
                }}
              >
                <button
                  onClick={() => onSelectWorkspace(ws.path)}
                  className="flex-1 text-left truncate min-w-0"
                >
                  <div className="truncate font-medium">{ws.name}</div>
                  <div
                    className="truncate text-[10px] font-mono"
                    style={{ color: 'var(--nx-muted)' }}
                  >
                    {ws.path}
                  </div>
                </button>
                {onRemoveWorkspace && workspaces.length > 1 && (
                  <button
                    onClick={(e) => {
                      e.stopPropagation();
                      onRemoveWorkspace(ws.path);
                    }}
                    className="opacity-0 group-hover:opacity-100 p-1 rounded transition ml-1 hover:bg-[var(--nx-surface)] text-[var(--nx-danger)]"
                    title={t('projectManager.remove', 'Remove Project')}
                    aria-label={t('projectManager.remove', 'Remove Project')}
                  >
                    <Trash2 className="w-3.5 h-3.5" />
                  </button>
                )}
              </div>
            );
          })}
        </div>
      </div>

      {/* Footer Info */}
      <div
        className="p-3 text-[11px] font-mono flex flex-col space-y-1"
        style={{
          borderTop: '1px solid var(--nx-border)',
          background: 'var(--nx-surface)',
          color: 'var(--nx-text-soft)',
        }}
      >
        <div className="flex items-center justify-between">
          <a
            href="https://github.com/IAPro-Community"
            target="_blank"
            rel="noreferrer"
            className="hover:underline transition flex items-center space-x-1 text-[11px]"
            style={{ color: 'var(--nx-text-soft)' }}
          >
            <span>IAPro-Community</span>
            <ExternalLink className="w-3 h-3" />
          </a>
          <span style={{ color: 'var(--nx-success)' }}>● 127.0.0.1</span>
        </div>
        <div className="text-[10px]" style={{ color: 'var(--nx-muted)' }}>
          Agentic Software Engineering
        </div>
      </div>
    </aside>
  );
};
