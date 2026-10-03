package nexus

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxProjectManifestBytes = 1 << 20

type ProjectManifest struct {
	Version    int                      `yaml:"version" json:"version"`
	Project    ProjectManifestProject   `yaml:"project" json:"project"`
	Workspace  ProjectManifestWorkspace `yaml:"workspace" json:"workspace"`
	Setup      ProjectManifestCommands  `yaml:"setup" json:"setup"`
	Validation ProjectManifestCommands  `yaml:"validation" json:"validation"`
	Execution  ProjectManifestExecution `yaml:"execution" json:"execution"`
	Routing    ProjectManifestRouting   `yaml:"routing" json:"routing"`
	Commands   []ProjectManifestCommand `yaml:"commands" json:"commands,omitempty"`
}

type ProjectManifestProject struct {
	Name string `yaml:"name" json:"name"`
}

type ProjectManifestWorkspace struct {
	Strategy          string   `yaml:"strategy" json:"strategy"`
	BaseRef           string   `yaml:"baseRef" json:"base_ref"`
	SharedDirectories []string `yaml:"sharedDirectories" json:"shared_directories"`
	CopiedFiles       []string `yaml:"copiedFiles" json:"copied_files"`
}

type ProjectManifestCommands struct {
	Commands []string `yaml:"commands" json:"commands"`
}

type ProjectManifestExecution struct {
	DefaultTarget string                  `yaml:"defaultTarget" json:"default_target"`
	Targets       []ProjectManifestTarget `yaml:"targets,omitempty" json:"targets,omitempty"`
}

// ProjectManifestTarget is a named execution target declared by the Project.
// v1.1 initially accepts managed SSH descriptors only. Declaring a target does
// not execute it; activation remains an explicit, feature-gated step.
type ProjectManifestTarget struct {
	ID        string `yaml:"id" json:"id"`
	Kind      string `yaml:"kind" json:"kind"`
	Location  string `yaml:"location" json:"location"`
	MachineID string `yaml:"machineId,omitempty" json:"machine_id,omitempty"`
	Workspace string `yaml:"workspace" json:"workspace"`
}

type ProjectManifestRouting struct {
	Strategy string `yaml:"strategy" json:"strategy"`
}

type ProjectManifestCommand struct {
	ID          string   `yaml:"id" json:"id"`
	Label       string   `yaml:"label" json:"label"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Command     string   `yaml:"command" json:"command"`
	Keywords    []string `yaml:"keywords,omitempty" json:"keywords,omitempty"`
}

var (
	projectManifestCommandIDPattern    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}package nexus

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxProjectManifestBytes = 1 << 20

type ProjectManifest struct {
	Version    int                      `yaml:"version" json:"version"`
	Project    ProjectManifestProject   `yaml:"project" json:"project"`
	Workspace  ProjectManifestWorkspace `yaml:"workspace" json:"workspace"`
	Setup      ProjectManifestCommands  `yaml:"setup" json:"setup"`
	Validation ProjectManifestCommands  `yaml:"validation" json:"validation"`
	Execution  ProjectManifestExecution `yaml:"execution" json:"execution"`
	Routing    ProjectManifestRouting   `yaml:"routing" json:"routing"`
	Commands   []ProjectManifestCommand `yaml:"commands" json:"commands,omitempty"`
}

type ProjectManifestProject struct {
	Name string `yaml:"name" json:"name"`
}

type ProjectManifestWorkspace struct {
	Strategy          string   `yaml:"strategy" json:"strategy"`
	BaseRef           string   `yaml:"baseRef" json:"base_ref"`
	SharedDirectories []string `yaml:"sharedDirectories" json:"shared_directories"`
	CopiedFiles       []string `yaml:"copiedFiles" json:"copied_files"`
}

type ProjectManifestCommands struct {
	Commands []string `yaml:"commands" json:"commands"`
}

type ProjectManifestExecution struct {
	DefaultTarget string                  `yaml:"defaultTarget" json:"default_target"`
	Targets       []ProjectManifestTarget `yaml:"targets,omitempty" json:"targets,omitempty"`
}

// ProjectManifestTarget is a named execution target declared by the Project.
// v1.1 initially accepts managed SSH descriptors only. Declaring a target does
// not execute it; activation remains an explicit, feature-gated step.
type ProjectManifestTarget struct {
	ID        string `yaml:"id" json:"id"`
	Kind      string `yaml:"kind" json:"kind"`
	Location  string `yaml:"location" json:"location"`
	MachineID string `yaml:"machineId,omitempty" json:"machine_id,omitempty"`
	Workspace string `yaml:"workspace" json:"workspace"`
}

type ProjectManifestRouting struct {
	Strategy string `yaml:"strategy" json:"strategy"`
}

type ProjectManifestCommand struct {
	ID          string   `yaml:"id" json:"id"`
	Label       string   `yaml:"label" json:"label"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Command     string   `yaml:"command" json:"command"`
	Keywords    []string `yaml:"keywords,omitempty" json:"keywords,omitempty"`
}

)
	projectManifestTargetIDPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}package nexus

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxProjectManifestBytes = 1 << 20

