package promotion

import "testing"

func TestLineShadowHoldsEnterAndTracksEditing(t *testing.T) {
	shadow := NewLineShadow()
	event := shadow.Consume([]byte("implemente backend\b e frontend\r"))
	if got, want := string(event.Forward), "implemente backend\b e frontend"; got != want {
		t.Fatalf("forward mismatch: got %q want %q", got, want)
	}
	if !event.Submit || event.Line != "implemente backen e frontend" {
		t.Fatalf("unexpected submit event: %+v", event)
	}
}

func TestLineShadowDoesNotLeakControlSequencesOrPasteIntoShadow(t *testing.T) {
	shadow := NewLineShadow()
	input := []byte("go \x1b[2K\x1b[200~backend\nfrontend\x1b[201~ api\r")
	event := shadow.Consume(input)
	if string(event.Forward) != string(input[:len(input)-1]) {
		t.Fatalf("control/paste bytes were not forwarded byte-for-byte: got %q", event.Forward)
	}
	if event.Line != "go  api" {
		t.Fatalf("control sequence or bracketed paste leaked into shadow: %q", event.Line)
	}
}

func TestLineShadowKeepsOSCCPRAndJSONBytesOutOfPrompt(t *testing.T) {
	shadow := NewLineShadow()
	input := []byte("fix \x1b]0;title\x07\x1b[6n{\"command\":\"input\"}\r")
	event := shadow.Consume(input)
	if string(event.Forward) != string(input[:len(input)-1]) {
		t.Fatalf("non-enter bytes changed: got %q", event.Forward)
	}
	if event.Line != "fix {\"command\":\"input\"}" {
		t.Fatalf("OSC/CPR leaked into prompt shadow: %q", event.Line)
	}
}

func TestLineShadowOffIsPassthrough(t *testing.T) {
	shadow := NewLineShadow()
	shadow.SetEnabled(false)
	event := shadow.Consume([]byte("hello\r"))
	if string(event.Forward) != "hello\r" || event.Submit {
		t.Fatalf("disabled shadow must passthrough: %+v", event)
	}
}

func TestLineShadowEscapeAndCtrlReset(t *testing.T) {
	shadow := NewLineShadow()
	shadow.Consume([]byte("hello"))
	if got := shadow.Consume([]byte{0x1b}); string(got.Forward) != "\x1b" || shadow.Line() != "hello" {
		t.Fatalf("plain escape should forward without mutating line: %+v line=%q", got, shadow.Line())
	}
	shadow.Consume([]byte{0x03})
	if shadow.Line() != "" {
		t.Fatalf("ctrl key must reset shadow, got %q", shadow.Line())
	}
}

func TestLineShadowBackspaceRemovesUTF8Rune(t *testing.T) {
	shadow := NewLineShadow()
	shadow.Consume([]byte("olá"))
	shadow.Consume([]byte{0x7f})
	if got := shadow.Line(); got != "ol" {
		t.Fatalf("backspace should remove complete UTF-8 rune, got %q", got)
	}
}
