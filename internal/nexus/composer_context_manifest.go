package nexus

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/IAPro-Community/nexus/internal/core/security"
	nexusskills "github.com/IAPro-Community/nexus/internal/nexus/skills"
)

const (
	composerContextManifestVersion    = "1"
	composerContextMaxEstimatedTokens = 8000
	composerContextMaxSkillCandidates = 8
	composerContextMaxMissionRefs     = 5
)

type ComposerContextDecision string

const (
	ComposerContextKeep          ComposerContextDecision = "KEEP"
	ComposerContextReferenceOnly ComposerContextDecision = "REFERENCE_ONLY"
	ComposerContextDrop          ComposerContextDecision = "DROP"
)

type ComposerContextItem struct {
	ID              string                  `json:"id"`
	Kind            string                  `json:"kind"`
	Source          string                  `json:"source"`
	Locator         string                  `json:"locator,omitempty"`
	Summary         string                  `json:"summary"`
	Confidence      string                  `json:"confidence,omitempty"`
	Decision        ComposerContextDecision `json:"decision"`
	Reason          string                  `json:"reason"`
	EstimatedTokens int                     `json:"estimated_tokens"`
}

type ComposerContextBudget struct {
	MaxEstimatedTokens  int `json:"max_estimated_tokens"`
	UsedEstimatedTokens int `json:"used_estimated_tokens"`
	KeptItems           int `json:"kept_items"`
	ReferencedItems     int `json:"referenced_items"`
	DroppedItems        int `json:"dropped_items"`
}

type ComposerContextManifest struct {
	Version     string                `json:"version"`
	ProjectID   string                `json:"project_id"`
	Fingerprint string                `json:"fingerprint"`
	GeneratedAt time.Time             `json:"generated_at"`
	Budget      ComposerContextBudget `json:"budget"`
	Items       []ComposerContextItem `json:"items"`
	Warnings    []string              `json:"warnings,omitempty"`
}

type composerContextCandidate struct {
	Item          ComposerContextItem
	Priority      int
	ReferenceOnly bool
}

type composerRankedSkill struct {
	Skill   nexusskills.Skill
	Score   int
	Reasons []string
}

