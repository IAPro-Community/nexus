import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import {
  Bot,
  Database,
  FileText,
  Layers,
  Palette,
  Play,
  Server,
  Shield,
  Terminal,
  Zap,
} from 'lucide-react';
import { Button, Card, Dialog, Input, Select } from '../../design-system';
import { nexus } from '../../nexus/api';
import { isRequiredResourceSelection } from '../../nexus/agentTerminalModel';
import type { Agent, AgentSpec, Project } from '../../types';
import styles from './NewAgentModal.module.scss';

export interface AgentTypePreset {
  id: string;
  name: string;
  role: string;
  description: string;
  icon: React.ComponentType<{ size?: number; className?: string }>;
  defaultProvider: string;
  defaultMode: 'Safe' | 'YOLO';
  recommendedSkills: string[];
  agentSpec: AgentSpec;
}

export const CANONICAL_AGENT_TYPES: AgentTypePreset[] = [
  {
    id: 'generalist',
    name: 'Generalist Engineer',
    role: 'generalist',
    description: 'Execução abrangente, tarefas multiárea e coordenação técnica.',
    icon: Bot,
    defaultProvider: 'claude',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-repo-health'],
    agentSpec: {
      role: 'generalist',
      instructions: [
        'Decomponha tarefas amplas antes de executar e coordene dependências com clareza.',
      ],
      responsibilities: ['Conectar decisões de produto, código, testes e documentação.'],
      capabilities: ['analysis', 'implementation', 'coordination'],
      domains: ['product', 'software-engineering'],
      strengths: ['systems-thinking', 'coordination'],
      tags: ['generalist', 'multi-area'],
      verification_policy: { require_evidence: true, require_tests: true },
    },
  },
  {
    id: 'backend-engineer',
    name: 'Backend Engineer',
    role: 'backend-engineer',
    description: 'APIs, microsserviços, banco de dados e arquitetura server-side.',
    icon: Server,
    defaultProvider: 'codex',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-database-migrations', 'skill-systematic-debugging'],
    agentSpec: {
      role: 'backend-engineer',
      instructions: [
        'Priorize contratos estáveis, validação de entrada e observabilidade no backend.',
      ],
      responsibilities: ['Projetar APIs, persistência e serviços server-side resilientes.'],
      capabilities: ['go', 'rest_api', 'database'],
      domains: ['backend', 'distributed-systems'],
      strengths: ['api-design', 'reliability'],
      tags: ['backend', 'server-side'],
      verification_policy: { require_evidence: true, require_tests: true },
    },
  },
  {
    id: 'frontend-engineer',
    name: 'Frontend Engineer',
    role: 'frontend-engineer',
    description: 'UI, design system, acessibilidade, estado e UX de produto.',
    icon: Palette,
    defaultProvider: 'claude',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-modern-ui-patterns', 'skill-frontend-ux-guardrails'],
    agentSpec: {
      role: 'frontend-engineer',
      instructions: [
        'Preserve acessibilidade, responsividade, i18n e os tokens visuais existentes.',
      ],
      responsibilities: ['Implementar interfaces React coesas e verificáveis.'],
      capabilities: ['react', 'typescript', 'accessibility', 'ui'],
      domains: ['frontend', 'web'],
      strengths: ['ux', 'component-design'],
      tags: ['frontend', 'react'],
      verification_policy: { require_evidence: true, require_tests: true },
    },
  },
  {
    id: 'qa-engineer',
    name: 'QA & Test Engineer',
    role: 'qa-engineer',
    description: 'Testes automatizados E2E, cobertura, regressão e smoke tests.',
    icon: Zap,
    defaultProvider: 'codex',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-webapp-testing', 'skill-verification-before-completion'],
    agentSpec: {
      role: 'qa-engineer',
      instructions: [
        'Reproduza falhas, cubra regressões e exija evidência observável antes de concluir.',
      ],
      responsibilities: ['Projetar testes unitários, integração, E2E e smoke tests.'],
      capabilities: ['testing', 'e2e', 'regression_analysis'],
      domains: ['quality', 'verification'],
      strengths: ['reproduction', 'regression-detection'],
      tags: ['qa', 'testing'],
      verification_policy: { require_evidence: true, require_tests: true },
    },
  },
  {
    id: 'code-reviewer',
    name: 'Code Reviewer',
    role: 'code-reviewer',
    description: 'Auditoria de diffs, segurança estática, conformidade e clean code.',
    icon: Shield,
    defaultProvider: 'claude',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-saas-security-scan', 'skill-quality-gate'],
    agentSpec: {
      role: 'code-reviewer',
      instructions: [
        'Procure riscos, regressões e claims sem evidência; classifique severidade objetivamente.',
      ],
      responsibilities: ['Revisar diffs, segurança, compatibilidade e qualidade de implementação.'],
      capabilities: ['code_review', 'security', 'risk_analysis'],
      domains: ['quality', 'security'],
      strengths: ['critical-reading', 'risk-classification'],
      tags: ['review', 'security'],
      verification_policy: { require_evidence: true, require_tests: true },
    },
  },
  {
    id: 'devops-release',
    name: 'DevOps & Release',
    role: 'devops-release',
    description: 'Pipelines CI/CD, Docker, packaging e confiabilidade operacional.',
    icon: Layers,
    defaultProvider: 'agy',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-release-engineering', 'skill-incident-response'],
    agentSpec: {
      role: 'devops-release',
      instructions: ['Priorize reprodutibilidade, rollback seguro e observabilidade operacional.'],
      responsibilities: ['Manter CI/CD, empacotamento, deploy e resposta operacional.'],
      capabilities: ['ci_cd', 'docker', 'release', 'operations'],
      domains: ['devops', 'release-engineering'],
      strengths: ['automation', 'rollback-planning'],
      tags: ['devops', 'release'],
      verification_policy: { require_evidence: true, require_tests: true },
    },
  },
  {
    id: 'data-analyst',
    name: 'Data Analyst',
    role: 'data-analyst',
    description: 'Modelagem analítica, consultas SQL e insights de dados.',
    icon: Database,
    defaultProvider: 'codex',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-unified-analytics'],
    agentSpec: {
      role: 'data-analyst',
      instructions: ['Declare a origem dos dados, incertezas e critérios usados nas conclusões.'],
      responsibilities: ['Modelar consultas, métricas e análises reproduzíveis.'],
      capabilities: ['sql', 'analytics', 'data_modeling'],
      domains: ['data', 'analytics'],
      strengths: ['measurement', 'evidence-based-analysis'],
      tags: ['data', 'analytics'],
      verification_policy: { require_evidence: true, require_tests: false },
    },
  },
  {
    id: 'docs-writer',
    name: 'Technical Writer',
    role: 'docs-writer',
    description: 'Documentação de arquitetura, manuais, ADRs e handoff.',
    icon: FileText,
    defaultProvider: 'claude',
    defaultMode: 'Safe',
    recommendedSkills: ['skill-adr', 'skill-deep-wiki'],
    agentSpec: {
      role: 'docs-writer',
      instructions: [
        'Escreva para o público correto, preserve histórico e diferencie fato de hipótese.',
      ],
      responsibilities: ['Produzir arquitetura, guias operacionais, ADRs e handoffs navegáveis.'],
      capabilities: ['technical_writing', 'architecture_docs', 'documentation'],
      domains: ['documentation', 'architecture'],
      strengths: ['clarity', 'knowledge-organization'],
      tags: ['docs', 'architecture'],
      verification_policy: { require_evidence: true, require_tests: false },
    },
  },
];

