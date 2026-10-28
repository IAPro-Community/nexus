package host

import (
    "context"
    "strings"
    "testing"
)

func inputReadyBackend(t *testing.T) *structuredBackend {
    t.Helper()
    b := newStructuredBackend("acp", t.TempDir(), nil, nil)
    b.started = true
    b.ctx = context.Background()
    return b
}

func TestStructuredInputDoesNotEchoSensitiveAnswers(t *testing.T) {
    b := inputReadyBackend(t)
    sensitive := "/answer ui-17 secret-value-that-must-not-enter-logs\r\n"
    if _, err := b.Write([]byte(sensitive)); err != nil { t.Fatal(err) }
    select {
    case got := <-b.inputLines:
        if got != strings.TrimSpace(sensitive) { t.Fatalf("wrong command queued: %q", got) }
    default:
        t.Fatal("command not queued")
    }
    select {
    case emitted := <-b.lines:
        if strings.Contains(emitted, "secret-value") || strings.Contains(emitted, "ui-17") {
            t.Fatalf("private terminal input was echoed: %q", emitted)
        }
    default:
        t.Fatal("no acknowledgement emitted")
    }
}

func TestStructuredInputSanitizesCursorAndBackspace(t *testing.T) {
    b := inputReadyBackend(t)
    // Simulate xterm arrow-up, a backspace and a UTF-8 correction.
    if _, err := b.Write([]byte("\x1b[Aabc\x7fõ\x7fX\r\n")); err != nil { t.Fatal(err) }
    select {
    case got := <-b.inputLines:
        if got != "abX" { t.Fatalf("control sequence leaked into prompt: %q", got) }
    default:
        t.Fatal("missing queued prompt")
    }
}

func TestStructuredInputRejectsOverLimitWithoutSubmittingTruncatedPrompt(t *testing.T) {
    b := inputReadyBackend(t)
    if _, err := b.Write([]byte(strings.Repeat("a", 16385)+"\r\n")); err != nil {
        t.Fatal(err)
    }
    select {
    case got := <-b.inputLines:
        t.Fatalf("truncated prompt incorrectly submitted: %q", got)
    default:
    }
}
func TestStructuredInputBoundedQueueFailsRatherThanSilentlyDropsCommands(t *testing.T) {
    b := inputReadyBackend(t)
    for i:=0;i<cap(b.inputLines);i++ {
        if _,err:=b.Write([]byte("hello\r"));err!=nil {t.Fatal(err)}
    }
    if _,err:=b.Write([]byte("overflow\r"));err==nil {
        t.Fatal("queue saturation silently discarded input")
    }
}
