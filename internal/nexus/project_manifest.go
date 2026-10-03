package nexus

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	DefaultTarget string `yaml:"defaultTarget" json:"default_target"`
}

type ProjectManifestRouting struct {
	Strategy string `yaml:"strategy" json:"strategy"`
}

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
	if err := validateProjectManifest(manifest); err != nil {
		return ProjectManifest{}, err
	}
	return manifest, nil
}

func validateProjectManifest(manifest ProjectManifest) error {
	for _, item := range append(append([]string{}, manifest.Workspace.SharedDirectories...), manifest.Workspace.CopiedFiles...) {
		if err := validateManifestRelativePath(item); err != nil {
			return err
		}
	}
	if len(manifest.Setup.Commands) > 64 || len(manifest.Validation.Commands) > 64 {
		return fmt.Errorf("project manifest command list exceeds 64 entries")
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
	return nil
}
