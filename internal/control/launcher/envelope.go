package launcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
)

// launchEnvelope keeps opaque provider arguments out of the runtime registry.
// It is private, one-time-use material consumed only by the detached host.
type launchEnvelope struct {
	Args          []string `json:"args"`
	CustomCommand bool     `json:"custom_command,omitempty"`
	BrowserEnv map[string]string `json:"browser_env,omitempty"`
}

func launchEnvelopePath(runtimeID string) (string, error) {
	if strings.TrimSpace(runtimeID) == "" || filepath.Base(runtimeID) != runtimeID {
		return "", fmt.Errorf("invalid runtime ID for launch envelope")
	}
	dataDir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "runtime", "launches", runtimeID+".json"), nil
}

func createLaunchEnvelope(runtimeID string, args []string, customCommand bool) error {
	return createLaunchEnvelopeWithBrowser(runtimeID,args,customCommand,nil)
}

func createLaunchEnvelopeWithBrowser(runtimeID string,args []string,customCommand bool,env []string) error {
	path, err := launchEnvelopePath(runtimeID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	browserEnv := map[string]string{}
	for _, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 { continue }
		switch parts[0] {
		case "NEXUS_BROWSER_TOKEN", "NEXUS_BROWSER_MCP_URL", "OPENCODE_CONFIG_CONTENT":
			browserEnv[parts[0]] = parts[1]
		}
	}
	raw, err := json.Marshal(launchEnvelope{Args: args, CustomCommand: customCommand, BrowserEnv:browserEnv})
	if err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if _, err := file.Write(raw); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

// ConsumeLaunchEnvelope reads and removes an envelope before launching a host.
func ConsumeLaunchEnvelope(runtimeID string) ([]string, bool, error) {
	args, custom, _, err := ConsumeLaunchEnvelopeWithBrowser(runtimeID)
	return args, custom, err
}

func ConsumeLaunchEnvelopeWithBrowser(runtimeID string) ([]string, bool, map[string]string, error) {
	path, err := launchEnvelopePath(runtimeID)
	if err != nil {
		return nil, false, nil, err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, false, nil, err
	}
	if err := os.Remove(path); err != nil {
		return nil, false, nil, err
	}
	var envelope launchEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, false, nil, err
	}
	return envelope.Args, envelope.CustomCommand, envelope.BrowserEnv, nil
}

func removeLaunchEnvelope(runtimeID string) {
	if path, err := launchEnvelopePath(runtimeID); err == nil {
		_ = os.Remove(path)
	}
}

func cleanupOrphanLaunchEnvelopes(maxAge time.Duration) {
	dataDir, err := config.DataDir()
	if err != nil {
		return
	}
	dir := filepath.Join(dataDir, "runtime", "launches")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		if info, statErr := entry.Info(); statErr == nil && info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}
