package launcher

import (
	"encoding/json"
	"github.com/IAPro-Community/nexus/internal/browsercontrol"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func setupBrowserEndpoint(t *testing.T) string {
	t.Helper()
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/health" {
			w.WriteHeader(http.StatusOK)
		} else {
			http.NotFound(w, r)
		}
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close(); browsercontrol.UnpublishBrowserEndpoint(os.Getpid()) })
	if err := browsercontrol.PublishBrowserEndpoint(listener.Addr().(*net.TCPAddr).Port, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	url := browsercontrol.DiscoverBrowserEndpoint()
	if url == "" {
		t.Fatal("discovery failed")
	}
	return url
}
func TestBrowserAutomaticMCPProviderConfig(t *testing.T) {
	url := setupBrowserEndpoint(t)
	for _, provider := range []string{"codex", "opencode"} {
		t.Run(provider, func(t *testing.T) {
			opts := LaunchOptions{RuntimeID: "test-" + provider, ProjectID: "test-project", ProviderID: provider}
			args, env, err := configureBrowserMCP(opts, []string{"AI_PROVIDER=" + provider}, []string{"exec", "sample"})
			if err != nil {
				t.Fatal(err)
			}
			secret := environmentValue(env, "NEXUS_BROWSER_TOKEN")
			if !strings.HasPrefix(secret, "nbr_") || environmentValue(env, "NEXUS_BROWSER_MCP_URL") != url {
				t.Fatal("runtime bridge mismatch")
			}
			if provider == "codex" {
				if len(args) < 6 || args[0] != "-c" || !strings.Contains(strings.Join(args, " "), "bearer_token_env_var") {
					t.Fatalf("Codex MCP flags missing: %v", args)
				}
				if strings.Contains(strings.Join(args, " "), secret) {
					t.Fatal("token exposed in CLI arguments")
				}
			} else {
				var doc map[string]any
				if err := json.Unmarshal([]byte(environmentValue(env, "OPENCODE_CONFIG_CONTENT")), &doc); err != nil {
					t.Fatal(err)
				}
				if doc["mcp"] == nil {
					t.Fatal("OpenCode MCP missing")
				}
				if strings.Contains(environmentValue(env, "OPENCODE_CONFIG_CONTENT"), secret) {
					t.Fatal("token exposed in OpenCode configuration")
				}
			}
			browsercontrol.RevokeRuntimeCapability(opts.RuntimeID)
		})
	}
}
func TestBrowserAutomaticMCPMissionRestriction(t *testing.T) {
	_ = setupBrowserEndpoint(t)
	opts := LaunchOptions{RuntimeID: "mission-1", ProjectID: "project-1", ProviderID: "codex", Options: map[string]any{"nexus_browser_enabled": false}}
	args, env, err := configureBrowserMCP(opts, nil, []string{"exec"})
	if err != nil || len(args) != 1 || len(env) != 0 {
		t.Fatalf("restricted mission gained browser: %v %v %v", args, env, err)
	}
}
func TestBrowserAutomaticMCPStdioProviders(t *testing.T) {
	_ = setupBrowserEndpoint(t)
	if runtime.GOOS == "windows" {
		t.Skip("Windows covered by platform CI")
	}
	dir := t.TempDir()
	fake := filepath.Join(dir, "nexus")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, provider := range []string{"claude", "gemini", "agy", "cursor"} {
		t.Run(provider, func(t *testing.T) {
			home := t.TempDir()
			opts := LaunchOptions{RuntimeID: "test-" + provider, ProjectID: "project-a", ProviderID: provider}
			args, env, err := configureBrowserMCP(opts, []string{"HOME=" + home, "CURSOR_HOME=" + home}, []string{})
			if err != nil {
				t.Fatal(err)
			}
			if environmentValue(env, "NEXUS_BROWSER_TOKEN") == "" {
				t.Fatal("token missing")
			}
			if provider == "claude" && len(args) != 2 {
				t.Fatalf("claude args: %v", args)
			}
			if provider == "gemini" {
				if _, err := os.Stat(filepath.Join(home, ".gemini", "settings.json")); err != nil {
					t.Fatal(err)
				}
			}
			if provider == "agy" {
				if _, err := os.Stat(filepath.Join(home, ".gemini", "config", "mcp_config.json")); err != nil {
					t.Fatal(err)
				}
			}
			if provider == "cursor" {
				if _, err := os.Stat(filepath.Join(home, ".cursor", "mcp.json")); err != nil {
					t.Fatal(err)
				}
			}
			browsercontrol.RevokeRuntimeCapability(opts.RuntimeID)
		})
	}
}