func (n *Nexus) BuildComposerContextManifest(ctx context.Context, projectID string, brief LivingBrief) (ComposerContextManifest, error) {
	if err := ctx.Err(); err != nil {
		return ComposerContextManifest{}, err
	}
	st, err := n.OpenProject()
	if err != nil {
		return ComposerContextManifest{}, err
	}
	project, err := st.GetProject(projectID)
	if err != nil {
		return ComposerContextManifest{}, err
	}
	manifest := ComposerContextManifest{
		Version:     composerContextManifestVersion,
		ProjectID:   projectID,
		GeneratedAt: time.Now().UTC(),
		Budget:      ComposerContextBudget{MaxEstimatedTokens: composerContextMaxEstimatedTokens},
		Items:       []ComposerContextItem{},
	}
	candidates := []composerContextCandidate{}

	add := func(kind, source, locator, summary, confidence, reason string, priority int, referenceOnly bool) {
		summary = security.Redact(boundedComposerContextSummary(summary, 4096))
		if strings.TrimSpace(summary) == "" {
			return
		}
		item := ComposerContextItem{
			Kind: kind, Source: source, Locator: strings.TrimSpace(locator),
			Summary: summary, Confidence: strings.TrimSpace(confidence),
			Reason: reason, EstimatedTokens: estimateComposerContextTokens(summary),
		}
		item.ID = stableComposerContextItemID(item)
		candidates = append(candidates, composerContextCandidate{Item: item, Priority: priority, ReferenceOnly: referenceOnly})
	}

	if readiness, readinessErr := n.ObserveContextReadiness(projectID); readinessErr == nil {
		identity := fmt.Sprintf("Project %s · branch %s · HEAD %s", project.Name,
			firstNonEmpty(readiness.CurrentFingerprint.Branch, "unknown"),
			firstNonEmpty(readiness.CurrentFingerprint.Head, "unknown"))
		add("PROJECT_IDENTITY", "nexus-context", readiness.CurrentFingerprintID, identity, "OBSERVED",
			"Current project identity anchors every later context decision.", 120, false)
	} else {
		manifest.Warnings = append(manifest.Warnings, "project identity unavailable: "+readinessErr.Error())
	}

	for _, file := range brief.Context.ReferencedFiles {
		summary := file.Path
		if strings.TrimSpace(file.Excerpt) != "" {
			summary += "\n" + file.Excerpt
		}
		add("REFERENCED_FILE", "user", file.Path, summary, "EXPLICIT",
			"Explicit user context outranks inferred repository context.", 130, false)
	}

	if memories, memoryErr := n.RecallIntelligenceMemories(ctx, MemoryRecallRequest{ProjectID: projectID, Limit: 12}); memoryErr == nil {
		for _, recalled := range memories.Items {
			m := recalled.Memory
			priority := 92
			if m.Scope == MemoryScopeProject {
				priority = 100
			}
			add("MEMORY", strings.ToLower(m.Scope), m.Key, m.Key+"="+m.Value, "DURABLE",
				"Reusable preference retained by Nexus memory.", priority, false)
		}
	} else {
		manifest.Warnings = append(manifest.Warnings, "memory recall unavailable: "+memoryErr.Error())
	}

	if grounded, groundedErr := n.BoundedProjectIntelligenceContext(ctx, projectID); groundedErr == nil {
		for _, fact := range composerManifestFacts(grounded) {
			confidence := strings.ToUpper(strings.TrimSpace(fmt.Sprint(fact["confidence"])))
			priority := 68
			switch confidence {
			case "HIGH":
				priority = 82
			case "MEDIUM":
				priority = 72
			case "LOW":
				priority = 58
			}
			key := strings.TrimSpace(fmt.Sprint(fact["key"]))
			value := strings.TrimSpace(fmt.Sprint(fact["value"]))
			source := strings.TrimSpace(fmt.Sprint(fact["source"]))
			locator := strings.TrimSpace(fmt.Sprint(fact["locator"]))
			if source == "" {
				source = "project-intelligence"
			}
			add("PROJECT_FACT", source, locator, key+"="+value, confidence,
				"Static project intelligence with provenance.", priority, false)
		}
	} else {
		manifest.Warnings = append(manifest.Warnings, "project intelligence unavailable: "+groundedErr.Error())
	}

	if catalog, catalogErr := n.skillCatalogForProject(projectID); catalogErr == nil {
		for _, ranked := range rankComposerSkillCandidates(catalog, brief) {
			skill := ranked.Skill
			summary := firstNonEmpty(skill.Name, skill.ID)
			if strings.TrimSpace(skill.Description) != "" {
				summary += " · " + skill.Description
			}
			add("SKILL_CANDIDATE", string(skill.Source), skill.ID, summary, "CATALOG",
				fmt.Sprintf("Candidate score %d: %s", ranked.Score, strings.Join(ranked.Reasons, "; ")),
				40+ranked.Score, true)
		}
	} else {
		manifest.Warnings = append(manifest.Warnings, "skill catalog unavailable: "+catalogErr.Error())
	}

	if runs, runsErr := st.ListMissionRunsForProject(projectID, composerContextMaxMissionRefs); runsErr == nil {
		for _, run := range runs {
			add("MISSION_REFERENCE", "mission", run.ID,
				fmt.Sprintf("Mission %s · plan %s · state %s · updated %s", run.ID, run.PlanID, run.State, run.UpdatedAt.UTC().Format(time.RFC3339)),
				"OBSERVED", "Recent Mission is available by reference; raw transcripts are intentionally excluded.", 35, true)
		}
	} else {
		manifest.Warnings = append(manifest.Warnings, "mission references unavailable: "+runsErr.Error())
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].ReferenceOnly != candidates[j].ReferenceOnly {
			return !candidates[i].ReferenceOnly
		}
		if candidates[i].Priority != candidates[j].Priority {
			return candidates[i].Priority > candidates[j].Priority
		}
		return candidates[i].Item.ID < candidates[j].Item.ID
	})

	for _, candidate := range candidates {
		item := candidate.Item
		switch {
		case candidate.ReferenceOnly:
			item.Decision = ComposerContextReferenceOnly
			manifest.Budget.ReferencedItems++
		case manifest.Budget.UsedEstimatedTokens+item.EstimatedTokens <= manifest.Budget.MaxEstimatedTokens:
			item.Decision = ComposerContextKeep
			manifest.Budget.UsedEstimatedTokens += item.EstimatedTokens
			manifest.Budget.KeptItems++
		default:
			item.Decision = ComposerContextDrop
			item.Reason = "Dropped from model context because the Composer context budget was exhausted; provenance remains visible."
			manifest.Budget.DroppedItems++
		}
		manifest.Items = append(manifest.Items, item)
	}

	manifest.Fingerprint = composerContextManifestFingerprint(projectID, brief, manifest.Items)
	return manifest, nil
}

