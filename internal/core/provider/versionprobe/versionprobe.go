package versionprobe

import (
	"context"
	"os"
	"time"

	sdk "github.com/IAPro-Community/IAPro-SDK/runtime/versionprobe"
	"github.com/IAPro-Community/nexus/internal/core/model"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

// Nexus retains its process runner; caching, coalescing, expiry and
// normalization are implemented only by IAPro-SDK.
var (
	runCommandFn = nexusruntime.RunCommandCapture
	nowFn        = time.Now
	cache        = sdk.New(func(ctx context.Context, binary string, args, env []string, dir string) (string, error) {
		return runCommandFn(ctx, binary, args, env, dir)
	}, func() time.Time { return nowFn() })
)

func InvalidateCache() { cache.Invalidate() }

func Detect(ctx context.Context, binary string) model.DetectionResult {
	return DetectArgs(ctx, binary, "--version")
}

func DetectArgs(ctx context.Context, binary string, args ...string) model.DetectionResult {
	return convert(cache.Detect(ctx, binary, args, os.Environ(), ""))
}

func convert(r sdk.Result) model.DetectionResult {
	return model.DetectionResult{Installed: r.Installed, Version: r.Version, BinaryPath: r.BinaryPath, Error: r.Error}
}

func resultFromProbe(binary, output string, err error) model.DetectionResult {
	return convert(sdk.Normalize(binary, output, err))
}
