package host

import (
	"testing"
)

// Characterization tests: lock down existing slash prefix router behavior
// before extending with colon prefix support.

func feedString(r *SlashPrefixRouter, s string) []RouterOutput {
	var outputs []RouterOutput
	for i := 0; i < len(s); i++ {
		outputs = append(outputs, r.ProcessByte(s[i]))
	}
	return outputs
}

func collectControlCmds(outputs []RouterOutput) []string {
	var cmds []string
	for _, o := range outputs {
		if o.Action == ActionControlCommand {
			cmds = append(cmds, o.ControlCmd)
		}
	}
	return cmds
}

func collectForwarded(outputs []RouterOutput) []byte {
	var out []byte
	for _, o := range outputs {
		if o.Action == ActionForwardBytes {
			out = append(out, o.ForwardBytes...)
		}
	}
	return out
}

// --- Existing /nexus behavior ---

func TestSlashNexus_Status(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, "/nexus status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != "/nexus status" {
		t.Errorf("expected ['/nexus status'], got %v", cmds)
	}
}

func TestSlashNexus_Accounts(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, "/nexus accounts\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != "/nexus accounts" {
		t.Errorf("expected ['/nexus accounts'], got %v", cmds)
	}
}

func TestSlashAI_IsForwardedAsLegacyText(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, "/ai status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 0 || string(collectForwarded(outputs)) != "/ai status\r" {
		t.Errorf("expected legacy /ai text to pass through, got commands=%v forwarded=%q", cmds, collectForwarded(outputs))
	}
}

func TestDoubleSlashNexus_EscapesToChild(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, "//nexus status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 0 {
		t.Errorf("expected no control commands for //nexus, got %v", cmds)
	}
	forwarded := collectForwarded(outputs)
	expected := "/nexus status\r"
	if string(forwarded) != expected {
		t.Errorf("expected forwarded %q, got %q", expected, string(forwarded))
	}
}

func TestDoubleSlashAI_IsForwardedLiterally(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, "//ai status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 0 {
		t.Errorf("expected no control commands for //ai, got %v", cmds)
	}
	forwarded := collectForwarded(outputs)
	expected := "//ai status\r"
	if string(forwarded) != expected {
		t.Errorf("expected forwarded %q, got %q", expected, string(forwarded))
	}
}

func TestOtherSlash_ForwardedToChild(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, "/model gpt-4\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 0 {
		t.Errorf("expected no control commands for /model, got %v", cmds)
	}
	forwarded := collectForwarded(outputs)
	expected := "/model gpt-4\r"
	if string(forwarded) != expected {
		t.Errorf("expected forwarded %q, got %q", expected, string(forwarded))
	}
}

func TestNormalText_ForwardedToChild(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, "hello world\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 0 {
		t.Errorf("expected no control commands for normal text, got %v", cmds)
	}
	forwarded := collectForwarded(outputs)
	expected := "hello world\r"
	if string(forwarded) != expected {
		t.Errorf("expected forwarded %q, got %q", expected, string(forwarded))
	}
}

func TestCtrlC_ResetsRouter(t *testing.T) {
	r := NewSlashPrefixRouter()
	// Start typing /nexus
	feedString(r, "/nex")
	// Ctrl+C should reset
	output := r.ProcessByte(0x03)
	if output.Action != ActionForwardBytes {
		t.Errorf("expected forward for Ctrl+C, got action %v", output.Action)
	}
	// Now typing /nexus again should work fresh
	outputs := feedString(r, "/nexus status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 {
		t.Errorf("expected 1 control command after Ctrl+C reset, got %v", cmds)
	}
}

func TestCtrlU_ResetsRouter(t *testing.T) {
	r := NewSlashPrefixRouter()
	feedString(r, "/nex")
	output := r.ProcessByte(0x15)
	if output.Action != ActionForwardBytes {
		t.Errorf("expected forward for Ctrl+U, got action %v", output.Action)
	}
	outputs := feedString(r, "/nexus status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 {
		t.Errorf("expected 1 control command after Ctrl+U reset, got %v", cmds)
	}
}

func TestBackspace_InControlCommand(t *testing.T) {
	r := NewSlashPrefixRouter()
	feedString(r, "/nexus statu")
	// Backspace removes 'u', leaving "/nexus stat"
	output := r.ProcessByte(0x7f)
	if output.Action != ActionNone {
		t.Errorf("expected none for backspace, got action %v", output.Action)
	}
	// Now type 's' and Enter → "/nexus stats"
	outputs := feedString(r, "s\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != "/nexus stats" {
		t.Errorf("expected ['/nexus stats'] after backspace, got %v", cmds)
	}
}

func TestTab_InIdle_Forwarded(t *testing.T) {
	r := NewSlashPrefixRouter()
	output := r.ProcessByte('\t')
	if output.Action != ActionForwardBytes {
		t.Errorf("expected forward for tab in idle, got action %v", output.Action)
	}
}

func TestTab_InSlashNexus_ShowsSuggestions(t *testing.T) {
	r := NewSlashPrefixRouter()
	feedString(r, "/nexus")
	output := r.ProcessByte('\t')
	if output.Action != ActionSuggestions {
		t.Errorf("expected suggestions for tab after /nexus, got action %v", output.Action)
	}
	if output.Suggestions == "" {
		t.Error("expected non-empty suggestions")
	}
}

func TestUTF8_InPassthrough(t *testing.T) {
	r := NewSlashPrefixRouter()
	// Type a regular character then UTF-8
	outputs := feedString(r, "aé\r")
	forwarded := collectForwarded(outputs)
	if string(forwarded) != "aé\r" {
		t.Errorf("expected 'aé\\r' forwarded, got %q", string(forwarded))
	}
}

func TestEmptyInput_NoOutput(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, "")
	if len(outputs) != 0 {
		t.Errorf("expected no outputs for empty input, got %d", len(outputs))
	}
}

func TestReset_ClearsState(t *testing.T) {
	r := NewSlashPrefixRouter()
	feedString(r, "/nex")
	r.Reset()
	// After reset, typing "nexus" should be forwarded normally
	outputs := feedString(r, "nexus\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 0 {
		t.Errorf("expected no commands after reset, got %v", cmds)
	}
}

func TestLF_AsWellAsCR(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, "/nexus status\n")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != "/nexus status" {
		t.Errorf("expected ['/nexus status'] with LF, got %v", cmds)
	}
}

func TestTab_ShowsSuggestionsInSubcommand(t *testing.T) {
	r := NewSlashPrefixRouter()
	feedString(r, "/nexus stat")
	output := r.ProcessByte('\t')
	if output.Action != ActionSuggestions {
		t.Errorf("expected suggestions, got action %v", output.Action)
	}
}

func TestMultipleCommands_ResetsBetween(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs1 := feedString(r, "/nexus status\r")
	cmds1 := collectControlCmds(outputs1)
	if len(cmds1) != 1 {
		t.Errorf("expected 1 command, got %d", len(cmds1))
	}
	// Second command should work independently
	outputs2 := feedString(r, "/nexus accounts\r")
	cmds2 := collectControlCmds(outputs2)
	if len(cmds2) != 1 || cmds2[0] != "/nexus accounts" {
		t.Errorf("expected ['/nexus accounts'], got %v", cmds2)
	}
}
