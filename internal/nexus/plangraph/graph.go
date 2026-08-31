package plangraph

import (
	"fmt"
	"sort"
	"strings"
)

// Node is the dependency-relevant projection of a WorkPackage.
type Node struct {
	ID           string
	Title        string
	Dependencies []string
}

// NormalizeAndValidate resolves dependency references to package IDs and
// rejects malformed DAGs before they can be persisted or executed.
func NormalizeAndValidate(nodes []Node) ([]Node, error) {
	out := make([]Node, len(nodes))
	copy(out, nodes)

	byID := make(map[string]int, len(out))
	titleIDs := make(map[string][]string, len(out))
	for i := range out {
		out[i].ID = strings.TrimSpace(out[i].ID)
		out[i].Title = strings.TrimSpace(out[i].Title)
		if out[i].ID == "" {
			return nil, fmt.Errorf("package id is required")
		}
		if _, exists := byID[out[i].ID]; exists {
			return nil, fmt.Errorf("duplicate package id %s", out[i].ID)
		}
		byID[out[i].ID] = i
		if out[i].Title != "" {
			titleIDs[out[i].Title] = append(titleIDs[out[i].Title], out[i].ID)
		}
	}

	for i := range out {
		seen := map[string]struct{}{}
		normalized := make([]string, 0, len(out[i].Dependencies))
		for _, raw := range out[i].Dependencies {
			dep := strings.TrimSpace(raw)
			if dep == "" {
				continue
			}
			resolved := dep
			if _, ok := byID[dep]; !ok {
				matches := titleIDs[dep]
				switch len(matches) {
				case 0:
					return nil, fmt.Errorf("package %s has unknown dependency %s", out[i].ID, dep)
				case 1:
					resolved = matches[0]
				default:
					return nil, fmt.Errorf("package %s has ambiguous dependency title %q", out[i].ID, dep)
				}
			}
			if resolved == out[i].ID {
				return nil, fmt.Errorf("package %s depends on itself", out[i].ID)
			}
			if _, duplicate := seen[resolved]; duplicate {
				continue
			}
			seen[resolved] = struct{}{}
			normalized = append(normalized, resolved)
		}
		out[i].Dependencies = normalized
	}

	deps := make(map[string][]string, len(out))
	for _, node := range out {
		deps[node.ID] = node.Dependencies
	}
	visiting := make(map[string]bool, len(out))
	visited := make(map[string]bool, len(out))
	var visit func(string) error
	visit = func(id string) error {
		if visiting[id] {
			return fmt.Errorf("dependency cycle includes %s", id)
		}
		if visited[id] {
			return nil
		}
		visiting[id] = true
		for _, dep := range deps[id] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for id := range deps {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// TopologicalOrder returns a stable order. IDs break ties so identical input
// always produces identical output regardless of map iteration order.
func TopologicalOrder(nodes []Node) ([]string, error) {
	normalized, err := NormalizeAndValidate(nodes)
	if err != nil {
		return nil, err
	}
	inDegree := make(map[string]int, len(normalized))
	dependents := make(map[string][]string, len(normalized))
	for _, node := range normalized {
		inDegree[node.ID] = len(node.Dependencies)
	}
	for _, node := range normalized {
		for _, dep := range node.Dependencies {
			dependents[dep] = append(dependents[dep], node.ID)
		}
	}
	ready := make([]string, 0)
	for id, degree := range inDegree {
		if degree == 0 {
			ready = append(ready, id)
		}
	}
	sort.Strings(ready)
	order := make([]string, 0, len(normalized))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		children := append([]string(nil), dependents[id]...)
		sort.Strings(children)
		for _, child := range children {
			inDegree[child]--
			if inDegree[child] == 0 {
				ready = append(ready, child)
			}
		}
		sort.Strings(ready)
	}
	if len(order) != len(normalized) {
		return nil, fmt.Errorf("dependency cycle detected")
	}
	return order, nil
}

// ExecutionWaves groups nodes whose dependencies are satisfied by prior
// waves. IDs inside each wave are sorted for deterministic dispatch.
func ExecutionWaves(nodes []Node) ([][]string, error) {
	normalized, err := NormalizeAndValidate(nodes)
	if err != nil {
		return nil, err
	}
	if _, err := TopologicalOrder(normalized); err != nil {
		return nil, err
	}
	remaining := make(map[string]Node, len(normalized))
	for _, node := range normalized {
		remaining[node.ID] = node
	}
	done := map[string]bool{}
	waves := make([][]string, 0)
	for len(remaining) > 0 {
		wave := make([]string, 0)
		for id, node := range remaining {
			ready := true
			for _, dep := range node.Dependencies {
				if !done[dep] {
					ready = false
					break
				}
			}
			if ready {
				wave = append(wave, id)
			}
		}
		if len(wave) == 0 {
			return nil, fmt.Errorf("dependency cycle detected")
		}
		sort.Strings(wave)
		waves = append(waves, wave)
		for _, id := range wave {
			done[id] = true
			delete(remaining, id)
		}
	}
	return waves, nil
}

// DependentsOf returns direct dependents in stable order.
func DependentsOf(nodes []Node, id string) ([]string, error) {
	normalized, err := NormalizeAndValidate(nodes)
	if err != nil {
		return nil, err
	}
	found := false
	out := make([]string, 0)
	for _, node := range normalized {
		if node.ID == id {
			found = true
		}
		for _, dep := range node.Dependencies {
			if dep == id {
				out = append(out, node.ID)
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("unknown package %s", id)
	}
	sort.Strings(out)
	return out, nil
}
