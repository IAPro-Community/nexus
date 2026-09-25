package web

import (
	"bytes"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	nexusruntime "github.com/IAPro-Community/nexus/internal/runtime"
)

func TestWebInteractiveLeadRequiresExplicitLabel(t *testing.T) {
	if webInteractiveLeadEnabled(registry.RuntimeSession{}) {
		t.Fatal("missing label must not enable Nexus Lead interception")
	}
	if webInteractiveLeadEnabled(registry.RuntimeSession{Labels: map[string]string{"nexus.interactive_lead": "false"}}) {
		t.Fatal("false label must keep provider-native terminal behavior")
	}
	if !webInteractiveLeadEnabled(registry.RuntimeSession{Labels: map[string]string{"nexus.interactive_lead": "TRUE"}}) {
		t.Fatal("explicit true label must enable the Lead promotion boundary")
	}
}

func TestWebTerminalInputWriterCoalescesSplitSelectorKeys(t *testing.T) {
	var out bytes.Buffer
	stream := nexusruntime.NewTerminalInputScrubber(terminalInputCommandWriter(func(data []byte) (int, error) {
		return out.Write(data)
	}))

	for _, chunk := range [][]byte{
		{0x1b},
		[]byte("[A"),
		{0x1b},
		{'O'},
		{'B'},
	} {
		if _, err := stream.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	if err := stream.Flush(); err != nil {
		t.Fatal(err)
	}

	if got, want := out.String(), "\x1b[A\x1bOB"; got != want {
		t.Fatalf("selector input changed: got %q want %q", got, want)
	}
}
