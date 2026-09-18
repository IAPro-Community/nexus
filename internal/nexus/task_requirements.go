package nexus

import (
	"encoding/json"
	"strings"
)

// ClassifyTaskRequirements turns the user's natural-language goal into the
// single TaskRequirements shape consumed by AgentMatcher and ResourceScheduler.
// It is intentionally conservative: unknown wording keeps the generic coding
// role instead of inventing capabilities that no evidence supports.
func ClassifyTaskRequirements(prompt string) TaskRequirements {
	text := strings.ToLower(strings.TrimSpace(prompt))
	req := TaskRequirements{
		TaskKind:             "coding",
		Role:                 "implementer",
		AcceptableRoles:      []string{"fullstack-engineer", "generalist"},
		RequiredCapabilities: []string{"headless", "submit_prompt"},
		Confidence:           "MEDIUM",
		Source:               "natural-language",
	}

	switch {
	case containsAny(text, "ecommerce", "e-commerce", "plataforma completa", "sistema completo"):
		req.Role = "generalist"
		req.PreferredRoles = []string{"generalist", "fullstack-engineer"}
		req.AcceptableRoles = []string{"generalist", "fullstack-engineer", "architect"}
		req.Domains = []string{"product", "frontend", "backend", "payments"}
		req.EstimatedComplexity = "high"
		req.RequiresDecomposition = true
	case containsAny(text, "auditoria de segurança", "security audit", "segurança", "security", "csrf", "oauth"):
		req.TaskKind = "security"
		req.Role = "security"
		req.PreferredRoles = []string{"security", "reviewer"}
		req.AcceptableRoles = []string{"security", "reviewer", "qa"}
		req.Domains = []string{"security", "backend"}
		req.RequiredCapabilities = append(req.RequiredCapabilities, "security")
	case containsAny(text, "revise este pr", "review", "regressões", "regressoes", "pull request"):
		req.TaskKind = "review"
		req.Role = "reviewer"
		req.PreferredRoles = []string{"reviewer", "qa"}
		req.AcceptableRoles = []string{"reviewer", "qa", "security"}
		req.Domains = []string{"quality", "verification"}
		req.RequiredCapabilities = append(req.RequiredCapabilities, "code_review")
	case containsAny(text, "kubernetes", "k8s", "deploy", "infraestrutura", "infra"):
		req.TaskKind = "coding"
		req.Role = "devops"
		req.PreferredRoles = []string{"devops"}
		req.AcceptableRoles = []string{"devops", "generalist"}
		req.Domains = []string{"devops", "kubernetes"}
		req.RequiredCapabilities = append(req.RequiredCapabilities, "kubernetes")
	case containsAny(text, "api rest em go", "go rest", "api em go", "golang"):
		req.Role = "backend-engineer"
		req.PreferredRoles = []string{"backend-engineer"}
		req.AcceptableRoles = []string{"backend-engineer", "fullstack-engineer"}
		req.Domains = []string{"backend", "go", "rest-api"}
		req.RequiredCapabilities = append(req.RequiredCapabilities, "go", "rest_api")
	case containsAny(text, "react", "ui", "frontend", "front-end", "formulário", "formulario"):
		req.Role = "frontend-engineer"
		req.PreferredRoles = []string{"frontend-engineer"}
		req.AcceptableRoles = []string{"frontend-engineer", "fullstack-engineer"}
		req.Domains = []string{"frontend", "web"}
		if containsAny(text, "react") {
			req.RequiredCapabilities = append(req.RequiredCapabilities, "react")
		}
	}

	return req
}

func containsAny(value string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(value, term) {
			return true
		}
	}
	return false
}

func taskRequirementsJSON(prompt, role string) string {
	req := ClassifyTaskRequirements(prompt)
	// "implementer" is a legacy execution label, not a specialization. Keep
	// the classifier's richer frontend/backend/devops role in that case.
	if strings.TrimSpace(role) != "" && !strings.EqualFold(strings.TrimSpace(role), "implementer") {
		req.Role = role
		req.PreferredRoles = []string{role}
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		return ""
	}
	return string(encoded)
}
