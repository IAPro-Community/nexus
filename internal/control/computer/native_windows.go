//go:build windows

package computer

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"unicode/utf16"

	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/executil"
)

const (
	maxNativeHelperRequestBytes  = 1 << 20
	maxNativeHelperResponseBytes = 4 << 20
	maxNativeHelperStderrBytes   = 64 << 10
)

//go:embed windows_helper.ps1
var windowsComputerPowerShellScript string

type boundedCapture struct {
	buf       bytes.Buffer
	remaining int
	truncated bool
}

func (w *boundedCapture) Write(p []byte) (int, error) {
	original := len(p)
	if w.remaining <= 0 {
		if original > 0 {
			w.truncated = true
		}
		return original, nil
	}
	keep := original
	if keep > w.remaining {
		keep = w.remaining
		w.truncated = true
	}
	_, _ = w.buf.Write(p[:keep])
	w.remaining -= keep
	return original, nil
}

func nativeHelperSupported() bool {
	return windowsPowerShellPath() != ""
}

func windowsPowerShellPath() string {
	for _, candidate := range []string{"powershell.exe", "pwsh.exe"} {
		if path, err := exec.LookPath(candidate); err == nil {
			return path
		}
	}
	return ""
}

func runNativeHelper(in io.Reader, out io.Writer) error {
	if in == nil || out == nil {
		return fmt.Errorf("computer-use helper stdio is required")
	}
	powerShell := windowsPowerShellPath()
	if powerShell == "" {
		return fmt.Errorf("%w: PowerShell is unavailable", ErrUnavailable)
	}

	raw, err := io.ReadAll(io.LimitReader(in, maxNativeHelperRequestBytes+1))
	if err != nil {
		return fmt.Errorf("read computer-use helper request: %w", err)
	}
	if len(raw) > maxNativeHelperRequestBytes {
		return fmt.Errorf("computer-use helper request exceeds %d bytes", maxNativeHelperRequestBytes)
	}

	var request helperRequest
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return fmt.Errorf("decode computer-use helper request: %w", err)
	}
	if request.ProtocolVersion != helperProtocolVersion {
		return fmt.Errorf("computer-use helper protocol %d is unsupported", request.ProtocolVersion)
	}
	switch strings.TrimSpace(strings.ToLower(request.Operation)) {
	case "capabilities", "permissions", "list_apps", "list_windows", "snapshot", "act":
	default:
		return fmt.Errorf("unsupported computer-use helper operation %q", request.Operation)
	}

	encoded, err := encodePowerShellCommand(windowsComputerPowerShellScript)
	if err != nil {
		return err
	}
	cmd := exec.Command(
		powerShell,
		"-NoLogo",
		"-NoProfile",
		"-NonInteractive",
		"-EncodedCommand",
		encoded,
	)
	executil.HideConsoleWindow(cmd)
	cmd.Stdin = bytes.NewReader(raw)
	stdout := &boundedCapture{remaining: maxNativeHelperResponseBytes}
	stderr := &boundedCapture{remaining: maxNativeHelperStderrBytes}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		message := security.Redact(strings.TrimSpace(stderr.buf.String()))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("Windows computer-use helper failed: %s", message)
	}
	if stdout.truncated {
		return fmt.Errorf("Windows computer-use helper response exceeded %d bytes", maxNativeHelperResponseBytes)
	}

	var envelope helperEnvelope
	if err := json.Unmarshal(stdout.buf.Bytes(), &envelope); err != nil {
		return fmt.Errorf("decode Windows computer-use helper response: %w", err)
	}
	if envelope.Error != "" {
		envelope.Error = security.Redact(envelope.Error)
	}
	clean, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("encode Windows computer-use helper response: %w", err)
	}
	if _, err := out.Write(clean); err != nil {
		return fmt.Errorf("write Windows computer-use helper response: %w", err)
	}
	return nil
}

func encodePowerShellCommand(script string) (string, error) {
	if strings.TrimSpace(script) == "" {
		return "", fmt.Errorf("Windows computer-use helper script is empty")
	}
	codeUnits := utf16.Encode([]rune(script))
	raw := make([]byte, len(codeUnits)*2)
	for i, unit := range codeUnits {
		binary.LittleEndian.PutUint16(raw[i*2:], unit)
	}
	return base64.StdEncoding.EncodeToString(raw), nil
}
