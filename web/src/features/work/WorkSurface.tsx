import React, { useEffect, useMemo, useState } from 'react';
import { BrainCircuit, Play, Route, Sparkles, TerminalSquare } from 'lucide-react';
import { Badge, Button, Card, Segmented, Textarea } from '../../design-system';
import type { Agent, Project } from '../../types';
import { useTranslation } from 'react-i18next';
import { PlanBuilderSurface } from './PlanBuilderSurface';
import { canPlan, type ContextReadiness } from './composerModel';
import { nexus } from '../../nexus/api';

export const WorkSurface: React.FC<{ project: Project; agents: Agent[]; onDirect: (agent: Agent) => void; onStartSession?: (mode: 'direct' | 'assisted', prompt: string) => void; onPlan: () => void; onMaestro: () => void }> = ({ project, agents, onDirect, onStartSession, onPlan, onMaestro }) => {
  const { t } = useTranslation();
  const [mode, setMode] = useState('direct');
  const [prompt, setPrompt] = useState('');
  const [readiness, setReadiness] = useState<ContextReadiness>('MISSING');
  const [contextError, setContextError] = useState('');
  const preferred = useMemo(() => agents.find((agent) => agent.status === 'WORKING') ?? agents[0], [agents]);

  useEffect(() => {
    let live = true;
    void nexus.getContextReadiness(project.id)
      .then((value) => live && setReadiness(value.state || 'MISSING'))
      .catch((error) => live && setContextError(error instanceof Error ? error.message : String(error)));
    return () => { live = false; };
  }, [project.id]);

  const handleModeChange = (newMode: string) => {
    if (newMode === 'planned' && !canPlan(readiness)) return;
    setMode(newMode);
    if (newMode === 'planned' && onPlan) {
      onPlan();
    }
  };

  if (mode === 'planned') {
    return (
      <div className="nx-surface-scroll">
        <div style={{ padding: '0 24px 16px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <Segmented
            ariaLabel={t('work.mode')}
            value={mode}
            onChange={handleModeChange}
            options={[
              { value: 'direct', label: t('work.direct') },
              { value: 'assisted', label: t('work.assisted') },
              { value: 'planned', label: t('work.planned') },
            ]}
          />
        </div>
        <PlanBuilderSurface project={project} agents={agents} onOpenAgent={onDirect} />
      </div>
    );
  }

  return (
    <div className="nx-surface-scroll nx-work-surface">
      <div className="nx-page-header">
        <div>
          <span className="nx-eyebrow">{t('work.eyebrow')}</span>
          <h1>{t('work.title')}</h1>
          <p>{t('work.intro')}</p>
        </div>
        <Badge tone="brand">{project.name}</Badge>
      </div>

      <Card className="nx-work-composer">
        <div role="status" style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 8, marginBottom: 10 }}>
          <span>Context <strong>{readiness}</strong>{contextError ? ` · ${contextError}` : ''}</span>
          {readiness !== 'READY' && <Button size="sm" onClick={() => { setContextError(''); void nexus.hydrateContext(project.id).then((value) => setReadiness(value.state || 'HYDRATING')).catch((error) => setContextError(error instanceof Error ? error.message : String(error))); }}>{readiness === 'MISSING' || readiness === 'STALE' ? 'Hydrate' : 'Retry'}</Button>}
        </div>
        <div className="nx-work-composer__top">
          <Segmented
            ariaLabel={t('work.mode')}
            value={mode}
            onChange={handleModeChange}
            options={[
              { value: 'direct', label: t('work.direct') },
              { value: 'assisted', label: t('work.assisted') },
              { value: 'planned', label: t('work.planned') },
            ]}
          />
          <span className="nx-muted-copy">
            {mode === 'direct' ? t('work.directHint') : mode === 'assisted' ? t('work.assistedHint') : t('work.plannedHint')}
          </span>
        </div>

        <Textarea rows={8} value={prompt} onChange={setPrompt} placeholder={t('work.placeholder')} />

        <div className="nx-work-composer__actions">
          <Button onClick={onMaestro}>
            <BrainCircuit size={14} /> {t('work.askMaestro')}
          </Button>
          <Button tone="brand" disabled={!prompt.trim() || (!onStartSession && !preferred)} onClick={() => {
            if (onStartSession && (mode === 'direct' || mode === 'assisted')) onStartSession(mode, prompt.trim());
            else if (preferred) onDirect(preferred);
          }}>
            <Play size={14} /> {mode === 'direct' ? 'Start direct AI session' : 'Start assisted AI session'}
          </Button>
        </div>
      </Card>

      <div className="nx-work-cards">
        <Card onClick={() => handleModeChange('direct')} style={{ cursor: 'pointer' }}>
          <span className="nx-work-card-icon">
            <TerminalSquare size={17} />
          </span>
          <strong>{t('work.direct')}</strong>
          <p>{t('work.directDescription')}</p>
        </Card>
        <Card onClick={() => handleModeChange('assisted')} style={{ cursor: 'pointer' }}>
          <span className="nx-work-card-icon">
            <Sparkles size={17} />
          </span>
          <strong>{t('work.assisted')}</strong>
          <p>{t('work.assistedDescription')}</p>
        </Card>
        <Card onClick={() => handleModeChange('planned')} style={{ cursor: 'pointer' }}>
          <span className="nx-work-card-icon">
            <Route size={17} />
          </span>
          <strong>{t('work.planned')}</strong>
          <p>{t('work.plannedDescription')}</p>
        </Card>
      </div>
    </div>
  );
};
