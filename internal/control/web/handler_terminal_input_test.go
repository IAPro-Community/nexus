package web

import (
	"bytes"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/nexus"
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

func TestWebInteractiveDelegationModeFailsClosed(t *testing.T) {
	t.Setenv("NEXUS_DELEGATION_MODE", "")
	if got := webInteractiveDelegationMode(registry.RuntimeSession{}); got != nexus.DelegationOff {
		t.Fatalf("missing web delegation mode=%q want OFF", got)
	}
	if got := webInteractiveDelegationMode(registry.RuntimeSession{Labels: map[string]string{"nexus.delegation_mode": "garbage"}}); got != nexus.DelegationOff {
		t.Fatalf("invalid web delegation mode=%q want OFF", got)
	}
	if got := webInteractiveDelegationMode(registry.RuntimeSession{Labels: map[string]string{"nexus.delegation_mode": "AUTO"}}); got != nexus.DelegationAuto {
		t.Fatalf("explicit web AUTO=%q", got)
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

func TestWriteAttachedTerminalInputPreservesRawBytes(t *testing.T) {
	var out bytes.Buffer
	input := "echo __NEXUS_DOCS_TERMINAL_OK__\r\x1b[A"
	if err := writeAttachedTerminalInput(&out, input); err != nil {
		t.Fatal(err)
	}
	if got := out.String(); got != input {
		t.Fatalf("attached terminal input changed: got %q want %q", got, input)
	}
	if bytes.Contains(out.Bytes(), []byte(`"command":"input"`)) {
		t.Fatalf("browser PTY input must not be wrapped as CmdInput RPC: %q", out.String())
	}
}
