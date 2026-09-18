package skills

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

type SourceID string

const (
	SourceBuiltin  SourceID = "builtin"
	SourceProject  SourceID = "project"
	SourceUser     SourceID = "user"
	SourceExternal SourceID = "external"
	SourceMaestro  SourceID = "maestro"
)

type Availability string

const (
	AvailabilityAvailable      Availability = "AVAILABLE"
	AvailabilitySynchronizable Availability = "SYNCHRONIZABLE"
	AvailabilityTaskOnly       Availability = "TASK_ONLY"
)

type Provenance struct {
	SourceID   string    `json:"source_id"`
	Path       string    `json:"path,omitempty"`
	Locator    string    `json:"locator,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

// Skill is the source-agnostic Nexus contract. Instructions are bounded
// knowledge/context, not executable plugin code.
type Skill struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Description  string       `json:"description,omitempty"`
	Source       SourceID     `json:"source"`
	Version      string       `json:"version,omitempty"`
	Hash         string       `json:"hash"`
	Capabilities []string     `json:"capabilities,omitempty"`
	Triggers     []string     `json:"triggers,omitempty"`
	Risk         string       `json:"risk,omitempty"`
	Instructions string       `json:"instructions,omitempty"`
	Dependencies []string     `json:"dependencies,omitempty"`
	Permissions  []string     `json:"permissions,omitempty"`
	Provenance   Provenance   `json:"provenance"`
	Availability Availability `json:"availability"`
	Mode         string       `json:"activation_mode,omitempty"`
	Copies       int          `json:"copies,omitempty"`
}

type Resolution struct {
	RequestedCapability string   `json:"requested_capability"`
	Candidates          []string `json:"candidate_skills,omitempty"`
	Selected            string   `json:"selected_skill,omitempty"`
	Source              SourceID `json:"source,omitempty"`
	Version             string   `json:"version,omitempty"`
	Hash                string   `json:"hash,omitempty"`
	Reason              string   `json:"reason,omitempty"`
}

type Source interface {
	ID() string
	Available(context.Context) bool
	Discover(context.Context) ([]Skill, error)
	Resolve(context.Context, string) (Skill, bool, error)
}

var ErrSourceUnavailable = errors.New("skill source unavailable")

func normalizeSkill(skill Skill, sourceID string) (Skill, error) {
	skill.ID = strings.TrimSpace(skill.ID)
	if skill.ID == "" {
		return Skill{}, fmt.Errorf("skill id is required")
	}
	if skill.Source == "" {
		skill.Source = SourceID(sourceID)
	}
	if skill.Name == "" {
		skill.Name = skill.ID
	}
	if skill.Version == "" {
		skill.Version = "1.0.0"
	}
	if skill.Availability == "" {
		skill.Availability = AvailabilityAvailable
	}
	if skill.Mode == "" {
		skill.Mode = "task"
	}
	if skill.Provenance.SourceID == "" {
		skill.Provenance.SourceID = string(skill.Source)
	}
	if skill.Provenance.ObservedAt.IsZero() {
		skill.Provenance.ObservedAt = time.Now().UTC()
	}
	if skill.Hash == "" {
		// Hash content only — ObservedAt and Copies must not destabilize the digest.
		canonical := struct {
			ID           string       `json:"id"`
			Name         string       `json:"name"`
			Description  string       `json:"description,omitempty"`
			Source       SourceID     `json:"source"`
			Version      string       `json:"version,omitempty"`
			Capabilities []string     `json:"capabilities,omitempty"`
			Triggers     []string     `json:"triggers,omitempty"`
			Instructions string       `json:"instructions,omitempty"`
			Dependencies []string     `json:"dependencies,omitempty"`
			Permissions  []string     `json:"permissions,omitempty"`
			Path         string       `json:"path,omitempty"`
			Availability Availability `json:"availability"`
			Mode         string       `json:"activation_mode,omitempty"`
		}{
			ID: skill.ID, Name: skill.Name, Description: skill.Description, Source: skill.Source,
			Version: skill.Version, Capabilities: skill.Capabilities, Triggers: skill.Triggers,
			Instructions: skill.Instructions, Dependencies: skill.Dependencies, Permissions: skill.Permissions,
			Path: skill.Provenance.Path, Availability: skill.Availability, Mode: skill.Mode,
		}
		raw, err := json.Marshal(canonical)
		if err != nil {
			return Skill{}, fmt.Errorf("hash skill %s: %w", skill.ID, err)
		}
		digest := sha256.Sum256(raw)
		skill.Hash = hex.EncodeToString(digest[:])
	}
	return skill, nil
}
