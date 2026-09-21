import React, { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Bot,
  Database,
  FileText,
  Layers,
  Palette,
  Play,
  Search,
  Server,
  Shield,
  Terminal,
  Zap,
} from 'lucide-react';
import { Button, Card, Dialog, Input, Select } from '../../design-system';
import { nexus } from '../../nexus/api';
import { isRequiredResourceSelection } from '../../nexus/agentTerminalModel';
import type { Agent, AgentMode, Project } from '../../types';
import styles from './NewAgentModal.module.scss';

type ModePresentation = {
  name: string;
  description: string;
  icon: React.ComponentType<{ size?: number; className?: string }>;
};

// Behavioral specialization is owned by the Go Agent Mode catalog. This map is
// presentation-only: changing it cannot alter AgentSpec or execution policy.
export const AGENT_MODE_PRESENTATION: Record<string, ModePresentation> = {
  developer: {
    name: 'Developer',
    description: 'Implementação, correções e evolução incremental com testes.',
    icon: Bot,
  },
  'senior-developer': {
    name: 'Senior Developer',
    description: 'Mudanças complexas, refactors cross-cutting e decisões técnicas de alto impacto.',
    icon: Bot,
  },
  reviewer: {
    name: 'Reviewer',
    description: 'Revisão crítica de diffs, regressões, evidências e qualidade.',
    icon: Shield,
  },
  investigator: {
    name: 'Investigator',
    description:
      'Causa raiz, comportamento intermitente, logs e diagnóstico orientado por evidência.',
    icon: Search,
  },
  architect: {
    name: 'Architect',
    description: 'Arquitetura, contratos, trade-offs, ADRs e evolução estrutural.',
    icon: Layers,
  },
  'security-reviewer': {
    name: 'Security Reviewer',
    description: 'Threat modeling, autenticação, autorização, secrets e superfícies de ataque.',
    icon: Shield,
  },
  'qa-engineer': {
    name: 'QA Engineer',
    description: 'Testes automatizados E2E, cobertura, regressão e smoke tests.',
    icon: Zap,
  },
  'performance-investigator': {
    name: 'Performance Investigator',
    description: 'Profiling, latência, throughput, memória e regressões de desempenho.',
    icon: Zap,
  },
  'documentation-specialist': {
    name: 'Documentation Specialist',
    description: 'Documentação de arquitetura, manuais, ADRs, guias e handoff.',
    icon: FileText,
  },
  generalist: {
    name: 'Generalist Engineer',
    description: 'Execução abrangente, tarefas multiárea e coordenação técnica.',
    icon: Bot,
  },
  'backend-engineer': {
    name: 'Backend Engineer',
    description: 'APIs, microsserviços, banco de dados e arquitetura server-side.',
    icon: Server,
  },
  'frontend-engineer': {
    name: 'Frontend Engineer',
    description: 'UI, design system, acessibilidade, estado e UX de produto.',
    icon: Palette,
  },
  'devops-release': {
    name: 'DevOps & Release',
    description: 'Pipelines CI/CD, Docker, packaging e confiabilidade operacional.',
    icon: Layers,
  },
  'data-analyst': {
    name: 'Data Analyst',
    description: 'Modelagem analítica, consultas SQL e insights de dados.',
    icon: Database,
  },
};

export const NATIVE_PROVIDERS = [
  { id: 'claude', label: 'Claude (nexus claude)', alias: 'nexus claude' },
  { id: 'codex', label: 'Codex (nexus codex)', alias: 'nexus codex' },
  { id: 'agy', label: 'AGY / OpenCode (nexus agy)', alias: 'nexus agy' },
  { id: 'gemini', label: 'Gemini (nexus gemini)', alias: 'nexus gemini' },
  { id: 'cursor', label: 'Cursor (nexus cursor)', alias: 'nexus cursor' },
];

function presentationFor(mode: AgentMode): ModePresentation {
  return (
    AGENT_MODE_PRESENTATION[mode.id] ?? {
      name: mode.id
        .split('-')
        .filter(Boolean)
        .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
        .join(' '),
      description: mode.role,
      icon: Bot,
    }
  );
}

function executionModeFor(mode: AgentMode): 'Safe' | 'YOLO' {
  if (mode.read_only_by_default) return 'Safe';
  return mode.default_execution_mode === 'YOLO' ? 'YOLO' : 'Safe';
}

