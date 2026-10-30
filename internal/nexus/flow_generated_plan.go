package nexus

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
)

const maxGeneratedFlowActivities = 32

var flowOrdinalReference = regexp.MustCompile(`^(?:#|step |etapa |atividade )?(\d+)$`)

// validateGeneratedFlowOutlines rejects uncertain or fabricated-looking AI
// topology BEFORE anything is persisted. Dependency references must resolve;
// dropping them could silently execute dependent work out of order.
func validateGeneratedFlowOutlines(prompt string, outlines []intelligence.WorkPackageOutline) error {
	if len(outlines) == 0 || len(outlines) > maxGeneratedFlowActivities {
		return fmt.Errorf("Flow planner returned %d activities; expected 1..%d", len(outlines), maxGeneratedFlowActivities)
	}
	if len(outlines) == 1 && requiresMultipleFlowActivities(prompt) {
		return fmt.Errorf("Flow planner returned one activity for a multi-workstream prompt; refine the prompt or retry generation")
	}
	index := make(map[string]int, len(outlines)*4)
	for i, outline := range outlines {
		title := strings.TrimSpace(outline.Title)
		goal := strings.TrimSpace(outline.Goal)
		if isSchemaPlaceholder(title) || isSchemaPlaceholder(goal) || len(goal) < 12 {
			return fmt.Errorf("Flow activity %d has no specific title/goal", i+1)
		}
		if len(outline.Acceptance) == 0 || len(sanitizeAcceptance(outline.Acceptance)) == 0 {
			return fmt.Errorf("Flow activity %d has no measurable acceptance criteria", i+1)
		}
		key := strings.ToLower(title)
		if _, exists := index[key]; exists {
			return fmt.Errorf("Flow contains ambiguous duplicate activity title %q", title)
		}
		index[key] = i
		index[strconv.Itoa(i+1)] = i
		index["#"+strconv.Itoa(i+1)] = i
		index["step "+strconv.Itoa(i+1)] = i
		index["etapa "+strconv.Itoa(i+1)] = i
		index["atividade "+strconv.Itoa(i+1)] = i
	}
	edges := make([][]int, len(outlines))
	for i, outline := range outlines {
		unique := make(map[int]bool)
		for _, reference := range outline.Dependencies {
			ref := strings.ToLower(strings.TrimSpace(reference))
			if ref == "" || isSchemaPlaceholder(ref) {
				continue
			}
			target, found := index[ref]
			if !found {
				if ordinal := flowOrdinalReference.FindStringSubmatch(ref); ordinal != nil {
					position, _ := strconv.Atoi(ordinal[1])
					target, found = position-1, position>=1 && position<=len(outlines)
				}
			}
			if !found {
				return fmt.Errorf("Flow activity %d references unknown dependency %q", i+1, reference)
			}
			if target == i {
				return fmt.Errorf("Flow activity %d depends on itself", i+1)
			}
			if !unique[target] {
				edges[i] = append(edges[i], target)
				unique[target] = true
			}
		}
	}
	state := make([]uint8, len(outlines))
	var visit func(int) error
	visit = func(i int) error {
		if state[i] == 1 {
			return fmt.Errorf("Flow planner returned circular dependencies involving activity %d", i+1)
		}
		if state[i] == 2 {
			return nil
		}
		state[i] = 1
		for _, dependent := range edges[i] {
			if err := visit(dependent); err != nil {
				return err
			}
		}
		state[i] = 2
		return nil
	}
	for i := range outlines {
		if err := visit(i); err != nil {
			return err
		}
	}
	return nil
}

func requiresMultipleFlowActivities(prompt string) bool {
	text := strings.ToLower(prompt)
	groups := [][]string{
		{"backend", "back-end", "servidor", "api"},
		{"frontend", "front-end", "interface", "react", "ui"},
		{"database", "banco de dados", "migration", "migração", "schema"},
		{"tests", "testes", "testing", "qa", "e2e"},
		{"review", "revisão", "security audit", "auditoria"},
	}
	found := 0
	for _, group := range groups {
		for _, word := range group {
			if strings.Contains(text, word) {
				found++
				break
			}
		}
	}
	return found >= 2
}
