import React, { useEffect, useState } from 'react';
import { EyeOff, PanelsTopLeft, ShieldCheck } from 'lucide-react';
import { Badge, Card, Segmented, Switch } from '../../design-system';
import {
  loadHudPreferences,
  saveHudPreferences,
  type HudMode,
  type HudPresentationPreferences,
} from './hudModel';

export const HudSettings: React.FC = () => {
  const [prefs, setPrefs] = useState<HudPresentationPreferences>(() => loadHudPreferences());

  useEffect(() => {
    const sync = () => setPrefs(loadHudPreferences());
    window.addEventListener('nexus:hud-preferences', sync);
    return () => window.removeEventListener('nexus:hud-preferences', sync);
  }, []);

  const patch = (changes: Partial<HudPresentationPreferences>) => {
    const next = { ...prefs, ...changes };
    setPrefs(next);
    saveHudPreferences(next);
  };

  return (
    <>
      <Card className="nx-settings-card">
        <div className="nx-settings-card__title">
          <PanelsTopLeft size={17} />
          <div><strong>Nexus Companion</strong><small>Escolha entre a cápsula atual e o guardião 2D. A fonte dos dados continua a mesma.</small></div>
        </div>
        <Segmented ariaLabel="Aparência do HUD" value={prefs.appearance}
          onChange={(value) => patch({ appearance: value === 'companion' ? 'companion' : 'bar' })}
          options={[{ value: 'bar', label: 'Barra' }, { value: 'companion', label: 'Mascote 2D' }]} />
        <Switch checked={prefs.companionMotion} onChange={(companionMotion) => patch({ companionMotion })}
          label="Microanimações" description="Desativadas automaticamente quando o sistema prefere menos movimento." />
        <Switch checked={prefs.attentionBubbles} onChange={(attentionBubbles) => patch({ attentionBubbles })}
          label="Balões de atenção" description="Apenas mensagens ligadas a intervenções e alertas reais, sem abrir páginas sozinho." />
      </Card>
      <Card className="nx-settings-card">
        <div className="nx-settings-card__title">
          <PanelsTopLeft size={17} />
          <div>
            <strong>Nexus HUD</strong>
            <small>
              Cápsula mínima e painel contextual. Não altera Capacity Monitor, ResourceScheduler,
              Agents, Missions ou Cost Intelligence.
            </small>
          </div>
          <Badge tone="warning">Experimental</Badge>
        </div>
        <Switch
          checked={prefs.enabled}
          onChange={(enabled) => patch({ enabled })}
          label="Exibir HUD"
          description="Mostra o HUD persistente no Nexus Web quando o rollout experimental estiver habilitado."
        />
        <Segmented
          ariaLabel="Informação prioritária ao expandir o Nexus HUD"
          value={prefs.mode}
          onChange={(value) => patch({ mode: value as HudMode })}
          options={[
            { value: 'AUTO', label: 'Auto' },
            { value: 'CAPACITY', label: 'Capacity' },
            { value: 'RESOURCE', label: 'Resource' },
            { value: 'AGENT', label: 'Agent' },
            { value: 'MISSION', label: 'Mission' },
            { value: 'ALERT', label: 'Alert' },
            { value: 'MINIMAL', label: 'Minimal' },
          ]}
        />
      </Card>

      <Card className="nx-settings-card">
        <div className="nx-settings-card__title">
          <PanelsTopLeft size={17} />
          <div>
            <strong>Posição no Web OS</strong>
            <small>Dock persistente ou widget flutuante dentro da viewport.</small>
          </div>
        </div>
        <Segmented
          ariaLabel="Superfície do Nexus HUD"
          value={prefs.surfaceMode}
          onChange={(value) => patch({ surfaceMode: value === 'floating' ? 'floating' : 'docked' })}
          options={[
            { value: 'docked', label: 'Dock' },
            { value: 'floating', label: 'Floating' },
          ]}
        />
        {prefs.surfaceMode === 'docked' ? (
          <Segmented
            ariaLabel="Posição do dock"
            value={prefs.dock}
            onChange={(value) => patch({ dock: value === 'top' ? 'top' : 'bottom' })}
            options={[
              { value: 'top', label: 'Topo' },
              { value: 'bottom', label: 'Rodapé' },
            ]}
          />
        ) : null}
      </Card>

      <Card className="nx-settings-card">
        <div className="nx-settings-card__title">
          <ShieldCheck size={17} />
          <div>
            <strong>Conteúdo</strong>
            <small>Escolha quais fontes canônicas podem aparecer na projeção.</small>
          </div>
        </div>
        <Switch
          checked={prefs.showCapacity}
          onChange={(showCapacity) => patch({ showCapacity })}
          label="Capacity"
        />
        <Switch
          checked={prefs.showRecommendation}
          onChange={(showRecommendation) => patch({ showRecommendation })}
          label="ResourceScheduler"
        />
        <Switch
          checked={prefs.showAgents}
          onChange={(showAgents) => patch({ showAgents })}
          label="Agents"
        />
        <Switch
          checked={prefs.showMissions}
          onChange={(showMissions) => patch({ showMissions })}
          label="Missions"
        />
        <Switch
          checked={prefs.showCost}
          onChange={(showCost) => patch({ showCost })}
          label="Cost Intelligence"
        />
      </Card>

      <Card className="nx-settings-card">
        <div className="nx-settings-card__title">
          <EyeOff size={17} />
          <div>
            <strong>Privacy Mode</strong>
            <small>
              Oculta nomes de Agents e custos na projeção para compartilhamento de tela e gravação.
            </small>
          </div>
        </div>
        <Switch
          checked={prefs.privacyMode}
          onChange={(privacyMode) => patch({ privacyMode })}
          label="Ativar Privacy Mode"
          description="Provider, capacidade e estado geral continuam visíveis."
        />
      </Card>
    </>
  );
};
