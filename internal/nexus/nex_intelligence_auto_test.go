package nexus

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	coreconfig "github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/nexus/intelligence"
	"github.com/IAPro-Community/nexus/internal/profile"
)

func installFakeCodexCLI(t *testing.T, argsCapturePath, replyText string) string {
	t.Helper()
	binDir := t.TempDir()
	scriptPath := filepath.Join(binDir, "codex")
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then\n" +
		"  echo \"codex-cli 0.99.0\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"printf '%s\\n' \"$*\" > " + shellQuoteTest(argsCapturePath) + "\n" +
		"printf '%s\\n' " + shellQuoteTest(replyText) + "\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake codex CLI: %v", err)
	}
	return binDir
}

func shellQuoteTest(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func setupIsolatedNexIntelligenceEnv(t *testing.T, withBinDir string) string {
	t.Helper()
	cfgDir := t.TempDir()
	dataDir := t.TempDir()
	homeDir := t.TempDir()
	t.Setenv("NEXUS_CONFIG_DIR", cfgDir)
	t.Setenv("NEXUS_DATA_DIR", dataDir)
	t.Setenv("HOME", homeDir)
	t.Setenv("NEXUS_REAL_HOME", "")
	if withBinDir != "" {
		t.Setenv("PATH", withBinDir+":"+os.Getenv("PATH"))
	} else {
		t.Setenv("PATH", t.TempDir())
	}
	return dataDir
}

func seedAuthenticatedCodexProfile(t *testing.T, profileName string) {
	t.Helper()
	if _, err := profile.Create("codex", profileName); err != nil {
		t.Fatalf("create codex profile: %v", err)
	}
	home, err := coreconfig.ProfileHome("codex", profileName)
	if err != nil {
		t.Fatalf("profile home: %v", err)
	}
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatalf("mkdir profile home: %v", err)
	}
	payload, err := json.Marshal(map[string]any{
		"email": "nex-test@example.com",
		"https://api.openai.com/auth": map[string]any{
			"chatgpt_account_id": "acc-nex-test",
			"chatgpt_plan_type":  "plus",
		},
	})
	if err != nil {
		t.Fatalf("marshal auth payload: %v", err)
	}
	jwt := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`)) + "." +
		base64.RawURLEncoding.EncodeToString(payload) + ".sig"
	authFile, err := json.Marshal(map[string]any{
		"tokens": map[string]any{
			"id_token":      jwt,
			"access_token":  "acc",
			"refresh_token": "ref",
		},
	})
	if err != nil {
		t.Fatalf("marshal auth.json: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, "auth.json"), authFile, 0o600); err != nil {
		t.Fatalf("write auth.json: %v", err)
	}
}

func TestResolveNexChatProviderAutoUsesReadyReadOnlyCodexAndEnforcesSandboxArgs(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "codex-args.txt")
	binDir := installFakeCodexCLI(t, argsFile, "Cotas normais em codex:work.")
	setupIsolatedNexIntelligenceEnv(t, binDir)
	seedAuthenticatedCodexProfile(t, "work")

	n := NewWithStoreAndExecutor(nil, nil)
	prov, source, assistance, err := n.ResolveNexChatProvider(context.Background(), "")
	if err != nil {
		t.Fatalf("expected auto resolution to succeed, got err=%v assistance=%+v", err, assistance)
	}
	if source != "auto_cli_readonly" {
		t.Fatalf("expected auto_cli_readonly source, got %q", source)
	}
	if assistance != nil {
		t.Fatalf("expected nil assistance when resolved, got %+v", assistance)
	}
	cliProv, ok := prov.(*intelligence.CLIProvider)
	if !ok || !cliProv.SupportsNexReadOnly() {
		t.Fatalf("expected CLIProvider with Nex read-only support, got %T", prov)
	}
	reply, err := cliProv.NexReply(context.Background(), "persona", "como estao minhas cotas?")
	if err != nil {
		t.Fatalf("NexReply error: %v", err)
	}
	if reply != "Cotas normais em codex:work." {
		t.Fatalf("unexpected reply: %q", reply)
	}
	rawArgs, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("read captured args: %v", err)
	}
	invoked := string(rawArgs)
	if !strings.Contains(invoked, "--sandbox read-only --ask-for-approval never exec") {
		t.Fatalf("expected read-only sandbox flags before exec, got %q", invoked)
	}
}

func TestResolveNexChatProviderReturnsSetupAssistanceWhenNoReadyProvider(t *testing.T) {
	setupIsolatedNexIntelligenceEnv(t, "")
	if _, err := profile.Create("codex", "default"); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	n := NewWithStoreAndExecutor(nil, nil)
	prov, source, assistance, err := n.ResolveNexChatProvider(context.Background(), "")
	if !errors.Is(err, intelligence.ErrIntelligenceUnavailable) {
		t.Fatalf("expected ErrIntelligenceUnavailable, got %v (prov=%v source=%q)", err, prov, source)
	}
	if assistance == nil {
		t.Fatal("expected structured setup assistance when no provider is available")
	}
	if assistance.CanAutoConfigure {
		t.Fatalf("expected CanAutoConfigure=false when binary is absent, got %+v", assistance)
	}
	if len(assistance.Candidates) != 1 || assistance.Candidates[0].Reason != "cli_not_installed" {
		t.Fatalf("unexpected setup candidates: %+v", assistance.Candidates)
	}
}

func TestAutoConfigureNexIntelligenceAutoAndOpenAI(t *testing.T) {
	argsFile := filepath.Join(t.TempDir(), "codex-args.txt")
	binDir := installFakeCodexCLI(t, argsFile, "ok")
	setupIsolatedNexIntelligenceEnv(t, binDir)
	seedAuthenticatedCodexProfile(t, "auto-prof")

	n := NewWithStoreAndExecutor(nil, nil)
	status, err := n.AutoConfigureNexIntelligence(context.Background(), AutoConfigureIntelligenceRequest{
		Mode: "AUTO",
	})
	if err != nil {
		t.Fatalf("AutoConfigure AUTO failed: %v", err)
	}
	if status.Mode != coreconfig.IntelligenceCLI || status.Provider != "codex" || status.Profile != "auto-prof" || !status.Available {
		t.Fatalf("unexpected AUTO status: %+v", status)
	}

	// Empty OpenAI base_url must fail validation cleanly.
	if _, err := n.AutoConfigureNexIntelligence(context.Background(), AutoConfigureIntelligenceRequest{
		Mode: coreconfig.IntelligenceOpenAICompatible,
	}); err == nil {
		t.Fatal("expected OpenAI auto-configure without base_url/model to fail validation")
	}

	// Valid local OpenAI-compatible (Ollama-style) configuration succeeds without raw secret in body.
	oaiStatus, err := n.AutoConfigureNexIntelligence(context.Background(), AutoConfigureIntelligenceRequest{
		Mode:    coreconfig.IntelligenceOpenAICompatible,
		BaseURL: "http://127.0.0.1:11434/v1",
		Model:   "qwen2.5-coder:7b",
	})
	if err != nil {
		t.Fatalf("AutoConfigure OpenAI failed: %v", err)
	}
	if oaiStatus.Mode != coreconfig.IntelligenceOpenAICompatible || oaiStatus.BaseURL != "http://127.0.0.1:11434/v1" {
		t.Fatalf("unexpected OpenAI status: %+v", oaiStatus)
	}
}
