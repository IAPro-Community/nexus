package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ResolvedCommand describes the artifact selected for a provider and the
// launcher required to execute it on the current operating system.
type ResolvedCommand struct {
	ArtifactPath  string   `json:"artifact_path"`
	LauncherPath  string   `json:"launcher_path"`
	PrefixArgs    []string `json:"prefix_args,omitempty"`
	Kind          string   `json:"kind"`
	SearchedPaths []string `json:"searched_paths,omitempty"`
}

const (
	CommandExecutable = "executable"
	CommandCmd        = "cmd"
	CommandBatch      = "batch"
	CommandPowerShell = "powershell"
)

// ResolveCommand finds a provider artifact and describes how it must be
// launched. On Windows PATHEXT and common npm/user bins are considered in a
// deterministic order; on Unix it preserves the existing PATH behavior.
func ResolveCommand(name string) (ResolvedCommand, error) {
	return resolveCommand(name, runtime.GOOS, os.Getenv, os.Stat, exec.LookPath)
}

func resolveCommand(name, goos string, getenv func(string) string, stat func(string) (os.FileInfo, error), lookPath func(string) (string, error)) (ResolvedCommand, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return ResolvedCommand{}, fmt.Errorf("executable name is empty")
	}

	searched := make([]string, 0, 12)
	if goos != "windows" {
		path, err := lookPath(name)
		if err != nil {
			return ResolvedCommand{SearchedPaths: searched}, err
		}
		return ResolvedCommand{ArtifactPath: path, LauncherPath: path, Kind: CommandExecutable, SearchedPaths: []string{path}}, nil
	}

	pathext := splitPathList(getenv("PATHEXT"), windowsListSeparator)
	if len(pathext) == 0 {
		pathext = []string{".COM", ".EXE", ".BAT", ".CMD", ".PS1"}
	}
	candidates := windowsCandidates(name, getenv, pathext, goos)
	for _, candidate := range candidates {
		searched = append(searched, candidate)
		info, err := stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		return windowsResolved(candidate, getenv, lookPath, pathext), nil
	}
	return ResolvedCommand{SearchedPaths: searched}, exec.ErrNotFound
}

func windowsCandidates(name string, getenv func(string) string, pathext []string, goos string) []string {
	if filepath.IsAbs(name) || strings.ContainsAny(name, `\/`) {
		return withExtensions(name, pathext)
	}
	paths := splitPathList(getenv("PATH"), pathListSeparatorForGOOS(goos))
	paths = append(paths,
		filepath.Join(getenv("APPDATA"), "npm"),
		filepath.Join(getenv("LOCALAPPDATA"), "Programs"),
		filepath.Join(getenv("USERPROFILE"), "AppData", "Roaming", "npm"),
	)
	var candidates []string
	for _, dir := range paths {
		if strings.TrimSpace(dir) == "" {
			continue
		}
		candidates = append(candidates, withExtensions(filepath.Join(dir, name), pathext)...)
	}
	return dedupeStrings(candidates)
}

func withExtensions(path string, pathext []string) []string {
	if strings.Contains(filepath.Base(path), ".") {
		return []string{path}
	}
	result := []string{path}
	for _, ext := range pathext {
		result = append(result, path+strings.ToLower(ext))
	}
	return result
}

func windowsResolved(artifact string, getenv func(string) string, lookPath func(string) (string, error), _ []string) ResolvedCommand {
	ext := strings.ToLower(filepath.Ext(artifact))
	switch ext {
	case ".cmd":
		launcher := getenv("COMSPEC")
		if launcher == "" {
			launcher = "cmd.exe"
		}
		return ResolvedCommand{ArtifactPath: artifact, LauncherPath: launcher, PrefixArgs: []string{"/D", "/S", "/C", artifact}, Kind: CommandCmd}
	case ".bat":
		launcher := getenv("COMSPEC")
		if launcher == "" {
			launcher = "cmd.exe"
		}
		return ResolvedCommand{ArtifactPath: artifact, LauncherPath: launcher, PrefixArgs: []string{"/D", "/S", "/C", artifact}, Kind: CommandBatch}
	case ".ps1":
		launcher := "pwsh.exe"
		if path, err := lookPath(launcher); err == nil {
			launcher = path
		} else if path, err := lookPath("powershell.exe"); err == nil {
			launcher = path
		}
		return ResolvedCommand{ArtifactPath: artifact, LauncherPath: launcher, PrefixArgs: []string{"-File", artifact}, Kind: CommandPowerShell}
	default:
		return ResolvedCommand{ArtifactPath: artifact, LauncherPath: artifact, Kind: CommandExecutable}
	}
}

// windowsListSeparator is the PATH/PATHEXT list separator on Windows.
// It is a constant (not os.PathListSeparator) because resolveCommand can
// target Windows while the host runs another OS (e.g. in tests).
const windowsListSeparator = ";"

// pathListSeparatorForGOOS returns the PATH list separator for the target
// OS, so cross-compiled/simulated resolution never inherits the host value.
func pathListSeparatorForGOOS(goos string) string {
	if goos == "windows" {
		return windowsListSeparator
	}
	return ":"
}

func splitPathList(value, separator string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, separator)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func dedupeStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
