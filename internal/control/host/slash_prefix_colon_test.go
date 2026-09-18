package host

import (
	"testing"
)

// Tests for colon prefix (:) canonical control commands.

func TestColonNexus_Status(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, ":nexus status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != ":nexus status" {
		t.Errorf("expected [':nexus status'], got %v", cmds)
	}
}

func TestColonNexus_Accounts(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, ":nexus accounts\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != ":nexus accounts" {
		t.Errorf("expected [':nexus accounts'], got %v", cmds)
	}
}

func TestColonAI_IsForwardedAsLegacyText(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, ":ai status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 0 || string(collectForwarded(outputs)) != ":ai status\r" {
		t.Errorf("expected legacy :ai text to pass through, got commands=%v forwarded=%q", cmds, collectForwarded(outputs))
	}
}

func TestDoubleColonNexus_EscapesToChild(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, "::nexus status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 0 {
		t.Errorf("expected no control commands for ::nexus, got %v", cmds)
	}
	forwarded := collectForwarded(outputs)
	expected := ":nexus status\r"
	if string(forwarded) != expected {
		t.Errorf("expected forwarded %q, got %q", expected, string(forwarded))
	}
}

func TestDoubleColonAI_IsForwardedLiterally(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, "::ai status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 0 {
		t.Errorf("expected no control commands for ::ai, got %v", cmds)
	}
	forwarded := collectForwarded(outputs)
	expected := "::ai status\r"
	if string(forwarded) != expected {
		t.Errorf("expected forwarded %q, got %q", expected, string(forwarded))
	}
}

func TestColonStatus_ShortAlias(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, ":status\r")
	// ":status" starts with ":" but is not ":nexus", so it's not
	// intercepted. It gets forwarded to the child as-is.
	forwarded := collectForwarded(outputs)
	expected := ":status\r"
	if string(forwarded) != expected {
		t.Errorf("expected forwarded %q for :status, got %q", expected, string(forwarded))
	}
}

func TestColonNexusTab_ShowsSuggestions(t *testing.T) {
	r := NewSlashPrefixRouter()
	feedString(r, ":nexus")
	output := r.ProcessByte('\t')
	if output.Action != ActionSuggestions {
		t.Errorf("expected suggestions for tab after :nexus, got action %v", output.Action)
	}
	if output.Suggestions == "" {
		t.Error("expected non-empty suggestions")
	}
}

func TestColonNexus_DelimiterSpace(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, ":nexus status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != ":nexus status" {
		t.Errorf("expected [':nexus status'], got %v", cmds)
	}
}

func TestMixedSlashesAndColons(t *testing.T) {
	r := NewSlashPrefixRouter()
	// /nexus then :nexus should both work independently
	outputs1 := feedString(r, "/nexus status\r")
	cmds1 := collectControlCmds(outputs1)
	if len(cmds1) != 1 {
		t.Errorf("expected 1 command from /nexus, got %d", len(cmds1))
	}
	outputs2 := feedString(r, ":nexus accounts\r")
	cmds2 := collectControlCmds(outputs2)
	if len(cmds2) != 1 || cmds2[0] != ":nexus accounts" {
		t.Errorf("expected [':nexus accounts'], got %v", cmds2)
	}
}

func TestColonNotRecognized_Forwarded(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, ":quit\r")
	forwarded := collectForwarded(outputs)
	expected := ":quit\r"
	if string(forwarded) != expected {
		t.Errorf("expected forwarded %q, got %q", expected, string(forwarded))
	}
}

func TestColonPartialPrefix_Buffered(t *testing.T) {
	r := NewSlashPrefixRouter()
	// ":ne" is a prefix of ":nexus", should be buffered
	outputs := feedString(r, ":ne")
	for _, o := range outputs {
		if o.Action != ActionNone {
			t.Errorf("expected none for partial prefix, got action %v", o.Action)
		}
	}
	// Complete it
	outputs = feedString(r, "xus status\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != ":nexus status" {
		t.Errorf("expected [':nexus status'], got %v", cmds)
	}
}

func TestDoubleColon_NotRecognizedForwarded(t *testing.T) {
	r := NewSlashPrefixRouter()
	// "::status" is not a recognized escape prefix, should be forwarded
	outputs := feedString(r, "::status\r")
	forwarded := collectForwarded(outputs)
	// The first colon triggers buffering, second colon continues buffering,
	// then "status" diverges from known prefixes and gets forwarded
	if len(forwarded) == 0 {
		t.Error("expected some forwarded bytes")
	}
}

func TestColonNexus_LF(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, ":nexus status\n")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != ":nexus status" {
		t.Errorf("expected [':nexus status'] with LF, got %v", cmds)
	}
}

func TestColonNexus_Help(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, ":nexus help\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != ":nexus help" {
		t.Errorf("expected [':nexus help'], got %v", cmds)
	}
}

func TestColonNexus_Stop(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, ":nexus stop\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != ":nexus stop" {
		t.Errorf("expected [':nexus stop'], got %v", cmds)
	}
}

func TestColonNexus_Detach(t *testing.T) {
	r := NewSlashPrefixRouter()
	outputs := feedString(r, ":nexus detach\r")
	cmds := collectControlCmds(outputs)
	if len(cmds) != 1 || cmds[0] != ":nexus detach" {
		t.Errorf("expected [':nexus detach'], got %v", cmds)
	}
}
