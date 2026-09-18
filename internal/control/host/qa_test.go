package host

import (
	"bytes"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

func TestQA_RapidAttachDetachSpam(t *testing.T) {
	runtimeID := "rt-qa-spam"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  testTerminalBinary,
		Args:    testTerminalArgs(),
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()

	waitForHostEndpoint(t, runtimeID)

	// Concurrently connect and disconnect 20 clients rapidly
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(clientID int) {
			defer wg.Done()
			client, err := protocol.NewClient(runtimeID)
			if err != nil {
				return
			}
			defer client.Close()

			_ = client.Ping()
			_, _ = client.Status()
			_, _ = client.Send(protocol.CmdAttach, nil)
			_, _ = client.Send(protocol.CmdDetach, nil)
		}(i)
	}

	wg.Wait()

	// Verify host remains healthy and responsive
	finalClient, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("host unreachable after attach/detach spam: %v", err)
	}
	defer finalClient.Close()

	if err := finalClient.Ping(); err != nil {
		t.Errorf("ping failed after spam: %v", err)
	}
}

func TestQA_TwoWritersLeaseHandover(t *testing.T) {
	runtimeID := "rt-qa-lease"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  testTerminalBinary,
		Args:    testTerminalArgs(),
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()

	waitForHostEndpoint(t, runtimeID)

	// Writer A attaches and acquires lease
	writerA, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect Writer A: %v", err)
	}
	_, err = writerA.Send(protocol.CmdAttach, nil)
	if err != nil {
		t.Fatalf("Writer A attach failed: %v", err)
	}

	// Writer B attaches while Writer A is active
	writerB, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect Writer B: %v", err)
	}
	defer writerB.Close()
	_, err = writerB.Send(protocol.CmdAttach, nil)
	if err != nil {
		t.Fatalf("Writer B attach failed: %v", err)
	}

	// Disconnect Writer A, then wait for host responsiveness before Writer B input.
	writerA.Close()
	readyDeadline := time.Now().Add(2 * time.Second)
	for {
		if err := writerB.Ping(); err == nil {
			break
		}
		if time.Now().After(readyDeadline) {
			t.Fatal("host not responsive after Writer A disconnect")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Now Writer B should be able to send input without panic or deadlock
	writeDone := make(chan error, 1)
	go func() {
		_, err := writerB.RawConn().Write([]byte("echo handover success\n"))
		writeDone <- err
	}()

	select {
	case err := <-writeDone:
		if err != nil {
			t.Fatalf("Writer B input failed after Writer A disconnect: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Writer B input hung on lease handover")
	}
}

func TestQA_LargeThroughputStreaming(t *testing.T) {
	runtimeID := "rt-qa-stream"
	sess := registry.RuntimeSession{
		RuntimeID:    runtimeID,
		ProviderID:   "test",
		ProfileID:    "default",
		Workspace:    os.TempDir(),
		State:        registry.StateStarting,
		ControlLevel: registry.ControlLevelTerminal,
	}

	sh, err := NewSessionHost(Config{
		Session: sess,
		Binary:  testTerminalBinary,
		Args:    testTerminalArgs(),
		Env:     os.Environ(),
		Cwd:     os.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to create SessionHost: %v", err)
	}

	if err := sh.Start(); err != nil {
		t.Fatalf("failed to start SessionHost: %v", err)
	}
	defer sh.Stop()

	waitForHostEndpoint(t, runtimeID)

	client, err := protocol.NewClient(runtimeID)
	if err != nil {
		t.Fatalf("failed to connect client: %v", err)
	}
	defer client.Close()
	_, _ = client.Send(protocol.CmdAttach, nil)

	// Stream 100KB in chunks
	payload := bytes.Repeat([]byte("abcdefghijklmnopqrstuvwxyz012345\n"), 3000)
	n, err := client.RawConn().Write(payload)
	if err != nil || n != len(payload) {
		t.Fatalf("failed large payload streaming: %v (wrote %d of %d)", err, n, len(payload))
	}
}
