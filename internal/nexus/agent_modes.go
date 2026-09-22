package nexus

import (
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

// AgentMode is a product preset over the existing persistent Agent/AgentSpec
// model. It contains no provider runtime state and does not replace
// AutonomyContract: operational permissions remain enforced by Mission/runtime
// boundaries. ReadOnlyByDefault is an execution-policy signal for surfaces that
// launch this specialist autonomously outside an explicit implementation assignment.
type AgentMode struct {
	ID                   string                 `json:"id"`
	Role                 string                 `json:"role"`
	DefaultProvider      string                 `json:"default_provider,omitempty"`
	DefaultExecutionMode string                 `json:"default_execution_mode"`
	RecommendedSkillIDs  []string               `json:"recommended_skill_ids,omitempty"`
	ReadOnlyByDefault    bool                   `json:"read_only_by_default,omitempty"`
	AgentSpec            intelligence.AgentSpec `json:"agent_spec"`
}

var canonicalAgentModes = []AgentMode{
	{
		ID: "developer", Role: "developer", DefaultProvider: "codex", DefaultExecutionMode: "Safe",
		RecommendedSkillIDs: []string{"skill-repo-health", "skill-verification-before-completion"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "developer",
			Instructions:       []string{"Implemente mudanças pequenas e médias de forma incremental, mantendo testes e compatibilidade."},
			Responsibilities:   []string{"Implementar, testar e documentar mudanças de software dentro do escopo aprovado."},
			Capabilities:       []string{"implementation", "debugging", "testing"},
			Domains:            []string{"software-engineering"},
			Strengths:          []string{"incremental-delivery", "maintainability"},
			Tags:               []string{"developer", "implementation"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
		},
	},
	{
		ID: "senior-developer", Role: "senior-developer", DefaultProvider: "codex", DefaultExecutionMode: "Safe",
		RecommendedSkillIDs: []string{"skill-repo-health", "skill-systematic-debugging"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "senior-developer",
			Instructions:       []string{"Entenda invariantes e efeitos sistêmicos antes de alterar código; prefira evolução incremental e verificável."},
			Responsibilities:   []string{"Executar mudanças complexas, orientar trade-offs e preservar compatibilidade entre áreas."},
			Capabilities:       []string{"implementation", "code_review", "architecture", "debugging"},
			Domains:            []string{"software-engineering", "cross-cutting"},
			Strengths:          []string{"systems-thinking", "refactoring", "risk-management"},
			Tags:               []string{"senior", "cross-cutting"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
		},
	},
	{
		ID: "reviewer", Role: "reviewer", DefaultProvider: "claude", DefaultExecutionMode: "Safe", ReadOnlyByDefault: true,
		RecommendedSkillIDs: []string{"skill-quality-gate", "skill-verification-before-completion"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "reviewer",
			Instructions:       []string{"Procure riscos, regressões e claims sem evidência; classifique severidade objetivamente."},
			Responsibilities:   []string{"Revisar diffs, compatibilidade, qualidade e evidência sem alterar a implementação."},
			Capabilities:       []string{"code_review", "risk_analysis", "verification"},
			Constraints:        []string{"Read-only by default; remediation must be assigned to an implementation task."},
			Domains:            []string{"quality", "verification"},
			Strengths:          []string{"critical-reading", "risk-classification"},
			Tags:               []string{"review", "quality"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
		},
	},
	{
		ID: "investigator", Role: "investigator", DefaultProvider: "codex", DefaultExecutionMode: "Safe", ReadOnlyByDefault: true,
		RecommendedSkillIDs: []string{"skill-systematic-debugging", "skill-verification-before-completion"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "investigator",
			Instructions:       []string{"Reproduza, isole variáveis e colete evidência antes de propor correção.", "Comece em modo de investigação; só recomende alteração quando a causa raiz estiver sustentada por evidência."},
			Responsibilities:   []string{"Diagnosticar falhas e produzir hipótese, evidência, causa raiz e próximos passos."},
			Capabilities:       []string{"debugging", "root_cause_analysis", "log_analysis"},
			Constraints:        []string{"Read-only by default until the root cause is evidenced and remediation is explicitly assigned."},
			Domains:            []string{"diagnostics", "quality"},
			Strengths:          []string{"hypothesis-testing", "evidence-gathering"},
			Tags:               []string{"investigation", "root-cause"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true},
		},
	},
	{
		ID: "architect", Role: "architect", DefaultProvider: "claude", DefaultExecutionMode: "Safe",
		RecommendedSkillIDs: []string{"skill-adr", "skill-repo-health"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "architect",
			Instructions:       []string{"Modele limites, contratos, dependências e trade-offs antes de recomendar mudanças estruturais."},
			Responsibilities:   []string{"Definir arquitetura evolutiva, registrar decisões e identificar riscos sistêmicos."},
			Capabilities:       []string{"architecture", "system_design", "tradeoff_analysis"},
			Domains:            []string{"architecture", "software-engineering"},
			Strengths:          []string{"systems-thinking", "decision-records"},
			Tags:               []string{"architecture", "design"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true},
		},
	},
	{
		ID: "security-reviewer", Role: "security-reviewer", DefaultProvider: "claude", DefaultExecutionMode: "Safe", ReadOnlyByDefault: true,
		RecommendedSkillIDs: []string{"skill-saas-security-scan", "skill-quality-gate"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "security-reviewer",
			Instructions:       []string{"Revise ameaças, fronteiras de confiança e regressões de segurança sem assumir que ausência de evidência é segurança."},
			Responsibilities:   []string{"Identificar vulnerabilidades, severidade, evidência e mitigação recomendada."},
			Capabilities:       []string{"security", "code_review", "threat_modeling"},
			Constraints:        []string{"Read-only by default; remediation belongs to an explicitly assigned implementation task."},
			Domains:            []string{"security", "quality"},
			Strengths:          []string{"threat-modeling", "risk-classification"},
			Tags:               []string{"security", "review"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true},
		},
	},
	{
		ID: "qa-engineer", Role: "qa-engineer", DefaultProvider: "codex", DefaultExecutionMode: "Safe",
		RecommendedSkillIDs: []string{"skill-webapp-testing", "skill-verification-before-completion"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "qa-engineer",
			Instructions:       []string{"Reproduza falhas, cubra regressões e exija evidência observável antes de concluir."},
			Responsibilities:   []string{"Projetar testes unitários, integração, E2E e smoke tests."},
			Capabilities:       []string{"testing", "e2e", "regression_analysis"},
			Domains:            []string{"quality", "verification"},
			Strengths:          []string{"reproduction", "regression-detection"},
			Tags:               []string{"qa", "testing"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
		},
	},
	{
		ID: "performance-investigator", Role: "performance-investigator", DefaultProvider: "codex", DefaultExecutionMode: "Safe", ReadOnlyByDefault: true,
		RecommendedSkillIDs: []string{"skill-systematic-debugging", "skill-verification-before-completion"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "performance-investigator",
			Instructions:       []string{"Estabeleça baseline e reproduza o gargalo antes de otimizar; compare resultados depois da mudança."},
			Responsibilities:   []string{"Medir gargalos, explicar a causa e definir testes controlados de desempenho reproduzíveis."},
			Capabilities:       []string{"performance_analysis", "profiling", "benchmarking"},
			Constraints:        []string{"Read-only by default until the bottleneck is measured and remediation is explicitly assigned."},
			Domains:            []string{"performance", "diagnostics"},
			Strengths:          []string{"measurement", "profiling"},
			Tags:               []string{"performance", "investigation"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true},
		},
	},
	{
		ID: "documentation-specialist", Role: "documentation-specialist", DefaultProvider: "claude", DefaultExecutionMode: "Safe",
		RecommendedSkillIDs: []string{"skill-adr", "skill-deep-wiki"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "documentation-specialist",
			Instructions:       []string{"Escreva para o público correto, preserve histórico e diferencie fato de hipótese."},
			Responsibilities:   []string{"Produzir arquitetura, guias operacionais, ADRs e handoffs navegáveis."},
			Capabilities:       []string{"technical_writing", "architecture_docs", "documentation"},
			Domains:            []string{"documentation", "architecture"},
			Strengths:          []string{"clarity", "knowledge-organization"},
			Tags:               []string{"docs", "architecture"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true},
		},
	},
	{
		ID: "generalist", Role: "generalist", DefaultProvider: "claude", DefaultExecutionMode: "Safe",
		RecommendedSkillIDs: []string{"skill-repo-health"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "generalist",
			Instructions:       []string{"Decomponha tarefas amplas antes de executar e coordene dependências com clareza."},
			Responsibilities:   []string{"Conectar decisões de produto, código, testes e documentação."},
			Capabilities:       []string{"analysis", "implementation", "coordination"},
			Domains:            []string{"product", "software-engineering"},
			Strengths:          []string{"systems-thinking", "coordination"},
			Tags:               []string{"generalist", "multi-area"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
		},
	},
	{
		ID: "backend-engineer", Role: "backend-engineer", DefaultProvider: "codex", DefaultExecutionMode: "Safe",
		RecommendedSkillIDs: []string{"skill-database-migrations", "skill-systematic-debugging"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "backend-engineer",
			Instructions:       []string{"Priorize contratos estáveis, validação de entrada e observabilidade no backend."},
			Responsibilities:   []string{"Projetar APIs, persistência e serviços server-side resilientes."},
			Capabilities:       []string{"go", "rest_api", "database"},
			Domains:            []string{"backend", "distributed-systems"},
			Strengths:          []string{"api-design", "reliability"},
			Tags:               []string{"backend", "server-side"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
		},
	},
	{
		ID: "frontend-engineer", Role: "frontend-engineer", DefaultProvider: "claude", DefaultExecutionMode: "Safe",
		RecommendedSkillIDs: []string{"skill-modern-ui-patterns", "skill-frontend-ux-guardrails"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "frontend-engineer",
			Instructions:       []string{"Preserve acessibilidade, responsividade, i18n e os tokens visuais existentes."},
			Responsibilities:   []string{"Implementar interfaces React coesas e verificáveis."},
			Capabilities:       []string{"react", "typescript", "accessibility", "ui"},
			Domains:            []string{"frontend", "web"},
			Strengths:          []string{"ux", "component-design"},
			Tags:               []string{"frontend", "react"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
		},
	},
	{
		ID: "devops-release", Role: "devops-release", DefaultProvider: "agy", DefaultExecutionMode: "Safe",
		RecommendedSkillIDs: []string{"skill-release-engineering", "skill-incident-response"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "devops-release",
			Instructions:       []string{"Priorize reprodutibilidade, rollback seguro e observabilidade da operação."},
			Responsibilities:   []string{"Manter CI/CD, empacotamento, deploy e resposta de operação."},
			Capabilities:       []string{"ci_cd", "docker", "release", "operations"},
			Domains:            []string{"devops", "release-engineering"},
			Strengths:          []string{"automation", "rollback-planning"},
			Tags:               []string{"devops", "release"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true, RequireTests: true},
		},
	},
	{
		ID: "data-analyst", Role: "data-analyst", DefaultProvider: "codex", DefaultExecutionMode: "Safe",
		RecommendedSkillIDs: []string{"skill-unified-analytics"},
		AgentSpec: intelligence.AgentSpec{
			Role:               "data-analyst",
			Instructions:       []string{"Declare a origem dos dados, incertezas e critérios usados nas conclusões."},
			Responsibilities:   []string{"Modelar consultas, métricas e análises reproduzíveis."},
			Capabilities:       []string{"sql", "analytics", "data_modeling"},
			Domains:            []string{"data", "analytics"},
			Strengths:          []string{"measurement", "evidence-based-analysis"},
			Tags:               []string{"data", "analytics"},
			VerificationPolicy: intelligence.VerificationPolicy{RequireEvidence: true},
		},
	},
}

// CanonicalAgentModes returns a defensive copy so transport/UI callers cannot
// mutate the process-wide mode contract.
func CanonicalAgentModes() []AgentMode {
	out := make([]AgentMode, len(canonicalAgentModes))
	for i := range canonicalAgentModes {
		out[i] = cloneAgentMode(canonicalAgentModes[i])
	}
	return out
}

func AgentModeForRole(role string) (AgentMode, bool) {
	role = normalizeAgentTaxonomy(role)
	for _, mode := range canonicalAgentModes {
		if normalizeAgentTaxonomy(mode.Role) == role {
			return cloneAgentMode(mode), true
		}
	}
	return AgentMode{}, false
}

func AgentModeForID(id string) (AgentMode, bool) {
	id = normalizeAgentTaxonomy(id)
	for _, mode := range canonicalAgentModes {
		if normalizeAgentTaxonomy(mode.ID) == id {
			return cloneAgentMode(mode), true
		}
	}
	return AgentMode{}, false
}

func cloneAgentMode(mode AgentMode) AgentMode {
	mode.RecommendedSkillIDs = append([]string(nil), mode.RecommendedSkillIDs...)
	mode.AgentSpec.Instructions = append([]string(nil), mode.AgentSpec.Instructions...)
	mode.AgentSpec.Responsibilities = append([]string(nil), mode.AgentSpec.Responsibilities...)
	mode.AgentSpec.Capabilities = append([]string(nil), mode.AgentSpec.Capabilities...)
	mode.AgentSpec.Constraints = append([]string(nil), mode.AgentSpec.Constraints...)
	mode.AgentSpec.Domains = append([]string(nil), mode.AgentSpec.Domains...)
	mode.AgentSpec.Strengths = append([]string(nil), mode.AgentSpec.Strengths...)
	mode.AgentSpec.Tags = append([]string(nil), mode.AgentSpec.Tags...)
	return mode
}

func validateAgentModeConfiguration(cfg AgentConfig) error {
	mode, ok := AgentModeForRole(cfg.AgentSpec.Role)
	if !ok || !mode.ReadOnlyByDefault {
		return nil
	}
	if strings.EqualFold(agentModeStringOption(cfg.Options, "mode"), "YOLO") {
		return fmt.Errorf("agent mode %s is read-only by default and cannot use YOLO mode", mode.ID)
	}
	if strings.EqualFold(agentModeStringOption(cfg.Options, "execution_adapter"), "command_template") {
		return fmt.Errorf("agent mode %s is read-only by default; custom command templates have no verified read-only contract", mode.ID)
	}
	for _, arg := range agentModeStringSliceOption(cfg.Options, "extra_args") {
		normalized := strings.ToLower(strings.TrimSpace(arg))
		if normalized == "--yolo" || normalized == "--dangerously-skip-permissions" || normalized == "--full-auto" {
			return fmt.Errorf("agent mode %s rejected unsafe execution argument %q", mode.ID, arg)
		}
	}
	return nil
}

func agentModeStringOption(options map[string]any, key string) string {
	if options == nil {
		return ""
	}
	value, _ := options[key].(string)
	return strings.TrimSpace(value)
}

func agentModeStringSliceOption(options map[string]any, key string) []string {
	if options == nil {
		return nil
	}
	switch value := options[key].(type) {
	case []string:
		return append([]string(nil), value...)
	case []any:
		out := make([]string, 0, len(value))
		for _, item := range value {
			if text, ok := item.(string); ok {
				out = append(out, text)
			}
		}
		return out
	default:
		return nil
	}
}