type ProjectManifest struct {
	Version    int                      `yaml:"version" json:"version"`
	Project    ProjectManifestProject   `yaml:"project" json:"project"`
	Workspace  ProjectManifestWorkspace `yaml:"workspace" json:"workspace"`
	Setup      ProjectManifestCommands  `yaml:"setup" json:"setup"`
	Validation ProjectManifestCommands  `yaml:"validation" json:"validation"`
	Execution  ProjectManifestExecution `yaml:"execution" json:"execution"`
	Routing    ProjectManifestRouting   `yaml:"routing" json:"routing"`
	Commands   []ProjectManifestCommand `yaml:"commands" json:"commands,omitempty"`
}

type ProjectManifestProject struct {
	Name string `yaml:"name" json:"name"`
}

type ProjectManifestWorkspace struct {
	Strategy          string   `yaml:"strategy" json:"strategy"`
	BaseRef           string   `yaml:"baseRef" json:"base_ref"`
	SharedDirectories []string `yaml:"sharedDirectories" json:"shared_directories"`
	CopiedFiles       []string `yaml:"copiedFiles" json:"copied_files"`
}

type ProjectManifestCommands struct {
	Commands []string `yaml:"commands" json:"commands"`
}

type ProjectManifestExecution struct {
	DefaultTarget string                  `yaml:"defaultTarget" json:"default_target"`
	Targets       []ProjectManifestTarget `yaml:"targets,omitempty" json:"targets,omitempty"`
}

// ProjectManifestTarget is a named execution target declared by the Project.
// v1.1 initially accepts managed SSH descriptors only. Declaring a target does
// not execute it; activation remains an explicit, feature-gated step.
type ProjectManifestTarget struct {
	ID        string `yaml:"id" json:"id"`
	Kind      string `yaml:"kind" json:"kind"`
	Location  string `yaml:"location" json:"location"`
	MachineID string `yaml:"machineId,omitempty" json:"machine_id,omitempty"`
	Workspace string `yaml:"workspace" json:"workspace"`
}

type ProjectManifestRouting struct {
	Strategy string `yaml:"strategy" json:"strategy"`
}

