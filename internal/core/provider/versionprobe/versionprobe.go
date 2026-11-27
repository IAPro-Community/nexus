package versionprobe

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/model"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

const (
	successCacheTTL = 60 * time.Second
	failureCacheTTL = 10 * time.Second
	probeMaxTimeout = 5 * time.Second
)

type cacheKey struct {
	binary  string
	argsKey string
	modUnix int64
	size    int64
}

type cacheEntry struct {
	result    model.DetectionResult
	expiresAt time.Time
}

type inflightCall struct {
	done   chan struct{}
	result model.DetectionResult
}

var (
	probeMu       sync.Mutex
	probeCache    = make(map[cacheKey]cacheEntry)
	probeInflight = make(map[cacheKey]*inflightCall)
	runCommandFn  = nexusruntime.RunCommandCapture
	nowFn         = time.Now
)

// InvalidateCache clears cached CLI version probes (e.g. after CLI install/upgrade).
func InvalidateCache() {
	probeMu.Lock()
	clear(probeCache)
	probeMu.Unlock()
}

// Detect runs the standard `--version` probe with short-lived binary-fingerprinted caching
// and singleflight coalescing across concurrent callers.
func Detect(ctx context.Context, binary string) model.DetectionResult {
	return DetectArgs(ctx, binary, "--version")
}

// DetectArgs runs a CLI version probe using the given arguments with short-lived
// binary-fingerprinted caching and singleflight coalescing.
func DetectArgs(ctx context.Context, binary string, args ...string) model.DetectionResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return resultFromProbe(binary, "", err)
	}
	if len(args) == 0 {
		args = []string{"--version"}
	}

	key := buildCacheKey(binary, args)
	now := nowFn()

	probeMu.Lock()
	if entry, ok := probeCache[key]; ok && now.Before(entry.expiresAt) {
		probeMu.Unlock()
		return entry.result
	}
	if call, ok := probeInflight[key]; ok {
		probeMu.Unlock()
		select {
		case <-call.done:
			return call.result
		case <-ctx.Done():
			return resultFromProbe(binary, "", ctx.Err())
		}
	}
	call := &inflightCall{done: make(chan struct{})}
	probeInflight[key] = call
	probeMu.Unlock()

	probeCtx, cancel := context.WithTimeout(context.Background(), probeMaxTimeout)
	output, err := runCommandFn(probeCtx, binary, args, os.Environ(), "")
	cancel()

	res := resultFromProbe(binary, output, err)

	probeMu.Lock()
	if err == nil && res.Version != "unknown" {
		probeCache[key] = cacheEntry{result: res, expiresAt: nowFn().Add(successCacheTTL)}
	} else if probeCtx.Err() == nil {
		probeCache[key] = cacheEntry{result: res, expiresAt: nowFn().Add(failureCacheTTL)}
	}
	call.result = res
	close(call.done)
	delete(probeInflight, key)
	probeMu.Unlock()

	if ctxErr := ctx.Err(); ctxErr != nil {
		return resultFromProbe(binary, "", ctxErr)
	}
	return res
}

func buildCacheKey(binary string, args []string) cacheKey {
	var modUnix int64
	var size int64
	if info, err := os.Stat(binary); err == nil {
		modUnix = info.ModTime().UnixNano()
		size = info.Size()
	}
	return cacheKey{
		binary:  binary,
		argsKey: strings.Join(args, "\x00") + ":" + strconv.Itoa(len(args)),
		modUnix: modUnix,
		size:    size,
	}
}

func resultFromProbe(binary, output string, err error) model.DetectionResult {
	if err != nil {
		return model.DetectionResult{
			Installed:  true,
			Version:    "unknown",
			BinaryPath: binary,
			Error:      fmt.Sprintf("version probe failed: %v", err),
		}
	}

	version := strings.TrimSpace(output)
	if version == "" {
		return model.DetectionResult{
			Installed:  true,
			Version:    "unknown",
			BinaryPath: binary,
			Error:      "version probe returned no output",
		}
	}

	return model.DetectionResult{Installed: true, Version: version, BinaryPath: binary}
}