func composerExecutionBrief(brief LivingBrief, manifest ComposerContextManifest) LivingBrief {
	if strings.TrimSpace(manifest.Fingerprint) == "" {
		// Legacy PromptArtifacts predate context-decision evidence. Preserve
		// their explicit file context instead of pretending Nexus knows which
		// references would have been KEEP or DROP.
		return brief
	}
	keepPaths := make(map[string]struct{})
	for _, item := range manifest.Items {
		if item.Kind == "REFERENCED_FILE" && item.Decision == ComposerContextKeep && strings.TrimSpace(item.Locator) != "" {
			keepPaths[item.Locator] = struct{}{}
		}
	}
	if len(brief.Context.ReferencedFiles) > 0 {
		filtered := make([]ContextFileRef, 0, len(brief.Context.ReferencedFiles))
		for _, file := range brief.Context.ReferencedFiles {
			if _, keep := keepPaths[file.Path]; keep {
				filtered = append(filtered, file)
			}
		}
		brief.Context.ReferencedFiles = filtered
	}
	// Preview is a derived field. The immutable audit brief may contain the
	// pre-pruning preview, so recompute it from the admitted execution brief
	// instead of serializing stale DROP/REFERENCE_ONLY context downstream.
	brief.Preview = renderCanonicalPrompt(brief)
	return brief
}

// renderComposerExecutionPrompt renders the admitted execution context and
// enforces the context-manifest KEEP invariant. Referenced files selected as
// KEEP must be present in the executable PromptArtifact even if another
// rendering layer omits them; DROP and REFERENCE_ONLY items remain excluded.
func renderComposerExecutionPrompt(brief LivingBrief, manifest ComposerContextManifest) string {
	prompt := renderCanonicalPrompt(composerExecutionBrief(brief, manifest))
	if strings.TrimSpace(manifest.Fingerprint) == "" {
		return prompt
	}

	missing := make([]ComposerContextItem, 0)
	for _, item := range manifest.Items {
		if item.Kind != "REFERENCED_FILE" || item.Decision != ComposerContextKeep {
			continue
		}
		locator := strings.TrimSpace(item.Locator)
		if locator == "" || strings.Contains(prompt, locator) {
			continue
		}
		missing = append(missing, item)
	}
	if len(missing) == 0 {
		return prompt
	}

	var b strings.Builder
	b.WriteString(strings.TrimSpace(prompt))
	if b.Len() > 0 {
		b.WriteString("\n\n")
	}
	b.WriteString("# Context — Manifest KEEP\n")
	for _, item := range missing {
		locator := strings.TrimSpace(item.Locator)
		b.WriteString("- ")
		b.WriteString(locator)
		b.WriteString("\n")
		summary := strings.TrimSpace(item.Summary)
		if summary != "" && summary != locator {
			b.WriteString("  ")
			b.WriteString(summary)
			if !strings.HasSuffix(summary, "\n") {
				b.WriteString("\n")
			}
		}
	}
	return strings.TrimSpace(b.String())
}

