//go:build windows

package host

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestCodexSessionHostDefaultsToConPTYAndForwardsArrowInput(t *testing.T) {
	t.Setenv("AI_CLI_ENABLE_CONPTY", "")
	t.Setenv("AI_CLI_DISABLE_CONPTY", "")

	powershell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Skip("Windows PowerShell is unavailable")
	}

	runtimeID := fmt.Sprintf("rt-codex-conpty-%d", time.Now().UnixNano())
	codexHome := t.TempDir()
	keyResultPath := filepath.Join(t.TempDir(), "conpty-key.txt")
	workspace := t.TempDir()
	env := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		key, _, _ := strings.Cut(entry, "=")
		if strings.EqualFold(key, "CODEX_HOME") || strings.EqualFold(key, "AI_CLI_ENABLE_CONPTY") || strings.EqualFold(key, "AI_CLI_DISABLE_CONPTY") || strings.EqualFold(key, "NEXUS_TEST_KEY_FILE") {
			continue
		}
		env = append(env, entry)
	}
	env = append(env, "CODEX_HOME="+codexHome)
	env = append(env, "NEXUS_TEST_KEY_FILE="+keyResultPath)

	registryPath := filepath.Join(t.TempDir(), "runtimes.json")
	ownedRegistry := registry.NewRegistry(registryPath)
	session := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "codex",
		ProfileID:    "windows-conpty-test",
		Workspace:    workspace,
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}
	sh, err := NewSessionHost(Config{
		Session:  session,
		Registry: ownedRegistry,
		Binary:   powershell,
		Args: []string{
			"-NoLogo",
			"-NoProfile",
			"-Command",
			"[Console]::WriteLine('NEXUS_STDIN_REDIRECTED=' + [Console]::IsInputRedirected); $key = [Console]::ReadKey($true); [System.IO.File]::WriteAllText($env:NEXUS_TEST_KEY_FILE, [string]$key.Key); [Console]::WriteLine('NEXUS_KEY=' + $key.Key); $null = [Console]::ReadKey($true)",
		},
		Env: env,
		Cwd: workspace,
	})
	if err != nil {
		t.Fatalf("create SessionHost: %v", err)
	}
	if err := sh.Start(); err != nil {
		t.Fatalf("start Codex-policy SessionHost: %v", err)
	}
	defer sh.Stop()
	waitForHostEndpoint(t, runtimeID)

	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("connect to SessionHost: %v", err)
	}
	defer client.Close()
	attach, err := client.Send(protocol.CmdAttach, nil)
	if err != nil {
		t.Fatalf("attach to SessionHost: %v", err)
	}
	output := waitForWindowsTerminalMarker(t, client, attach.Data, "NEXUS_STDIN_REDIRECTED=False")

	if _, err := client.RawConn().Write([]byte("\x1b[B")); err != nil {
		t.Fatalf("send Down Arrow over Nexus attach: %v", err)
	}
	output = waitForWindowsTerminalMarker(t, client, []byte(output), "NEXUS_KEY=")
	if !strings.Contains(output, "NEXUS_KEY=DownArrow") {
		key, _ := os.ReadFile(keyResultPath)
		t.Fatalf("Nexus attach did not deliver Down Arrow to the Codex TUI console: key=%q output=%q", key, output)
	}
}

func waitForWindowsTerminalMarker(t *testing.T, client *protocol.Client, initial []byte, marker string) string {
	t.Helper()
	var output strings.Builder
	output.Write(initial)
	if strings.Contains(output.String(), marker) {
		return output.String()
	}
	if err := client.RawConn().SetReadDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatalf("set terminal read deadline: %v", err)
	}
	defer client.ClearDeadline()
	for !strings.Contains(output.String(), marker) {
		chunk, err := client.Reader().ReadString('\n')
		if err != nil {
			t.Fatalf("terminal marker %q not observed: %v; output=%q", marker, err, output.String())
		}
		output.WriteString(chunk)
	}
	return output.String()
}
