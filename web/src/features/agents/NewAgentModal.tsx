import React, { useCallback, useEffect, useState } from 'react';
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
import { api } from '../../api';
import {
  bindStructuredHarnessProfile,
  structuredProtocolsForProvider,
  type StructuredHarnessProtocol,
} from './structuredHarnessOptions';
import { isRequiredResourceSelection } from '../../nexus/agentTerminalModel';
import { withAgentExecutionMode } from '../../nexus/agentExecutionMode';
import type {
  Agent,
  AgentConfig,
  AgentMode,
  ProfileInfo,
  Project,
  ProviderInfo,
} from '../../types';
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
  { id: 'agy', label: 'AGY (nexus agy)', alias: 'nexus agy' },
  { id: 'opencode', label: 'OpenCode (nexus opencode)', alias: 'nexus opencode' },
  { id: 'gemini', label: 'Gemini (nexus gemini)', alias: 'nexus gemini' },
  { id: 'cursor', label: 'Cursor (nexus cursor)', alias: 'nexus cursor' },
];

const CUSTOM_TARGET_KINDS = [
  { id: 'docker', label: 'Docker / VPN Workspace' },
  { id: 'docker-compose', label: 'Docker Compose' },
  { id: 'wsl', label: 'WSL' },
  { id: 'ssh', label: 'SSH / Remote wrapper' },
  { id: 'custom', label: 'Custom executable / wrapper' },
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

export function buildNewAgentConfig(
  selectedType: AgentMode,
  preferredProvider: string,
  workspace: string | undefined,
  options: Record<string, unknown>,
): AgentConfig {
  return {
    provider: '',
    profile: '',
    isolation: 'project',
    workspace,
    allocation: preferredProvider ? { prefer_provider: preferredProvider } : undefined,
    agent_spec: selectedType.agent_spec,
    options,
  };
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
  const [providerDiscovery, setProviderDiscovery] = useState<ProviderInfo[]>([]);
  const [availableProfiles, setAvailableProfiles] = useState<ProfileInfo[]>([]);
  const [structuredProfile, setStructuredProfile] = useState('');
  const [agentProtocol, setAgentProtocol] = useState<StructuredHarnessProtocol>('');
  const [mode, setMode] = useState<'Safe' | 'YOLO'>('Safe');
  const [targetKind, setTargetKind] = useState('docker');
  const [targetWorkspace, setTargetWorkspace] = useState('');
  const [targetProfileHome, setTargetProfileHome] = useState('');
  const [targetProbeTemplate, setTargetProbeTemplate] = useState(
    'docker exec vpn-dev-workspace-terminal-1 {provider} {args}',
  );
  const [commandTemplate, setCommandTemplate] = useState(
    'docker exec -it -w "{target_cwd}" vpn-dev-workspace-terminal-1 {provider} {args}',
  );
  const [busy, setBusy] = useState(false);
  const [loadingModes, setLoadingModes] = useState(false);
  const [error, setError] = useState('');

  const localizedPresentationFor = useCallback(
    (agentMode: AgentMode): ModePresentation => {
      const fallback = presentationFor(agentMode);
      return {
        ...fallback,
        name: String(
          t(`agents.agentModes.${agentMode.id}.name`, {
            defaultValue: fallback.name,
          }),
        ),
        description: String(
          t(`agents.agentModes.${agentMode.id}.description`, {
            defaultValue: fallback.description,
          }),
        ),
      };
    },
    [t],
  );

  useEffect(() => {
    if (!open) return;
    let alive = true;
    setLoadingModes(true);
    setError('');
    setAgentProtocol('');
    setProviderDiscovery([]);
    setAvailableProfiles([]);
    setStructuredProfile('');
    void api
      .getProfiles()
      .then((profiles) => {
        if (alive) setAvailableProfiles(profiles);
      })
      .catch(() => {
        if (alive) setAvailableProfiles([]);
      });
    void api
      .getProviders()
      .then((providers) => {
        if (!alive) return;
        // The Core only registers experimental harnesses when its opt-in flag
        // is enabled. A failed probe must not advertise preview support.
        setProviderDiscovery(providers);
      })
      .catch(() => {
        if (alive) setProviderDiscovery([]);
      });
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
        setName(localizedPresentationFor(initial).name);
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
  }, [open, localizedPresentationFor]);

  const handleSelectType = (preset: AgentMode) => {
    setSelectedType(preset);
    setName(localizedPresentationFor(preset).name);
    setProvider(preset.default_provider || 'codex');
    setAgentProtocol('');
    setStructuredProfile('');
    setMode(executionModeFor(preset));
  };

  const installedProviders = providerDiscovery
    .filter((entry) => entry.installed)
    .map((entry) => entry.id);
  const structuredChoices = structuredProtocolsForProvider(
    providerDiscovery.find((entry) => entry.id === provider),
  );
  const structuredProfiles = availableProfiles.filter((entry) => entry.provider === provider);
  const validStructuredProfile = structuredProfiles.some(
    (entry) => entry.name === structuredProfile && entry.authenticated,
  );
  const resolvedPreview =
    origin === 'native'
      ? `nexus ${provider}${agentProtocol ? ` [${agentProtocol.toUpperCase()} experimental]` : mode === 'YOLO' ? ' --yolo' : ''}`
      : commandTemplate
          .replaceAll('{cwd}', project.canonical_path || '/workspace')
          .replaceAll(
            '{target_cwd}',
            targetWorkspace.trim() || project.canonical_path || '/workspace',
          )
          .replaceAll('{provider}', provider)
          .replaceAll('{profile}', 'default')
          .replaceAll('{profile_home}', targetProfileHome || '<shared-profile-home>')
          .replace('{args}', mode === 'YOLO' ? '--yolo' : '');

  const handleCreate = async () => {
    if (!name.trim() || !selectedType) return;
    if (agentProtocol && (!structuredChoices.includes(agentProtocol) || !validStructuredProfile)) {
      setError(t('agents.structuredProfileError'));
      return;
    }
    setBusy(true);
    setError('');
    let created: Agent | undefined;
    try {
      created = await nexus.createAgent(project.id, name.trim(), selectedType.role);

      const configOptions = withAgentExecutionMode(
        {
          agent_mode_id: selectedType.id,
          read_only_by_default: Boolean(selectedType.read_only_by_default),
          role_type: selectedType.role,
          origin,
        },
        mode,
      );
      if (origin === 'custom') {
        configOptions.command_template = commandTemplate;
        configOptions.execution_adapter = 'command_template';
        configOptions.target_kind = targetKind;
        if (targetWorkspace.trim()) {
          configOptions.target_workspace = targetWorkspace.trim();
        }
        if (targetProfileHome.trim()) {
          configOptions.target_profile_home = targetProfileHome.trim();
        }
        if (targetProbeTemplate.trim()) {
          configOptions.target_probe_template = targetProbeTemplate.trim();
        }
      } else {
        configOptions.execution_adapter = 'native_alias';
        if (agentProtocol && structuredChoices.includes(agentProtocol)) {
          // Keep protocol choice in the existing AgentConfig -> LaunchOptions path.
          // The launcher performs the final feature-gate and provider validation.
          configOptions.agent_protocol = agentProtocol;
        }
      }

      const config = buildNewAgentConfig(
        selectedType,
        provider,
        project.canonical_path || undefined,
        configOptions,
      );
      const finalConfig = agentProtocol
        ? bindStructuredHarnessProfile(config, provider, structuredProfile)
        : config;
      await nexus.applyAgentConfig(created.id, finalConfig);

      onCreated(created);
      onClose();
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      if (created && isRequiredResourceSelection(message)) {
        onCreated(created);
        onClose();
        return;
      }
      if (created) {
        try {
          await nexus.deleteAgent(created.id);
        } catch {
          // Preserve the original configuration error. The agent surface can
          // still remove the orphan explicitly if rollback itself fails.
        }
      }
      setError(message);
    } finally {
      setBusy(false);
    }
  };

  const selectedPresentation = selectedType ? localizedPresentationFor(selectedType) : null;
  const SelectedIcon = selectedPresentation?.icon ?? Bot;

  return (
    <Dialog open={open} onClose={onClose} title={t('agents.newSpecialistAgent')} wide>
      <div className={styles.modalBody}>
        {error && <Card className="nx-inline-error">{error}</Card>}

        <div>
          <label className={styles.sectionLabel}>{t('agents.selectSpecialty')}</label>
          <div className={styles.presetsGrid}>
            {agentModes.map((preset) => {
              const presentation = localizedPresentationFor(preset);
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
          {loadingModes && (
            <small className={styles.hintText}>{t('agents.agentModesLoading')}</small>
          )}
        </div>

        <section className={styles.summaryCard} aria-labelledby="new-agent-summary-title">
          <header className={styles.summaryHeader}>
            <span className={styles.summaryIcon} aria-hidden="true">
              {selectedPresentation ? <SelectedIcon size={16} /> : <Bot size={16} />}
            </span>
            <div className={styles.summaryHeading}>
              <h3 id="new-agent-summary-title" className={styles.summaryTitle}>
                {t('agents.summaryTitle', 'Resumo')}
              </h3>
              <p className={styles.summarySubtitle}>
                {selectedPresentation?.description ??
                  t('agents.summaryEmpty', 'Escolha uma especialidade para começar.')}
              </p>
            </div>
          </header>

          <div className={styles.summaryGrid}>
            <div className={styles.summaryField}>
              <label className={styles.summaryFieldLabel} htmlFor="new-agent-name">
                {t('agents.displayNameField')}
              </label>
              <Input
                id="new-agent-name"
                value={name}
                onChange={setName}
                placeholder={t('agents.namePlaceholder')}
              />
            </div>
            <div className={styles.summaryField}>
              <label className={styles.summaryFieldLabel} htmlFor="new-agent-provider">
                {t('agents.providerAndAlias')}
              </label>
              <Select
                id="new-agent-provider"
                value={provider}
                onChange={(val) => {
                  setProvider(val);
                  setAgentProtocol('');
                  setStructuredProfile('');
                }}
                options={[
                  ...NATIVE_PROVIDERS.map((p) => ({ value: p.id, label: p.label })),
                  ...(['pi', 'omp', 'mimo', 'dsh'] as const)
                    .filter((id) => installedProviders.includes(id))
                    .map((id) => ({ value: id, label: `${id.toUpperCase()} (experimental)` })),
                ]}
              />
            </div>
          </div>

          <div className={styles.summaryRow}>
            <div className={styles.summaryChoice}>
              <span className={styles.summaryChoiceLabel}>{t('agents.executionOrigin')}</span>
              <div
                className={styles.buttonRow}
                role="group"
                aria-label={t('agents.executionOrigin')}
              >
                <button
                  type="button"
                  className="nx-button"
                  data-tone={origin === 'native' ? 'brand' : 'default'}
                  data-size="sm"
                  aria-pressed={origin === 'native'}
                  onClick={() => setOrigin('native')}
                >
                  {t('agents.originNative')}
                </button>
                <button
                  type="button"
                  className="nx-button"
                  data-tone={origin === 'custom' ? 'brand' : 'default'}
                  data-size="sm"
                  aria-pressed={origin === 'custom'}
                  onClick={() => {
                    setOrigin('custom');
                    setAgentProtocol('');
                  }}
                >
                  {t('agents.originCustom')}
                </button>
              </div>
            </div>

            <div className={styles.summaryChoice}>
              <span className={styles.summaryChoiceLabel}>{t('agents.operationalMode')}</span>
              <div
                className={styles.buttonRow}
                role="group"
                aria-label={t('agents.operationalMode')}
              >
                {(['Safe', 'YOLO'] as const).map((candidateMode) => {
                  const disabled =
                    candidateMode === 'YOLO' &&
                    (Boolean(selectedType?.read_only_by_default) || Boolean(agentProtocol));
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
                      aria-pressed={mode === candidateMode}
                      disabled={disabled}
                      onClick={() => setMode(candidateMode)}
                    >
                      {candidateMode === 'YOLO' ? t('agents.modeYolo') : t('agents.modeSafe')}
                    </button>
                  );
                })}
              </div>
            </div>
          </div>

          {selectedType?.read_only_by_default && (
            <small className={styles.hintText}>{t('agents.agentModeReadOnlyHint')}</small>
          )}

          <div className={styles.previewBox}>
            <div className={styles.previewHeader}>
              <Terminal size={12} />
              <span>{t('agents.finalCommand')}</span>
            </div>
            <code className={styles.previewCode}>{resolvedPreview}</code>
          </div>
        </section>

        <details className={styles.advanced}>
          <summary className={styles.advancedSummary}>
            <span>{t('agents.advancedTitle', 'Avançado')}</span>
            <small>
              {t('agents.advancedHint', 'Destino personalizado, templates e ponte de perfil.')}
            </small>
          </summary>
          <div className={styles.advancedBody}>
            {origin === 'native' ? (
              <div className={styles.fieldGroup}>
                <p className={styles.hintText}>
                  {t(
                    'agents.advancedNativeHint',
                    'A execução nativa não precisa de ajustes. Escolha “Personalizado” para usar Docker, WSL, SSH ou um wrapper.',
                  )}
                </p>
                {structuredChoices.length > 0 && (
                  <>
                    <label className={styles.fieldSublabel} htmlFor="new-agent-protocol">
                      {t('agents.structuredProtocol')}
                    </label>
                    <Select
                      id="new-agent-protocol"
                      value={agentProtocol}
                      onChange={(value) => {
                        setAgentProtocol(value as StructuredHarnessProtocol);
                        if (value) {
                          setMode('Safe');
                          setStructuredProfile(
                            structuredProfiles.find((entry) => entry.is_default)?.name ||
                              structuredProfiles[0]?.name ||
                              '',
                          );
                        } else {
                          setStructuredProfile('');
                        }
                      }}
                      options={[
                        { value: '', label: t('agents.structuredProtocolPTY') },
                        ...structuredChoices.map((value) => ({
                          value,
                          label:
                            value === 'acp' ? t('agents.structuredProtocolACP') : t('agents.structuredProtocolRPC'),
                        })),
                      ]}
                    />
                    {agentProtocol && (
                      <>
                        <label className={styles.fieldSublabel} htmlFor="new-agent-profile">
                          {t('agents.structuredProfile')}
                        </label>
                        <Select
                          id="new-agent-profile"
                          value={structuredProfile}
                          onChange={setStructuredProfile}
                          options={[
                            { value: '', label: t('agents.structuredProfileChoose') },
                            ...structuredProfiles.map((entry) => ({
                              value: entry.name,
                              label: `${entry.name}${entry.authenticated ? '' : t('agents.structuredProfileAuthRequired')}`,
                            })),
                          ]}
                        />
                      </>
                    )}
                    <small className={styles.hintText}>
                      {t('agents.structuredProtocolWarning')}
                    </small>
                  </>
                )}
              </div>
            ) : (
              <div className={styles.fieldGroup}>
                <label className={styles.fieldSublabel}>
                  {t('agents.customTargetKind', { defaultValue: 'Execution target' })}
                </label>
                <Select
                  value={targetKind}
                  onChange={(val) => setTargetKind(val)}
                  options={CUSTOM_TARGET_KINDS.map((target) => ({
                    value: target.id,
                    label: target.label,
                  }))}
                />

                <label className={styles.fieldSublabel}>
                  {t('agents.targetWorkspace', {
                    defaultValue: 'Workspace path inside target (optional)',
                  })}
                </label>
                <Input
                  value={targetWorkspace}
                  onChange={setTargetWorkspace}
                  data-mono="true"
                  placeholder="/workspace"
                />
                <small className={styles.hintText}>
                  {t('agents.targetWorkspaceHint', {
                    defaultValue:
                      'Use when the target sees the project at a different path. For VPN Dev Workspace on Windows use /workspace. {target_cwd} falls back to the Nexus project path when empty.',
                  })}
                </small>

                <label className={styles.fieldSublabel}>{t('agents.commandTemplateLabel')}</label>
                <Input
                  value={commandTemplate}
                  onChange={setCommandTemplate}
                  data-mono="true"
                  placeholder='docker exec -it -w "{target_cwd}" vpn-dev-workspace-terminal-1 {provider} {args}'
                />
                <small className={styles.hintText}>
                  {t('agents.commandTemplateHint', {
                    defaultValue:
                      'Placeholders: {cwd}, {target_cwd}, {args}, {provider}, {profile}, {profile_home}. Nexus executes argv directly; shell operators remain blocked.',
                  })}
                </small>

                <label className={styles.fieldSublabel}>
                  {t('agents.targetProfileHome', {
                    defaultValue: 'Shared target profile home (optional)',
                  })}
                </label>
                <Input
                  value={targetProfileHome}
                  onChange={setTargetProfileHome}
                  data-mono="true"
                  placeholder="/home/developer"
                />
                <small className={styles.hintText}>
                  {t('agents.targetProfileHomeHint', {
                    defaultValue:
                      'Set this only when Nexus and the custom target see the same provider-home files, such as a bind-mounted VPN workspace. This allows native Agent Projection to use that shared home.',
                  })}
                </small>

                <label className={styles.fieldSublabel}>
                  {t('agents.targetProbeTemplate', {
                    defaultValue: 'Target version probe (optional)',
                  })}
                </label>
                <Input
                  value={targetProbeTemplate}
                  onChange={setTargetProbeTemplate}
                  data-mono="true"
                  placeholder="docker exec vpn-dev-workspace-terminal-1 {provider} {args}"
                />
                <small className={styles.hintText}>
                  {t('agents.targetProbeTemplateHint', {
                    defaultValue:
                      'Nexus invokes this safely with {args} = --version to detect the provider inside Docker/WSL/SSH before native Agent Projection. Avoid interactive flags such as docker -t.',
                  })}
                </small>
              </div>
            )}
          </div>
        </details>

        <div className={styles.actionsRow}>
          <Button onClick={onClose} disabled={busy}>
            {t('common.cancel')}
          </Button>
          <Button
            tone="brand"
            onClick={handleCreate}
            disabled={
              busy ||
              loadingModes ||
              !selectedType ||
              !name.trim() ||
              (Boolean(agentProtocol) && !validStructuredProfile)
            }
          >
            <Play size={13} /> {busy ? t('agents.creating') : t('agents.createAndOpenTerminal')}
          </Button>
        </div>
      </div>
    </Dialog>
  );
};