func composerExecutionContextJSON(projectID string, brief LivingBrief, manifest ComposerContextManifest) string {
	payload := map[string]any{
		"project_id": projectID,
		"brief":      composerExecutionBrief(brief, manifest),
	}
	if strings.TrimSpace(manifest.Fingerprint) != "" {
		payload["context_manifest"] = manifest.IntelligenceContext()
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func (m ComposerContextManifest) IntelligenceContext() map[string]any {
	items := make([]map[string]any, 0, m.Budget.KeptItems)
	for _, item := range m.Items {
		if item.Decision != ComposerContextKeep {
			continue
		}
		items = append(items, map[string]any{
			"id": item.ID, "kind": item.Kind, "source": item.Source,
			"locator": item.Locator, "summary": item.Summary, "confidence": item.Confidence,
		})
	}
	return map[string]any{
		"version":     m.Version,
		"fingerprint": m.Fingerprint,
		"budget": map[string]any{
			"max_estimated_tokens":  m.Budget.MaxEstimatedTokens,
			"used_estimated_tokens": m.Budget.UsedEstimatedTokens,
		},
		"items": items,
	}
}

func rankComposerSkillCandidates(catalog nexusskills.Catalog, brief LivingBrief) []composerRankedSkill {
	tokens := composerContextSearchTokens(brief)
	if len(tokens) == 0 {
		return nil
	}
	out := []composerRankedSkill{}
	for _, skill := range catalog.Library {
		if skill.Availability != nexusskills.AvailabilityAvailable {
			continue
		}
		hay := strings.ToLower(strings.Join([]string{
			skill.ID, skill.Name, skill.Description,
			strings.Join(skill.Capabilities, " "), strings.Join(skill.Triggers, " "),
		}, " "))
		score := 0
		reasons := []string{}
		for _, token := range tokens {
			if !strings.Contains(hay, token) {
				continue
			}
			score += 8
			if len(reasons) < 3 {
				reasons = append(reasons, "matches "+token)
			}
			if containsNormalized(skill.Capabilities, token) || containsNormalized(skill.Triggers, token) {
				score += 12
			}
		}
		if score == 0 {
			continue
		}
		switch skill.Source {
		case nexusskills.SourceProject:
			score += 6
			reasons = append(reasons, "project-local")
		case nexusskills.SourceUser:
			score += 4
			reasons = append(reasons, "user-local")
		case nexusskills.SourceMaestro:
			score += 2
			reasons = append(reasons, "Maestro catalog")
		}
		out = append(out, composerRankedSkill{Skill: skill, Score: score, Reasons: reasons})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Skill.ID < out[j].Skill.ID
	})
	if len(out) > composerContextMaxSkillCandidates {
		out = out[:composerContextMaxSkillCandidates]
	}
	return out
}

func composerContextSearchTokens(brief LivingBrief) []string {
	text := strings.ToLower(strings.Join([]string{
		brief.Goal, brief.Intent.Objective, brief.Intent.DesiredOutcome,
		strings.Join(brief.Scope.InScope, " "),
		strings.Join(brief.Constraints.Security, " "),
		strings.Join(brief.Constraints.Technical, " "),
		strings.Join(brief.Quality.AcceptanceCriteria, " "),
		strings.Join(brief.Quality.Testing, " "),
		strings.Join(brief.Risks, " "),
		strings.Join(brief.Environment.Stack, " "),
	}, " "))
	stop := map[string]bool{
		"the": true, "and": true, "for": true, "with": true, "from": true,
		"para": true, "com": true, "uma": true, "que": true, "por": true,
		"this": true, "that": true, "isso": true, "essa": true, "este": true,
	}
	seen := map[string]bool{}
	out := []string{}
	for _, token := range strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' && r != '-'
	}) {
		token = strings.TrimSpace(token)
		if len(token) < 3 || stop[token] || seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
		if len(out) >= 40 {
			break
		}
	}
	return out
}

func composerManifestFacts(grounded map[string]any) []map[string]any {
	if facts, ok := grounded["facts"].([]map[string]any); ok {
		return facts
	}
	out := []map[string]any{}
	if facts, ok := grounded["facts"].([]any); ok {
		for _, raw := range facts {
			if fact, ok := raw.(map[string]any); ok {
				out = append(out, fact)
			}
		}
	}
	return out
}

func estimateComposerContextTokens(value string) int {
	if value == "" {
		return 0
	}
	return max(1, (len([]rune(value))+3)/4)
}

func boundedComposerContextSummary(value string, maxRunes int) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + "…"
}

func stableComposerContextItemID(item ComposerContextItem) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{item.Kind, item.Source, item.Locator, item.Summary}, "\x00")))
	return "ctx_" + hex.EncodeToString(sum[:8])
}

func composerContextManifestFingerprint(projectID string, brief LivingBrief, items []ComposerContextItem) string {
	payload := struct {
		ProjectID string                `json:"project_id"`
		Goal      string                `json:"goal"`
		Items     []ComposerContextItem `json:"items"`
	}{ProjectID: projectID, Goal: brief.Goal, Items: items}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func mustComposerContextManifestJSON(manifest ComposerContextManifest) string {
	raw, err := json.Marshal(manifest)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func decodeComposerContextManifest(raw string) ComposerContextManifest {
	var manifest ComposerContextManifest
	if strings.TrimSpace(raw) == "" {
		return manifest
	}
	if json.Unmarshal([]byte(raw), &manifest) != nil {
		return ComposerContextManifest{}
	}
	return manifest
}
