package nexus

import (
	"strings"
	"unicode/utf8"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

// compactComposerIntelligenceBrief strips heavy LivingBrief fields (preview,
// facts, motivation, file excerpts, long source prompts) before they are
// serialized into a headless CLI prompt. Dumping the full brief into agy/codex
// is what triggers signal: killed (OOM) and freezes the Composer turn.
func compactComposerIntelligenceBrief(brief LivingBrief) map[string]any {
	files := make([]string, 0, len(brief.Context.ReferencedFiles))
	for _, file := range brief.Context.ReferencedFiles {
		path := strings.TrimSpace(file.Path)
		if path == "" {
			continue
		}
		files = append(files, path)
		if len(files) >= 12 {
			break
		}
	}
	openUnknowns := make([]map[string]string, 0, 8)
	for _, unknown := range brief.Unknowns {
		if unknown.Status != PromptUnknownOpen {
			continue
		}
		openUnknowns = append(openUnknowns, map[string]string{
			"field":    unknown.Field,
			"question": unknown.Question,
			"severity": string(unknown.Severity),
		})
		if len(openUnknowns) >= 8 {
			break
		}
	}
	source := strings.TrimSpace(brief.SourcePrompt)
	if utf8.RuneCountInString(source) > 400 {
		source = string([]rune(source)[:400]) + "…"
	}
	return map[string]any{
		"goal":             strings.TrimSpace(brief.Goal),
		"source_prompt":    source,
		"intent":           brief.Intent,
		"scope":            brief.Scope,
		"quality":          brief.Quality,
		"constraints":      brief.Constraints,
		"decisions":        brief.Decisions,
		"risks":            brief.Risks,
		"environment":      map[string]any{"stack": brief.Environment.Stack},
		"context":          map[string]any{"existing_state": brief.Context.ExistingState, "domain": brief.Context.Domain},
		"open_unknowns":    openUnknowns,
		"readiness_state":  brief.Readiness.State,
		"referenced_files": files,
	}
}

// isLowSignalComposerTurn reports short acknowledgements that should not launch
// a headless coding-agent CLI (continue?, ok, sim…).
func isLowSignalComposerTurn(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	normalized = strings.Trim(normalized, "?.!…")
	normalized = strings.Join(strings.Fields(normalized), " ")
	if normalized == "" {
		return true
	}
	if utf8.RuneCountInString(normalized) > 48 {
		return false
	}
	switch normalized {
	case "continue", "continuar", "continua", "ok", "okay", "sim", "yes", "y", "pode",
		"go", "next", "proximo", "próximo", "segue", "seguir", "valeu", "thanks", "obrigado",
		"beleza", "show", "certo", "isso", "uai", "vai":
		return true
	}
	if strings.HasPrefix(normalized, "continue ") || strings.HasPrefix(normalized, "continuar ") {
		return true
	}
	return false
}

func shouldEnrichComposerTurn(brief LivingBrief, content string) bool {
	if PromptIsShallow(brief) {
		return true
	}
	return !isLowSignalComposerTurn(content)
}

// turnIntelligenceNotice maps provider failures to a short user-facing line.
// Transient kills/timeouts stay silent — heuristics already merged the turn.
func turnIntelligenceNotice(err error) string {
	if err == nil || intelligence.IsTransientIntelligenceFailure(err) {
		return ""
	}
	return "Inteligência local indisponível nesta rodada; continuei com o brief atual."
}