type ProjectManifestCommand struct {
	ID          string   `yaml:"id" json:"id"`
	Label       string   `yaml:"label" json:"label"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Command     string   `yaml:"command" json:"command"`
	Keywords    []string `yaml:"keywords,omitempty" json:"keywords,omitempty"`
}

)
	projectManifestTargetLocationPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:@%+\[\]-]{0,239}package nexus

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const maxProjectManifestBytes = 1 << 20

type ProjectManifest struct {
	Version    int                      `yaml:"version" json:"version"`
	Project    ProjectManifestProject   `yaml:"project" json:"project"`
	Workspace  ProjectManifestWorkspace `yaml:"workspace" json:"workspace"`
	Setup      ProjectManifestCommands  `yaml:"setup" json:"setup"`
	Validation ProjectManifestCommands  `yaml:"validation" json:"validation"`
	Execution  ProjectManifestExecution `yaml:"execution" json:"execution"`
	Routing    ProjectManifestRouting   `yaml:"routing" json:"routing"`
	Commands   []ProjectManifestCommand `yaml:"commands" json:"commands,omitempty"`
}

type ProjectManifestProject struct {
	Name string `yaml:"name" json:"name"`
}

type ProjectManifestWorkspace struct {
	Strategy          string   `yaml:"strategy" json:"strategy"`
	BaseRef           string   `yaml:"baseRef" json:"base_ref"`
	SharedDirectories []string `yaml:"sharedDirectories" json:"shared_directories"`
	CopiedFiles       []string `yaml:"copiedFiles" json:"copied_files"`
}

type ProjectManifestCommands struct {
	Commands []string `yaml:"commands" json:"commands"`
}

type ProjectManifestExecution struct {
	DefaultTarget string                  `yaml:"defaultTarget" json:"default_target"`
	Targets       []ProjectManifestTarget `yaml:"targets,omitempty" json:"targets,omitempty"`
}

// ProjectManifestTarget is a named execution target declared by the Project.
// v1.1 initially accepts managed SSH descriptors only. Declaring a target does
// not execute it; activation remains an explicit, feature-gated step.
type ProjectManifestTarget struct {
	ID        string `yaml:"id" json:"id"`
	Kind      string `yaml:"kind" json:"kind"`
	Location  string `yaml:"location" json:"location"`
	MachineID string `yaml:"machineId,omitempty" json:"machine_id,omitempty"`
	Workspace string `yaml:"workspace" json:"workspace"`
}

type ProjectManifestRouting struct {
	Strategy string `yaml:"strategy" json:"strategy"`
}

type ProjectManifestCommand struct {
	ID          string   `yaml:"id" json:"id"`
	Label       string   `yaml:"label" json:"label"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Command     string   `yaml:"command" json:"command"`
	Keywords    []string `yaml:"keywords,omitempty" json:"keywords,omitempty"`
}

)
)
func LoadProjectManifest(projectPath string) (ProjectManifest, string, error) {
	for _, name := range []string{"nexus.yaml", "nexus.yml"} {
		path := filepath.Join(projectPath, name)
		manifest, err := loadProjectManifestFile(path)
		if err == nil {
			return manifest, path, nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return ProjectManifest{}, path, err
		}
	}
	return ProjectManifest{}, "", os.ErrNotExist
}

func loadProjectManifestFile(path string) (ProjectManifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return ProjectManifest{}, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(io.LimitReader(file, maxProjectManifestBytes+1))
	decoder.KnownFields(true)
	var manifest ProjectManifest
	if err := decoder.Decode(&manifest); err != nil {
		return ProjectManifest{}, fmt.Errorf("decode Nexus project manifest: %w", err)
	}
	if manifest.Version == 0 {
		manifest.Version = 1
	}
	if manifest.Version != 1 {
		return ProjectManifest{}, fmt.Errorf("unsupported Nexus project manifest version %d", manifest.Version)
	}
	if err := validateProjectManifest(&manifest); err != nil {
		return ProjectManifest{}, err
	}
	return manifest, nil
}

