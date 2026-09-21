package nexus

import "testing"

func TestClassifyTaskRequirementsCoversCanonicalInputs(t *testing.T) {
	tests := []struct {
		prompt, role, domain, capability string
		decompose                        bool
	}{
		{"corrija a UI React", "frontend-engineer", "frontend", "react", false},
		{"crie uma API REST em Go", "backend-engineer", "backend", "go", false},
		{"configure Kubernetes", "devops", "kubernetes", "kubernetes", false},
		{"faça auditoria de segurança", "security-reviewer", "security", "security", false},
		{"revise este PR procurando regressões", "reviewer", "quality", "code_review", false},
		{"investigue a causa raiz de uma falha intermitente", "investigator", "diagnostics", "root_cause_analysis", false},
		{"faça profiling e benchmark da latência", "performance-investigator", "performance", "performance_analysis", false},
		{"desenhe a arquitetura e registre um ADR", "architect", "architecture", "architecture", false},
		{"crie um ecommerce completo", "generalist", "product", "", true},
	}

	for _, test := range tests {
		t.Run(test.prompt, func(t *testing.T) {
			req := ClassifyTaskRequirements(test.prompt)
			if req.Role != test.role || !containsNormalized(req.Domains, test.domain) {
				t.Fatalf("requirements=%+v, want role=%q domain=%q", req, test.role, test.domain)
			}
			if test.capability != "" && !containsNormalized(req.RequiredCapabilities, test.capability) {
				t.Fatalf("required capabilities=%v, want %q", req.RequiredCapabilities, test.capability)
			}
			if req.RequiresDecomposition != test.decompose {
				t.Fatalf("RequiresDecomposition=%v, want %v", req.RequiresDecomposition, test.decompose)
			}
		})
	}
}
