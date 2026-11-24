// Command fixturecli is a provider-CLI stand-in for tests. It behaves the same
// on Linux, macOS and Windows: no shell script, no PATHEXT dependency.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type response struct {
	WhenEnvContains [][2]string `json:"when_env_contains"`
	Stdout          []string    `json:"stdout"`
	ExitCode        int         `json:"exit_code"`
	SleepMS         int         `json:"sleep_ms"`
}

type spec struct {
	Version   string            `json:"version"`
	ArgsFile  string            `json:"args_file"`
	EnvFile   string            `json:"env_file"`
	EnvKeys   []string          `json:"env_keys"`
	Files     map[string]string `json:"files"`
	Responses []response        `json:"responses"`
	Stdout    []string          `json:"stdout"`
	ExitCode  int               `json:"exit_code"`
	SleepMS   int               `json:"sleep_ms"`
}

func main() {
	var s spec
	if err := json.Unmarshal([]byte(os.Getenv("FIXTURE_CLI_SPEC")), &s); err != nil {
		os.Stderr.WriteString("fixturecli: invalid FIXTURE_CLI_SPEC: " + err.Error() + "\n")
		os.Exit(2)
	}
	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--version" {
			version := s.Version
			if version == "" {
				version = "fixture 1.0"
			}
			emit([]string{version})
			os.Exit(0)
		}
	}
	if s.ArgsFile != "" {
		_ = os.WriteFile(s.ArgsFile, []byte(strings.Join(args, "\x00")+"\x00"), 0o600)
	}
	if s.EnvFile != "" {
		var lines []string
		for _, key := range s.EnvKeys {
			lines = append(lines, key+"="+os.Getenv(key))
		}
		_ = os.WriteFile(s.EnvFile, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
	}
	for path, content := range s.Files {
		path = expandPath(path)
		_ = os.MkdirAll(fileDir(path), 0o700)
		_ = os.WriteFile(path, []byte(content), 0o600)
	}
	chosen := response{Stdout: s.Stdout, ExitCode: s.ExitCode, SleepMS: s.SleepMS}
	for _, candidate := range s.Responses {
		if matches(candidate.WhenEnvContains) {
			chosen = candidate
			break
		}
	}
	emit(chosen.Stdout)
	if chosen.SleepMS > 0 {
		time.Sleep(time.Duration(chosen.SleepMS) * time.Millisecond)
	}
	os.Exit(chosen.ExitCode)
}

func emit(lines []string) {
	for _, line := range lines {
		os.Stdout.WriteString(line + "\n")
	}
}

// matches compares env values with "/" separators so a Windows CODEX_HOME still
// matches a fixture rule written with forward slashes.
func matches(rules [][2]string) bool {
	for _, rule := range rules {
		value := strings.ReplaceAll(os.Getenv(rule[0]), `\`, "/")
		if !strings.Contains(value, rule[1]) {
			return false
		}
	}
	return true
}

// expandPath replaces {VAR} placeholders with fixture environment values and
// turns "|" into the platform separator, so a spec can target files under HOME
// without OS-specific paths.
func expandPath(path string) string {
	path = strings.ReplaceAll(path, "|", "/")
	for {
		start := strings.Index(path, "{")
		if start < 0 {
			return filepath.FromSlash(path)
		}
		end := strings.Index(path[start:], "}")
		if end < 0 {
			return filepath.FromSlash(path)
		}
		name := path[start+1 : start+end]
		path = path[:start] + os.Getenv(name) + path[start+end+1:]
	}
}

// fileDir returns the directory part of a "/"-separated spec path.
func fileDir(path string) string {
	index := strings.LastIndexAny(path, `/\`)
	if index <= 0 {
		return "."
	}
	return path[:index]
}