func validateProjectManifest(manifest *ProjectManifest) error {
	if manifest == nil {
		return fmt.Errorf("project manifest is required")
	}
	for _, item := range append(append([]string{}, manifest.Workspace.SharedDirectories...), manifest.Workspace.CopiedFiles...) {
		if err := validateManifestRelativePath(item); err != nil {
			return err
		}
	}
	if len(manifest.Setup.Commands) > 64 || len(manifest.Validation.Commands) > 64 {
		return fmt.Errorf("project manifest command list exceeds 64 entries")
	}
	if len(manifest.Commands) > 32 {
		return fmt.Errorf("project manifest quick command list exceeds 32 entries")
	}
	manifest.Execution.DefaultTarget = strings.TrimSpace(manifest.Execution.DefaultTarget)
	if manifest.Execution.DefaultTarget == "" {
		manifest.Execution.DefaultTarget = "local"
	}
	if len(manifest.Execution.Targets) > 16 {
		return fmt.Errorf("project manifest execution target list exceeds 16 entries")
	}
	seenTargetIDs := make(map[string]struct{}, len(manifest.Execution.Targets))
	for i := range manifest.Execution.Targets {
		target := &manifest.Execution.Targets[i]
		target.ID = strings.TrimSpace(target.ID)
		target.Kind = strings.ToLower(strings.TrimSpace(target.Kind))
		target.Location = strings.TrimSpace(target.Location)
		target.MachineID = strings.TrimSpace(target.MachineID)
		target.Workspace = strings.TrimSpace(target.Workspace)

		if !projectManifestTargetIDPattern.MatchString(target.ID) || strings.EqualFold(target.ID, "local") {
			return fmt.Errorf("project manifest execution target id %q is invalid or reserved", target.ID)
		}
		if _, exists := seenTargetIDs[target.ID]; exists {
			return fmt.Errorf("project manifest execution target id %q is duplicated", target.ID)
		}
		seenTargetIDs[target.ID] = struct{}{}
		if target.Kind != "ssh" {
			return fmt.Errorf("project manifest execution target %q kind %q is not supported yet", target.ID, target.Kind)
		}
		if !projectManifestTargetLocationPattern.MatchString(target.Location) || strings.HasPrefix(target.Location, "-") {
			return fmt.Errorf("project manifest SSH target %q location %q is invalid", target.ID, target.Location)
		}
		if target.Workspace == "" || len(target.Workspace) > 4096 || strings.ContainsAny(target.Workspace, "\r\n\x00") {
			return fmt.Errorf("project manifest SSH target %q requires a target workspace", target.ID)
		}
		if len(target.MachineID) > 240 || strings.ContainsAny(target.MachineID, "\r\n\x00") {
			return fmt.Errorf("project manifest SSH target %q machineId is invalid", target.ID)
		}
	}
	if manifest.Execution.DefaultTarget != "local" {
		if _, exists := seenTargetIDs[manifest.Execution.DefaultTarget]; !exists {
			return fmt.Errorf("project manifest defaultTarget %q is not declared in execution.targets", manifest.Execution.DefaultTarget)
		}
	}

	seenCommandIDs := make(map[string]struct{}, len(manifest.Commands))
	for i := range manifest.Commands {
		command := &manifest.Commands[i]
		command.ID = strings.TrimSpace(command.ID)
		command.Label = strings.TrimSpace(command.Label)
		command.Description = strings.TrimSpace(command.Description)
		command.Command = strings.TrimSpace(command.Command)
		if !projectManifestCommandIDPattern.MatchString(command.ID) {
			return fmt.Errorf("project manifest quick command id %q is invalid", command.ID)
		}
		if _, exists := seenCommandIDs[command.ID]; exists {
			return fmt.Errorf("project manifest quick command id %q is duplicated", command.ID)
		}
		seenCommandIDs[command.ID] = struct{}{}
		if command.Label == "" || len(command.Label) > 120 {
			return fmt.Errorf("project manifest quick command %q requires a label up to 120 bytes", command.ID)
		}
		if command.Command == "" || len(command.Command) > 4096 {
			return fmt.Errorf("project manifest quick command %q requires a command up to 4096 bytes", command.ID)
		}
		if strings.ContainsAny(command.Command, "\r\n\x00") {
			return fmt.Errorf("project manifest quick command %q must be a single terminal line", command.ID)
		}
		if len(command.Description) > 240 {
			return fmt.Errorf("project manifest quick command %q description exceeds 240 bytes", command.ID)
		}
		if len(command.Keywords) > 12 {
			return fmt.Errorf("project manifest quick command %q has too many keywords", command.ID)
		}
	}
	for _, command := range append(append([]string{}, manifest.Setup.Commands...), manifest.Validation.Commands...) {
		if strings.TrimSpace(command) == "" {
			return fmt.Errorf("project manifest commands must not be empty")
		}
		if len(command) > 4096 {
			return fmt.Errorf("project manifest command exceeds 4096 bytes")
		}
	}
	return nil
}

// ProjectExecutionTarget returns one normalized named target from a loaded
// manifest. "local" is implicit and intentionally has no descriptor.
func (manifest ProjectManifest) ProjectExecutionTarget(id string) (ProjectManifestTarget, bool) {
	id = strings.TrimSpace(id)
	if id == "" || id == "local" {
		return ProjectManifestTarget{}, false
	}
	for _, target := range manifest.Execution.Targets {
		if target.ID == id {
			return target, true
		}
	}
	return ProjectManifestTarget{}, false
}

func validateManifestRelativePath(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("project manifest path must not be empty")
	}
	if filepath.IsAbs(value) {
		return fmt.Errorf("project manifest path %q must be relative", value)
	}
	clean := filepath.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("project manifest path %q escapes the project root", value)
	}
	parts := strings.FieldsFunc(filepath.ToSlash(clean), func(r rune) bool { return r == '/' })
	for _, part := range parts {
		if part == ".git" {
			return fmt.Errorf("project manifest path %q must not target Git metadata", value)
		}
	}
	return nil
}
