import React from 'react';
import { Plus, Sparkles, TerminalSquare } from 'lucide-react';
import { useTranslation } from 'react-i18next';

export type ProjectCreateActionsProps = {
  onNewAgent?: () => void;
  onNewAISession?: () => void;
  onProjectShell?: () => void;
  /** Compact fits the workspace tab strip; default is full overview-style labels. */
  size?: 'sm' | 'md';
  className?: string;
};

/** Global create CTAs — visible from any product tab (workspace chrome / topbar / rail). */
export const ProjectCreateActions: React.FC<ProjectCreateActionsProps> = ({
  onNewAgent,
  onNewAISession,
  onProjectShell,
  size = 'sm',
  className,
}) => {
  const { t } = useTranslation();
  if (!onNewAgent && !onNewAISession && !onProjectShell) return null;

  return (
    <div
      className={`nx-project-create-actions${className ? ` ${className}` : ''}`}
      data-size={size}
      role="group"
      aria-label={t('overview.createInProject', 'Criar no projeto')}
    >
      {onNewAgent && (
        <button
          type="button"
          className="nx-button"
          data-tone="brand"
          data-size="sm"
          onClick={onNewAgent}
          title={t('agents.createInProjectTooltip', 'Criar novo Agente no Projeto')}
        >
          <Plus size={size === 'sm' ? 12 : 14} />
          <span>{t('agents.new', 'Novo Agente')}</span>
        </button>
      )}
      {onNewAISession && (
        <button
          type="button"
          className="nx-button"
          data-size="sm"
          onClick={onNewAISession}
          title={t('overview.newAISession')}
        >
          <Sparkles size={size === 'sm' ? 12 : 14} />
          <span>{t('overview.newAISession')}</span>
        </button>
      )}
      {onProjectShell && (
        <button
          type="button"
          data-testid="overview-terminal-btn"
          className="nx-button nx-button--terminal"
          data-size="sm"
          onClick={onProjectShell}
          title={t('overview.projectShell')}
        >
          <TerminalSquare size={size === 'sm' ? 12 : 14} />
          <span>{t('overview.projectShell')}</span>
        </button>
      )}
    </div>
  );
};
