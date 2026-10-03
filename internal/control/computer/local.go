package computer

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
)

const helperProtocolVersion = 1

// NewLocalProvider discovers a native helper without making it a hard runtime
// dependency. v1.1 will ship/version platform helpers separately; until then
// capability discovery remains honest and returns unavailable.
func NewLocalProvider() Provider {
	if override := strings.TrimSpace(os.Getenv("NEXUS_COMPUTER_HELPER")); override != "" {
		if path, err := exec.LookPath(override); err == nil {
			return NewHelperProvider(path, runtime.GOOS)
		}
	}
	for _, candidate := range []string{
		"nexus-computer-use-" + runtime.GOOS,
		"nexus-computer-use",
	} {
		if path, err := exec.LookPath(candidate); err == nil {
			return NewHelperProvider(path, runtime.GOOS)
		}
	}
	return UnavailableProvider{
		Platform: runtime.GOOS,
		Reason:   "no version-compatible Nexus computer-use helper was discovered",
	}
}
