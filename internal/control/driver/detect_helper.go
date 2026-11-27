package driver

import (
	"context"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/provider/spec"
	"github.com/IAPro-Community/nexus/internal/core/provider/versionprobe"
)

func detectProviderCLI(ctx context.Context, providerID, missingError string, args ...string) (model.DetectionResult, error) {
	bin, err := spec.MustGet(providerID).ResolveBinary()
	if err != nil {
		return model.DetectionResult{Installed: false, Error: missingError}, nil
	}
	res := versionprobe.DetectArgs(ctx, bin, args...)
	if res.Error != "" || strings.Contains(res.Version, "No such file or directory") {
		errText := strings.TrimSpace(res.Error)
		if strings.Contains(res.Version, "No such file or directory") {
			errText = strings.TrimSpace(res.Version)
		}
		return model.DetectionResult{
			Installed:  false,
			BinaryPath: bin,
			Error:      errText,
		}, nil
	}
	version := res.Version
	for _, line := range strings.Split(strings.TrimSpace(res.Version), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "Tip:") {
			version = line
			break
		}
	}
	return model.DetectionResult{
		Installed:  true,
		BinaryPath: bin,
		Version:    version,
	}, nil
}