export const NATIVE_PROVIDERS = [
  { id: 'claude', label: 'Claude (nexus claude)', alias: 'nexus claude' },
  { id: 'codex', label: 'Codex (nexus codex)', alias: 'nexus codex' },
  { id: 'agy', label: 'AGY / OpenCode (nexus agy)', alias: 'nexus agy' },
  { id: 'gemini', label: 'Gemini (nexus gemini)', alias: 'nexus gemini' },
  { id: 'cursor', label: 'Cursor (nexus cursor)', alias: 'nexus cursor' },
];

export const NewAgentModal: React.FC<{
  open: boolean;
  onClose: () => void;
  project: Project;
  onCreated: (agent: Agent) => void;
}> = ({ open, onClose, project, onCreated }) => {
  const { t } = useTranslation();
  const [selectedType, setSelectedType] = useState<AgentTypePreset>(CANONICAL_AGENT_TYPES[1]);
  const [name, setName] = useState(CANONICAL_AGENT_TYPES[1].name);
  const [origin, setOrigin] = useState<'native' | 'custom'>('native');
  const [provider, setProvider] = useState(CANONICAL_AGENT_TYPES[1].defaultProvider);
  const [mode, setMode] = useState<'Safe' | 'YOLO'>('Safe');
  const [commandTemplate, setCommandTemplate] = useState(
    'docker exec -it -w "{cwd}" vpn-dev-workspace-terminal-1 opencode {args}',
  );
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const handleSelectType = (preset: AgentTypePreset) => {
    setSelectedType(preset);
    setName(preset.name);
    setProvider(preset.defaultProvider);
    setMode(preset.defaultMode);
  };

  const resolvedPreview =
    origin === 'native'
      ? `nexus ${provider}${mode === 'YOLO' ? ' --yolo' : ''}`
      : commandTemplate
          .replace('{cwd}', project.canonical_path || '/workspace')
          .replace('{args}', mode === 'YOLO' ? '--yolo' : '');

  const handleCreate = async () => {
    if (!name.trim()) return;
    setBusy(true);
    setError('');
    let created: Agent | undefined;
    try {
      // 1. Create persistent agent in database
      created = await nexus.createAgent(project.id, name.trim(), selectedType.role);

      // 2. Configure revision with provider & adapter options
      const extraArgs: string[] = [];
      if (mode === 'YOLO') extraArgs.push('--yolo');

      const configOptions: Record<string, any> = {
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
        agent_spec: selectedType.agentSpec,
        options: configOptions,
      });

      onCreated(created);
      onClose();
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (created && isRequiredResourceSelection(message)) {
        // The Agent is intentionally kept persistent. The terminal opens its
        // resource picker so the user can allocate an available provider.
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

        {/* 1. Escolha da Especialidade */}
        <div>
          <label className={styles.sectionLabel}>{t('agents.selectSpecialty')}</label>
          <div className={styles.presetsGrid}>
            {CANONICAL_AGENT_TYPES.map((preset) => {
              const Icon = preset.icon;
              const isSelected = selectedType.id === preset.id;
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
                    <strong className={styles.presetTitle}>{preset.name}</strong>
                    <small className={styles.presetDesc}>{preset.description}</small>
                  </div>
                </button>
              );
            })}
          </div>
        </div>

        {/* 2. Nome do Agente */}
        <div className={styles.fieldGroup}>
          <label className={styles.sectionLabel}>{t('agents.displayNameField')}</label>
          <Input value={name} onChange={setName} placeholder={t('agents.namePlaceholder')} />
        </div>

        {/* 3. Origem e Adaptador de Execução */}
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

        {/* 4. Modo Inicial */}
        <div className={styles.fieldGroup}>
          <label className={styles.sectionLabel}>{t('agents.operationalMode')}</label>
          <div className={styles.buttonRow}>
            {(['Safe', 'YOLO'] as const).map((m) => (
              <button
                key={m}
                type="button"
                className="nx-button"
                data-size="sm"
                data-tone={mode === m ? (m === 'YOLO' ? 'warning' : 'brand') : 'default'}
                onClick={() => setMode(m)}
              >
                {m === 'YOLO' ? t('agents.modeYolo') : t('agents.modeSafe')}
              </button>
            ))}
          </div>
        </div>

        {/* 5. Preview do Comando Resolvido */}
        <div className={styles.previewBox}>
          <div className={styles.previewHeader}>
            <Terminal size={12} />
            <span>{t('agents.finalCommand')}</span>
          </div>
          <code className={styles.previewCode}>{resolvedPreview}</code>
        </div>

        {/* Ações */}
        <div className={styles.actionsRow}>
          <Button onClick={onClose} disabled={busy}>
            {t('common.cancel')}
          </Button>
          <Button tone="brand" onClick={handleCreate} disabled={busy || !name.trim()}>
            <Play size={13} /> {busy ? t('agents.creating') : t('agents.createAndOpenTerminal')}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