export const NewAgentModal: React.FC<{
  open: boolean;
  onClose: () => void;
  project: Project;
  onCreated: (agent: Agent) => void;
}> = ({ open, onClose, project, onCreated }) => {
  const { t } = useTranslation();
  const [agentModes, setAgentModes] = useState<AgentMode[]>([]);
  const [selectedType, setSelectedType] = useState<AgentMode | null>(null);
  const [name, setName] = useState('');
  const [origin, setOrigin] = useState<'native' | 'custom'>('native');
  const [provider, setProvider] = useState('codex');
  const [mode, setMode] = useState<'Safe' | 'YOLO'>('Safe');
  const [commandTemplate, setCommandTemplate] = useState(
    'docker exec -it -w "{cwd}" vpn-dev-workspace-terminal-1 opencode {args}',
  );
  const [busy, setBusy] = useState(false);
  const [loadingModes, setLoadingModes] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!open) return;
    let alive = true;
    setLoadingModes(true);
    setError('');
    void nexus
      .listAgentModes()
      .then((modes) => {
        if (!alive) return;
        if (modes.length === 0) {
          throw new Error('Nexus returned an empty Agent Mode catalog.');
        }
        setAgentModes(modes);
        const initial =
          modes.find((candidate) => candidate.id === 'developer') ??
          modes.find((candidate) => candidate.id === 'backend-engineer') ??
          modes[0];
        setSelectedType(initial);
        setName(presentationFor(initial).name);
        setProvider(initial.default_provider || 'codex');
        setMode(executionModeFor(initial));
      })
      .catch((err: unknown) => {
        if (!alive) return;
        setAgentModes([]);
        setSelectedType(null);
        setError(
          err instanceof Error
            ? `Não foi possível carregar os Agent Modes: ${err.message}`
            : 'Não foi possível carregar os Agent Modes.',
        );
      })
      .finally(() => {
        if (alive) setLoadingModes(false);
      });
    return () => {
      alive = false;
    };
  }, [open]);

  const handleSelectType = (preset: AgentMode) => {
    setSelectedType(preset);
    setName(presentationFor(preset).name);
    setProvider(preset.default_provider || 'codex');
    setMode(executionModeFor(preset));
  };

  const resolvedPreview =
    origin === 'native'
      ? `nexus ${provider}${mode === 'YOLO' ? ' --yolo' : ''}`
      : commandTemplate
          .replace('{cwd}', project.canonical_path || '/workspace')
          .replace('{args}', mode === 'YOLO' ? '--yolo' : '');

  const handleCreate = async () => {
    if (!name.trim() || !selectedType) return;
    setBusy(true);
    setError('');
    let created: Agent | undefined;
    try {
      created = await nexus.createAgent(project.id, name.trim(), selectedType.role);

      const extraArgs: string[] = [];
      if (mode === 'YOLO') extraArgs.push('--yolo');

      const configOptions: Record<string, unknown> = {
        agent_mode_id: selectedType.id,
        read_only_by_default: Boolean(selectedType.read_only_by_default),
        role_type: selectedType.role,
        origin,
        mode,
      };
      if (extraArgs.length > 0) {
        configOptions.extra_args = extraArgs;
      }
      if (origin === 'custom') {
        configOptions.command_template = commandTemplate;
        configOptions.execution_adapter = 'command_template';
      } else {
        configOptions.execution_adapter = 'native_alias';
      }

      await nexus.applyAgentConfig(created.id, {
        provider: origin === 'custom' ? 'opencode' : provider,
        profile: 'default',
        isolation: 'project',
        workspace: project.canonical_path || undefined,
        agent_spec: selectedType.agent_spec,
        options: configOptions,
      });

      onCreated(created);
      onClose();
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (created && isRequiredResourceSelection(message)) {
        onCreated(created);
        onClose();
        return;
      }
      setError(message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onClose={onClose} title={t('agents.newSpecialistAgent')} wide>
      <div className={styles.modalBody}>
        {error && <Card className="nx-inline-error">{error}</Card>}

        <div>
          <label className={styles.sectionLabel}>{t('agents.selectSpecialty')}</label>
          <div className={styles.presetsGrid}>
            {agentModes.map((preset) => {
              const presentation = presentationFor(preset);
              const Icon = presentation.icon;
              const isSelected = selectedType?.id === preset.id;
              return (
                <button
                  type="button"
                  key={preset.id}
                  onClick={() => handleSelectType(preset)}
                  className={styles.presetButton}
                  data-selected={isSelected ? 'true' : 'false'}
                >
                  <span className={styles.presetIconWrap}>
                    <Icon size={15} />
                  </span>
                  <div className={styles.presetContent}>
                    <strong className={styles.presetTitle}>{presentation.name}</strong>
                    <small className={styles.presetDesc}>{presentation.description}</small>
                  </div>
                </button>
              );
            })}
          </div>
          {loadingModes && <small className={styles.hintText}>Carregando especialidades…</small>}
        </div>

        <div className={styles.fieldGroup}>
          <label className={styles.sectionLabel}>{t('agents.displayNameField')}</label>
          <Input value={name} onChange={setName} placeholder={t('agents.namePlaceholder')} />
        </div>

        <div className={styles.fieldGroup}>
          <label className={styles.sectionLabel}>{t('agents.executionOrigin')}</label>
          <div className={styles.buttonRow}>
            <button
              type="button"
              className="nx-button"
              data-tone={origin === 'native' ? 'brand' : 'default'}
              data-size="sm"
              onClick={() => setOrigin('native')}
            >
              {t('agents.originNative')}
            </button>
            <button
              type="button"
              className="nx-button"
              data-tone={origin === 'custom' ? 'brand' : 'default'}
              data-size="sm"
              onClick={() => setOrigin('custom')}
            >
              {t('agents.originCustom')}
            </button>
          </div>

          {origin === 'native' ? (
            <div className={styles.fieldGroup}>
              <label className={styles.fieldSublabel}>{t('agents.providerAndAlias')}</label>
              <Select
                value={provider}
                onChange={(val) => setProvider(val)}
                options={NATIVE_PROVIDERS.map((p) => ({
                  value: p.id,
                  label: p.label,
                }))}
              />
            </div>
          ) : (
            <div className={styles.fieldGroup}>
              <label className={styles.fieldSublabel}>{t('agents.commandTemplateLabel')}</label>
              <Input
                value={commandTemplate}
                onChange={setCommandTemplate}
                data-mono="true"
                placeholder='docker exec -it -w "{cwd}" vpn-dev-workspace-terminal-1 opencode {args}'
              />
              <small className={styles.hintText}>{t('agents.commandTemplateHint')}</small>
            </div>
          )}
        </div>

        <div className={styles.fieldGroup}>
          <label className={styles.sectionLabel}>{t('agents.operationalMode')}</label>
          <div className={styles.buttonRow}>
            {(['Safe', 'YOLO'] as const).map((candidateMode) => {
              const disabled =
                candidateMode === 'YOLO' && Boolean(selectedType?.read_only_by_default);
              return (
                <button
                  key={candidateMode}
                  type="button"
                  className="nx-button"
                  data-size="sm"
                  data-tone={
                    mode === candidateMode
                      ? candidateMode === 'YOLO'
                        ? 'warning'
                        : 'brand'
                      : 'default'
                  }
                  disabled={disabled}
                  onClick={() => setMode(candidateMode)}
                >
                  {candidateMode === 'YOLO' ? t('agents.modeYolo') : t('agents.modeSafe')}
                </button>
              );
            })}
          </div>
          {selectedType?.read_only_by_default && (
            <small className={styles.hintText}>
              Esta especialidade é read-only por padrão; alterações devem ser delegadas a uma tarefa
              de implementação.
            </small>
          )}
        </div>

        <div className={styles.previewBox}>
          <div className={styles.previewHeader}>
            <Terminal size={12} />
            <span>{t('agents.finalCommand')}</span>
          </div>
          <code className={styles.previewCode}>{resolvedPreview}</code>
        </div>

        <div className={styles.actionsRow}>
          <Button onClick={onClose} disabled={busy}>
            {t('common.cancel')}
          </Button>
          <Button
            tone="brand"
            onClick={handleCreate}
            disabled={busy || loadingModes || !selectedType || !name.trim()}
          >
            <Play size={13} /> {busy ? t('agents.creating') : t('agents.createAndOpenTerminal')}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
