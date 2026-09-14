package flags

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
)

// IsHelpFlag checks whether the argument list contains a request for help.
func IsHelpFlag(args []string) bool {
	for _, a := range args {
		if a == "--help" || a == "-h" || a == "help" {
			return true
		}
	}
	return false
}

// ShowProviderHelp captures the native help of the provider and prints the enriched merged help.
func ShowProviderHelp(providerID string) error {
	providerID = strings.ToLower(strings.TrimSpace(providerID))
	var nativeHelp string
	if s, ok := spec.Get(providerID); ok {
		if bin, err := s.ResolveBinary(); err == nil && bin != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, bin, "--help")
			cmd.Env = os.Environ()
			out, _ := cmd.CombinedOutput()
			nativeHelp = string(out)
		}
	}

	merged := RenderMergedHelp(providerID, nativeHelp)
	fmt.Print(merged)
	return nil
}
