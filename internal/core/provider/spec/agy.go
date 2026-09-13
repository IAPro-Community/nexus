package spec

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kivervinicius/ai-cli/internal/runtime"
)

var agySpec = Spec{
	ID:               "agy",
	BinaryCandidates: []string{"agy"},
	EnvKeys: func(profileHome, binDir, profileName string) map[string]string {
		internalBin, _ := runtime.InternalBinDir()
		pathBins := make([]string, 0, 2)
		if strings.TrimSpace(internalBin) != "" {
			pathBins = append(pathBins, internalBin)
		}
		if strings.TrimSpace(binDir) != "" {
			pathBins = append(pathBins, binDir)
		}
		env := map[string]string{
			"HOME":                             profileHome,
			"XDG_CONFIG_HOME":                  filepath.Join(profileHome, ".config"),
			"XDG_CACHE_HOME":                   filepath.Join(profileHome, ".cache"),
			"XDG_DATA_HOME":                    filepath.Join(profileHome, ".local", "share"),
			"XDG_STATE_HOME":                   filepath.Join(profileHome, ".local", "state"),
			"AI_PROFILE":                       profileName,
			"AI_PROVIDER":                      "agy",
			"PYTHON_KEYRING_BACKEND":           "keyring.backends.null.Keyring",
			"AI_HOST_DBUS_SESSION_BUS_ADDRESS": os.Getenv("DBUS_SESSION_BUS_ADDRESS"),
			"DBUS_SESSION_BUS_ADDRESS":         "unix:path=/dev/null",
			"PATH":                             runtime.EnhancedPATH(pathBins...),
		}
		if strings.TrimSpace(internalBin) != "" {
			env["BROWSER"] = filepath.Join(internalBin, "ai-browser")
		} else if strings.TrimSpace(binDir) != "" {
			// Contract/tests may lack InternalBinDir; never emit a bare relative name.
			env["BROWSER"] = filepath.Join(binDir, "ai-browser")
		}
		return env
	},
	EnvDelete: []string{"GNOME_KEYRING_CONTROL", "GNOME_KEYRING_PID"},
	Kickoff: func(prompt string) []string {
		return []string{"-p", prompt}
	},
	Resume: func(sessionID string) []string {
		return []string{"--conversation=" + sessionID}
	},
	AutoApproval:  []string{"--dangerously-skip-permissions", "--sandbox", "--print-timeout", "60m"},
	ReviewMode:    nil,
	Quota:         QuotaLiveCLI,
	KnownVersions: []string{"1.0.0"},
}
