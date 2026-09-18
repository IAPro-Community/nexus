package skills

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type Catalog struct {
	Operational []Skill `json:"operational"`
	Library     []Skill `json:"library"`
	Counts      struct {
		Operational int `json:"operational"`
		Library     int `json:"library"`
		Copies      int `json:"copies"`
	} `json:"counts"`
}

func NewCatalog(sources ...Source) (Catalog, error) {
	merged := map[string]Skill{}
	copies := map[string]int{}
	for _, source := range sources {
		if source == nil || !source.Available(context.Background()) {
			continue
		}
		items, err := source.Discover(context.Background())
		if err != nil {
			if errors.Is(err, ErrSourceUnavailable) {
				continue
			}
			return Catalog{}, fmt.Errorf("discover skills from %s: %w", source.ID(), err)
		}
		for _, item := range items {
			item, err = normalizeSkill(item, source.ID())
			if err != nil {
				continue
			}
			copies[item.ID]++
			current, exists := merged[item.ID]
			if !exists || better(item, current) {
				merged[item.ID] = item
			}
		}
	}
	result := Catalog{Operational: []Skill{}, Library: []Skill{}}
	for id, item := range merged {
		item.Copies = copies[id]
		result.Library = append(result.Library, item)
		if item.Availability == AvailabilityAvailable {
			result.Operational = append(result.Operational, item)
		}
	}
	sort.Slice(result.Library, func(i, j int) bool { return result.Library[i].ID < result.Library[j].ID })
	sort.Slice(result.Operational, func(i, j int) bool { return result.Operational[i].ID < result.Operational[j].ID })
	result.Counts.Library = len(result.Library)
	result.Counts.Operational = len(result.Operational)
	for _, item := range result.Library {
		result.Counts.Copies += item.Copies
	}
	return result, nil
}

func (c Catalog) Resolve(id string) (Skill, bool) {
	id = strings.TrimSpace(id)
	for _, item := range c.Library {
		if item.ID == id {
			return item, true
		}
	}
	return Skill{}, false
}

func (c Catalog) ResolveCapability(capability string) (Resolution, bool) {
	capability = strings.ToLower(strings.TrimSpace(capability))
	if capability == "" {
		return Resolution{}, false
	}
	resolution := Resolution{RequestedCapability: capability}
	for _, item := range c.Library {
		if contains(item.Capabilities, capability) || contains(item.Triggers, capability) || strings.Contains(strings.ToLower(item.ID), capability) {
			resolution.Candidates = append(resolution.Candidates, item.ID)
		}
	}
	if len(resolution.Candidates) == 0 {
		return resolution, false
	}
	selected, _ := c.Resolve(resolution.Candidates[0])
	resolution.Selected = selected.ID
	resolution.Source = selected.Source
	resolution.Version = selected.Version
	resolution.Hash = selected.Hash
	resolution.Reason = "deterministic capability match"
	return resolution, true
}

func better(left, right Skill) bool {
	if compareVersion(left.Version, right.Version) != 0 {
		return compareVersion(left.Version, right.Version) > 0
	}
	if sourcePriority(left.Source) != sourcePriority(right.Source) {
		return sourcePriority(left.Source) > sourcePriority(right.Source)
	}
	return left.Provenance.SourceID < right.Provenance.SourceID
}

func sourcePriority(source SourceID) int {
	switch source {
	case SourceProject:
		return 5
	case SourceUser:
		return 4
	case SourceBuiltin:
		return 3
	case SourceMaestro:
		return 2
	case SourceExternal:
		return 1
	default:
		return 0
	}
}

func compareVersion(left, right string) int {
	parse := func(value string) []int {
		parts := strings.Split(strings.TrimPrefix(value, "v"), ".")
		result := make([]int, 3)
		for i := 0; i < len(parts) && i < len(result); i++ {
			result[i], _ = strconv.Atoi(parts[i])
		}
		return result
	}
	l, r := parse(left), parse(right)
	for i := range l {
		if l[i] != r[i] {
			if l[i] < r[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func contains(values []string, wanted string) bool {
	for _, value := range values {
		if strings.ToLower(strings.TrimSpace(value)) == wanted {
			return true
		}
	}
	return false
}
