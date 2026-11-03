package web

import "testing"

func TestSendTerminalReplayMarksEndAfterOutput(t *testing.T) {
	var frames []TerminalMessage
	send := func(value any) error {
		frame := value.(TerminalMessage)
		frames = append(frames, frame)
		return nil
	}
	history := "screen\x1b[H\n" +
		"{\"version\":1,\"id\":\"req-1\",\"command\":\"lease_acquire\",\"timestamp\":\"2026-01-01T00:00:00Z\"}\n" +
		"visible\n"
	if err := sendTerminalReplay(send, history); err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 || frames[0].Type != "output" || frames[1].Type != "replay_complete" {
		t.Fatalf("unexpected replay order: %+v", frames)
	}
	if frames[0].Data != "screen\x1b[H\nvisible\n" {
		t.Fatalf("invalid filtered replay: %q", frames[0].Data)
	}
}

func TestSendTerminalReplayMarksEmptyHistory(t *testing.T) {
	var frames []TerminalMessage
	if err := sendTerminalReplay(func(value any) error {
		frames = append(frames, value.(TerminalMessage))
		return nil
	}, ""); err != nil {
		t.Fatal(err)
	}
	if len(frames) != 1 || frames[0].Type != "replay_complete" {
		t.Fatalf("empty replay must still send completion: %+v", frames)
	}
}
