package desktop

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type nativeVoiceCaptureState struct {
	cmd       *exec.Cmd
	path      string
	startedAt time.Time
	stderr    *bytes.Buffer
}

type NativeVoiceRecording struct {
	DataBase64 string `json:"dataBase64"`
	MIMEType   string `json:"mimeType"`
	DurationMS int64  `json:"durationMs"`
}

func (a *App) StartVoiceCapture() error {
	if runtime.GOOS != "linux" {
		return fmt.Errorf("%w: native voice capture", ErrCapabilityUnavailable)
	}

	a.voiceMu.Lock()
	defer a.voiceMu.Unlock()
	if a.voiceCapture != nil {
		return errors.New("voice capture already active")
	}

	path, err := voiceCaptureTempPath()
	if err != nil {
		return err
	}
	cmd, err := voiceCaptureCommand(path)
	if err != nil {
		_ = os.Remove(path)
		return err
	}
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr
	if err := cmd.Start(); err != nil {
		_ = os.Remove(path)
		return fmt.Errorf("start native voice capture: %w", err)
	}
	a.voiceCapture = &nativeVoiceCaptureState{
		cmd:       cmd,
		path:      path,
		startedAt: time.Now(),
		stderr:    stderr,
	}
	return nil
}

func (a *App) StopVoiceCapture() (NativeVoiceRecording, error) {
	state, err := a.takeVoiceCapture(false)
	if err != nil {
		return NativeVoiceRecording{}, err
	}
	if err := stopVoiceCaptureProcess(state, false); err != nil {
		_ = os.Remove(state.path)
		return NativeVoiceRecording{}, err
	}
	defer os.Remove(state.path)

	data, err := os.ReadFile(state.path)
	if err != nil {
		return NativeVoiceRecording{}, fmt.Errorf("read native voice capture: %w", err)
	}
	if len(data) <= 44 {
		return NativeVoiceRecording{}, errors.New("native voice capture returned no audio")
	}
	return NativeVoiceRecording{
		DataBase64: base64.StdEncoding.EncodeToString(data),
		MIMEType:   "audio/wav",
		DurationMS: max(time.Since(state.startedAt).Milliseconds(), 1),
	}, nil
}

func (a *App) CancelVoiceCapture() {
	a.cancelNativeVoiceCapture()
}

func (a *App) cancelNativeVoiceCapture() {
	state, err := a.takeVoiceCapture(true)
	if err != nil {
		return
	}
	_ = stopVoiceCaptureProcess(state, true)
	_ = os.Remove(state.path)
}

func (a *App) takeVoiceCapture(cancel bool) (*nativeVoiceCaptureState, error) {
	a.voiceMu.Lock()
	defer a.voiceMu.Unlock()
	if a.voiceCapture == nil {
		if cancel {
			return nil, errors.New("no active voice capture")
		}
		return nil, errors.New("voice capture is not active")
	}
	state := a.voiceCapture
	a.voiceCapture = nil
	return state, nil
}

func voiceCaptureTempPath() (string, error) {
	file, err := os.CreateTemp("", "nexus-voice-*.wav")
	if err != nil {
		return "", fmt.Errorf("create native voice capture file: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return path, nil
}

func voiceCaptureCommand(path string) (*exec.Cmd, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("%w: native voice capture", ErrCapabilityUnavailable)
	}
	if binary, err := exec.LookPath("pw-record"); err == nil {
		return exec.Command(
			binary,
			"--rate=16000",
			"--channels=1",
			"--channel-map=mono",
			"--format=s16",
			path,
		), nil
	}
	if binary, err := exec.LookPath("arecord"); err == nil {
		return exec.Command(
			binary,
			"-q",
			"-f", "S16_LE",
			"-r", "16000",
			"-c", "1",
			"-t", "wav",
			path,
		), nil
	}
	return nil, fmt.Errorf("%w: install pw-record (PipeWire) or arecord (ALSA)", ErrCapabilityUnavailable)
}

func stopVoiceCaptureProcess(state *nativeVoiceCaptureState, cancel bool) error {
	if state == nil || state.cmd == nil || state.cmd.Process == nil {
		return errors.New("invalid native voice capture state")
	}
	_ = state.cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() {
		done <- state.cmd.Wait()
	}()

	var waitErr error
	select {
	case waitErr = <-done:
	case <-time.After(2 * time.Second):
		_ = state.cmd.Process.Kill()
		waitErr = <-done
	}
	if cancel {
		return nil
	}
	if waitErr != nil {
		message := strings.TrimSpace(state.stderr.String())
		if message != "" {
			return fmt.Errorf("native voice capture failed: %s", message)
		}
		// Recording tools commonly report an interrupt status when the user
		// intentionally stops capture. A finalized WAV is authoritative.
		if info, statErr := os.Stat(state.path); statErr != nil || info.Size() <= 44 {
			return fmt.Errorf("native voice capture failed: %w", waitErr)
		}
	}
	return nil
}
