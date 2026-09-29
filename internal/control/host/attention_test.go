package host

import (
	"sync"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestAttentionDetectorQuestions(t *testing.T) {
	var mu sync.Mutex
	var gotReason, gotContext, gotTitle string
	var gotState registry.RuntimeState

	detector := NewAttentionDetector("rt-123", "agy", "default", "/workspace/my-project", func(reason, context, dynamicTitle string, state registry.RuntimeState) {
		mu.Lock()
		defer mu.Unlock()
		gotReason = reason
		gotContext = context
		gotTitle = dynamicTitle
		gotState = state
	})

	promptChunk := []byte("\r\n\x1b[33mPor favor, revise o plano. Se estiver de acordo, aprove para iniciarmos a implementação imediata.\x1b[0m\r\n? for shortcuts\r\n")
	detector.ProcessChunk(promptChunk)

	mu.Lock()
	defer mu.Unlock()

	if gotReason != "QUESTION" {
		t.Fatalf("expected reason QUESTION, got %q", gotReason)
	}
	if gotContext == "" || !contains(gotContext, "revise o plano") {
		t.Fatalf("expected context with question phrase, got %q", gotContext)
	}
	if detector.lastPrompt != PromptKindFreeText {
		t.Fatalf("expected prompt_kind free_text, got %q", detector.lastPrompt)
	}
	if gotState != registry.StateWaiting {
		t.Fatalf("expected state WAITING, got %v", gotState)
	}
	if gotTitle == "" || !contains(gotTitle, "my-project") {
		t.Fatalf("expected dynamic title with project, got %q", gotTitle)
	}
}

func TestAttentionDetectorIgnoresShortcutHint(t *testing.T) {
	var gotReason string
	detector := NewAttentionDetector("rt-shortcuts", "codex", "default", "/workspace/project", func(reason, _, _ string, _ registry.RuntimeState) {
		gotReason = reason
	})

	detector.ProcessChunk([]byte("\r\n? for shortcuts\r\n"))
	if gotReason == "QUESTION" {
		t.Fatal("terminal shortcut hint must not be treated as an agent question")
	}
}

func TestAttentionDetectorIgnoresIsolatedYN(t *testing.T) {
	var gotReason string
	detector := NewAttentionDetector("rt-yn-alone", "codex", "default", "/workspace/project", func(reason, _, _ string, _ registry.RuntimeState) {
		gotReason = reason
	})
	detector.ProcessChunk([]byte("\r\n[y/n]\r\n"))
	if gotReason == "QUESTION" {
		t.Fatal("isolated [y/n] must not be treated as a question")
	}
}

func TestAttentionDetectorYNWithQuestion(t *testing.T) {
	var gotReason, gotContext string
	detector := NewAttentionDetector("rt-yn-q", "codex", "default", "/workspace/project", func(reason, context, _ string, _ registry.RuntimeState) {
		gotReason = reason
		gotContext = context
	})
	detector.ProcessChunk([]byte("\r\nDeseja executar a migração? [y/N]\r\n"))
	if gotReason != "QUESTION" {
		t.Fatalf("expected QUESTION, got %q", gotReason)
	}
	if detector.lastPrompt != PromptKindYN {
		t.Fatalf("expected yn, got %q", detector.lastPrompt)
	}
	if !contains(gotContext, "Deseja executar") {
		t.Fatalf("context must include the question, got %q", gotContext)
	}
}

func TestAttentionDetectorClearsOnWorkingOutput(t *testing.T) {
	detector := NewAttentionDetector("rt-clear", "codex", "default", "/workspace/project", nil)
	detector.ProcessChunk([]byte("\r\nDeseja executar a migração? [y/N]\r\n"))
	if detector.lastReason != "QUESTION" {
		t.Fatalf("setup failed: %q", detector.lastReason)
	}
	detector.ProcessChunk([]byte("\r\nthinking...\r\n"))
	if detector.lastReason == "QUESTION" {
		t.Fatal("working output must clear QUESTION state")
	}
	if detector.lastKind != AttentionWorking {
		t.Fatalf("expected working kind, got %q", detector.lastKind)
	}
}

func TestAttentionDetectorTaskCompleted(t *testing.T) {
	var mu sync.Mutex
	var gotReason, gotContext, gotTitle string

	detector := NewAttentionDetector("rt-456", "codex", "default", "/workspace/backend-api", func(reason, context, dynamicTitle string, state registry.RuntimeState) {
		mu.Lock()
		defer mu.Unlock()
		gotReason = reason
		gotContext = context
		gotTitle = dynamicTitle
	})

	chunk := []byte("\r\n\x1b[32mTask completed: Authentication endpoints created and all 15 tests pass.\x1b[0m\r\n")
	detector.ProcessChunk(chunk)

	mu.Lock()
	defer mu.Unlock()

	if gotReason != "TASK_COMPLETED" {
		t.Fatalf("expected reason TASK_COMPLETED, got %q", gotReason)
	}
	if gotContext == "" {
		t.Fatal("expected non-empty context")
	}
	if gotTitle == "" || !contains(gotTitle, "backend-api") {
		t.Fatalf("expected dynamic title with project, got %q", gotTitle)
	}
}

func TestAttentionDetectorRecognizesAnchoredFatalFailures(t *testing.T) {
	cases := []string{
		"fatal error: provider crashed",
		"uncaught exception: boom",
		"panic: runtime failure",
		"failure: verification failed",
		"process exited with code 2",
	}
	for _, line := range cases {
		t.Run(line, func(t *testing.T) {
			detector := NewAttentionDetector("rt-fatal", "codex", "default", "/workspace/demo", nil)
			detector.ProcessChunk([]byte("\r\n" + line + "\r\n"))
			if detector.lastReason != "ERROR" || detector.lastKind != AttentionError {
				t.Fatalf("expected live fatal marker to classify as ERROR, got reason=%q kind=%q", detector.lastReason, detector.lastKind)
			}
		})
	}

	detector := NewAttentionDetector("rt-fatal-noise", "codex", "default", "/workspace/demo", nil)
	detector.ProcessChunk([]byte("\r\nThe log said `panic: runtime failure` yesterday\r\n"))
	if detector.lastReason == "ERROR" || detector.lastKind == AttentionError {
		t.Fatal("inline quoted fatal marker must remain narration, not a live error")
	}
}

func TestAttentionDetectorOSNotifyOnlyOnEvidence(t *testing.T) {
	rec := &notify.Recorder{}
	notify.SetDefault(rec)
	t.Cleanup(func() { notify.SetDefault(nil) })

	detector := NewAttentionDetector("rt-notify", "codex", "default", "/workspace/demo", nil)
	detector.ProcessChunk([]byte("\r\n? for shortcuts\r\n"))
	if len(rec.Payloads) != 0 {
		t.Fatalf("shortcut chrome must not OS-notify, got %#v", rec.Payloads)
	}

	detector.ProcessChunk([]byte("\r\nDeseja executar a migração? [y/N]\r\n"))
	if len(rec.Payloads) != 0 {
		t.Fatalf("needs_user must not OS-notify (bus/UI only), got %d payloads", len(rec.Payloads))
	}
	if detector.lastKind != AttentionNeedsUser {
		t.Fatalf("expected needs_user classification, got %q", detector.lastKind)
	}

	detector.ProcessChunk([]byte("\r\nerror: build failed hard\r\n"))
	if len(rec.Payloads) != 1 {
		t.Fatalf("expected one OS notify for agent error, got %d", len(rec.Payloads))
	}

	detector.ProcessChunk([]byte("\r\nerror: build failed hard\r\n"))
	if len(rec.Payloads) != 1 {
		t.Fatalf("same fingerprint must not re-notify, got %d", len(rec.Payloads))
	}
}

func TestAttentionDetectorIgnoresCitedCreateProcessWFailure(t *testing.T) {
	rec := &notify.Recorder{}
	notify.SetDefault(rec)
	t.Cleanup(func() { notify.SetDefault(nil) })

	detector := NewAttentionDetector("rt-fc01-cite", "codex", "default", "/workspace/ai-manager", nil)
	// Exact toast payload shape: agent narration quoting a CI/closure checklist,
	// not a live ConPTY failure on this host.
	chunk := []byte("\r\n15 +| FC-01 | Windows SessionHost / ConPTY startup | STILL_PRESENT | CI run 34646679957 Windows diagnostics includes `CreateProcessW failed: The parameter is incorrect`\r\n")
	detector.ProcessChunk(chunk)
	if detector.lastReason == "ERROR" || detector.lastKind == AttentionError {
		t.Fatalf("cited CreateProcessW failure must not be live ERROR, reason=%q kind=%q", detector.lastReason, detector.lastKind)
	}
	if len(rec.Payloads) != 0 {
		t.Fatalf("cited CI diagnostics must not OS-notify, got %#v", rec.Payloads)
	}

	detector.ProcessChunk([]byte("\r\nthinking...\r\n"))
	detector.ProcessChunk([]byte("\r\nWindows diagnostics includes `CreateProcessW failed: The parameter is incorrect`\r\n"))
	if len(rec.Payloads) != 0 {
		t.Fatalf("inline backtick citation must not OS-notify after fingerprint reset, got %#v", rec.Payloads)
	}
}

func TestAttentionDetectorShellIgnoresInteractivePrompts(t *testing.T) {
	rec := &notify.Recorder{}
	notify.SetDefault(rec)
	t.Cleanup(func() { notify.SetDefault(nil) })

	var gotReason string
	detector := NewAttentionDetector("rt-shell", "shell", "default", "/workspace/project", func(reason, _, _ string, _ registry.RuntimeState) {
		gotReason = reason
	})
	detector.ProcessChunk([]byte("\r\nDo you want to continue? [Y/n]\r\n"))
	if gotReason == "QUESTION" || detector.lastKind == AttentionNeedsUser {
		t.Fatal("project shell must not classify apt/npm-style prompts as needs_user")
	}
	if len(rec.Payloads) != 0 {
		t.Fatalf("shell must not OS-notify, got %#v", rec.Payloads)
	}
	detector.ProcessChunk([]byte("\r\nerror: something broke\r\n"))
	if len(rec.Payloads) != 0 {
		t.Fatalf("shell must not OS-notify on error either, got %#v", rec.Payloads)
	}
}

func TestAttentionDetectorEventsControlLevelDisablesPTYHeuristic(t *testing.T) {
	var gotReason string
	detector := NewAttentionDetector("rt-events", "codex", "default", "/workspace/project", func(reason, _, _ string, _ registry.RuntimeState) {
		gotReason = reason
	})
	detector.SetControlPolicy(registry.ControlLevelEvents, false, "agent-1")
	detector.ProcessChunk([]byte("\r\nDeseja executar a migração? [y/N]\r\n"))
	if gotReason == "QUESTION" || detector.lastKind == AttentionNeedsUser {
		t.Fatal("EVENTS control level must not scrape PTY for needs_user")
	}
}

func TestAttentionDetectorStructuredEventsDisablesPTYHeuristic(t *testing.T) {
	var gotReason string
	detector := NewAttentionDetector("rt-struct", "codex", "default", "/workspace/project", func(reason, _, _ string, _ registry.RuntimeState) {
		gotReason = reason
	})
	detector.SetControlPolicy(registry.ControlLevelTerminal, true, "agent-1")
	detector.ProcessChunk([]byte("\r\nDeseja executar a migração? [y/N]\r\n"))
	if gotReason == "QUESTION" || detector.lastKind == AttentionNeedsUser {
		t.Fatal("structured events adapter must own attention; PTY heuristic must stay off")
	}
}

func TestPtyHeuristicAllowed(t *testing.T) {
	if !ptyHeuristicAllowed("codex", registry.ControlLevelTerminal, false) {
		t.Fatal("codex TERMINAL without structured events should allow PTY fallback")
	}
	if ptyHeuristicAllowed("shell", registry.ControlLevelTerminal, false) {
		t.Fatal("shell must never allow PTY agent heuristic")
	}
	if ptyHeuristicAllowed("codex", registry.ControlLevelAPI, false) {
		t.Fatal("CONTROL_API must not scrape PTY")
	}
	if ptyHeuristicAllowed("codex", registry.ControlLevelTerminal, true) {
		t.Fatal("structured events must disable PTY heuristic")
	}
}

func TestAttentionDetectorIgnoresNumberedListAlone(t *testing.T) {
	var gotReason string
	detector := NewAttentionDetector("rt-list", "codex", "default", "/workspace/project", func(reason, _, _ string, _ registry.RuntimeState) {
		gotReason = reason
	})
	detector.ProcessChunk([]byte("\r\n1. Implement auth middleware\r\n2. Add tests\r\n"))
	if gotReason == "QUESTION" {
		t.Fatal("numbered narration must not be treated as a choice prompt")
	}
}

func TestAttentionDetectorIgnoresReplacementGarbage(t *testing.T) {
	var gotReason string
	detector := NewAttentionDetector("rt-garbage", "codex", "default", "/workspace/project", func(reason, _, _ string, _ registry.RuntimeState) {
		gotReason = reason
	})
	detector.ProcessChunk([]byte("\r\n\uFFFD\uFFFD\uFFFD\uFFFD what next?\r\n"))
	// After sanitization-like filtering, diamond-heavy lines are garbage.
	// Even with a trailing question, majority garbage must not notify.
	detector.ProcessChunk([]byte("\r\n\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD\uFFFD\r\n"))
	if gotReason == "QUESTION" {
		t.Fatal("UTF-8 replacement garbage must not be treated as needs_user")
	}
}

func TestFormatDesktopAttentionBodyFramesYN(t *testing.T) {
	got := formatDesktopAttentionBody(PromptKindYN, "Continue anyway? [y/N]:")
	want := "Um agente pede confirmação (Sim/Não): Continue anyway? [y/N]: — Abra o Nexus e responda no terminal."
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestExtractOSCTitleUsesNewestSequence(t *testing.T) {
	raw := "\x1b]0;first\x07work\x1b]2;Question 1 of 3\x1b\\"
	got := ExtractOSCTitle(raw)
	if got != "Question 1 of 3" {
		t.Fatalf("ExtractOSCTitle=%q, want newest OSC title", got)
	}
}

func TestAttentionDetectorShellAppliesOSCTitle(t *testing.T) {
	var gotTitle string
	detector := NewAttentionDetector("rt-shell-title", "shell", "default", "/workspace/project", func(_, _, dynamicTitle string, _ registry.RuntimeState) {
		gotTitle = dynamicTitle
	})
	detector.ProcessChunk([]byte("\x1b]0;agy · thinking\x07"))
	if gotTitle != "agy · thinking" {
		t.Fatalf("project shell must surface OSC settitle, got %q", gotTitle)
	}
}

func TestAttentionDetectorShellQuestionnaireIsNeedsUser(t *testing.T) {
	detector := NewAttentionDetector("rt-shell-q", "shell", "default", "/workspace/project", nil)
	detector.ProcessChunk([]byte("\r\nQuestion 1 of 2\r\nSelect all that apply\r\n"))
	if detector.lastReason != "QUESTION" || detector.lastKind != AttentionNeedsUser {
		t.Fatalf("AGY questionnaire in project shell must be needs_user, reason=%q kind=%q", detector.lastReason, detector.lastKind)
	}
	if detector.lastPrompt != PromptKindChoice {
		t.Fatalf("expected choice prompt for questionnaire, got %q", detector.lastPrompt)
	}
}

func TestAttentionDetectorNumberedListWithAskIsChoice(t *testing.T) {
	detector := NewAttentionDetector("rt-choice-ask", "codex", "default", "/workspace/project", nil)
	detector.ProcessChunk([]byte("\r\nSelect an option:\r\n1. Continue\r\n"))
	if detector.lastReason != "QUESTION" {
		t.Fatalf("expected QUESTION for numbered choice under an ask, got %q", detector.lastReason)
	}
	if detector.lastPrompt != PromptKindChoice {
		t.Fatalf("expected choice, got %q", detector.lastPrompt)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
