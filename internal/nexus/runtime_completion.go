package nexus

import (
	"fmt"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func runtimeCompletionOutcome(state registry.RuntimeState, streamClosed bool) error {
	if !streamClosed {
		return fmt.Errorf("runtime stream did not close")
	}
	if state != registry.StateStopped {
		return fmt.Errorf("runtime ended in non-terminal state %q", state)
	}
	return nil
}
