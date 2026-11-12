package nexus

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/protocol"
	"github.com/IAPro-Community/nexus/internal/control/registry"
)

type attentionTransportFixture struct {
	runtimeID string
	requests  chan protocol.Request
	count     atomic.Int32
	release   chan struct{}
	once      sync.Once
	close     func() error
}

func newAttentionTransportFixture(t *testing.T, acknowledge bool) *attentionTransportFixture {
	t.Helper()
	f := &attentionTransportFixture{
		runtimeID: fmt.Sprintf("attention-retry-%d", time.Now().UnixNano()),
		requests:  make(chan protocol.Request, 8),
		release:   make(chan struct{}),
	}
	listener, err := protocol.Listen(f.runtimeID)
	if err != nil {
		t.Fatal(err)
	}
	f.close = listener.Close
	t.Cleanup(func() {
		f.once.Do(func() { close(f.release) })
		_ = listener.Close()
	})
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer conn.Close()
				var request protocol.Request
				if err := json.NewDecoder(conn).Decode(&request); err != nil {
					return
				}
				f.count.Add(1)
				f.requests <- request
				<-f.release
				if acknowledge {
					_ = json.NewEncoder(conn).Encode(protocol.Response{Version: protocol.ProtocolVersion, ID: request.ID, OK: true})
				}
			}()
		}
	}()
	return f
}

func attentionRetrySession(runtimeID string, structured bool) (registry.RuntimeSession, registry.AttentionResponseExpectation) {
	session := registry.RuntimeSession{
		RuntimeID: runtimeID, State: registry.StateWaiting,
		HostGeneration: 44, ProviderGeneration: 55,
		AttentionReason: "QUESTION", AttentionKind: "needs_user", AttentionContext: "Continue?",
		PromptKind: "yn", AttentionSource: "INFERRED", AttentionFingerprint: "retry-fingerprint", AttentionRevision: 3,
	}
	expected := registry.AttentionResponseExpectation{Fingerprint: session.AttentionFingerprint, Revision: 3, Generation: "44:55"}
	if structured {
		session.State = registry.StateApproval
		session.AttentionReason = "APPROVAL"
		session.PromptKind = "choice"
		session.AttentionSource = "AUTHORITATIVE"
		session.AttentionInteractionID = "req-1"
		session.AttentionOptions = []registry.AttentionOption{{ID: "allow_once", Name: "Allow once"}, {ID: "deny", Name: "Deny"}}
		expected.InteractionID = session.AttentionInteractionID
	}
	return session, expected
}

func TestAttentionResponseRetryNeverDispatchesTwice(t *testing.T) {
	transport := newAttentionTransportFixture(t, true)
	path := t.TempDir() + "/runtimes.json"
	reg := registry.NewRegistry(path)
	session, expected := attentionRetrySession(transport.runtimeID, false)
	if err := reg.Register(session); err != nil {
		t.Fatal(err)
	}
	service := NewRuntimeApplicationService(reg, nil, nil)
	first := make(chan error, 1)
	go func() { first <- service.RespondToAttention(context.Background(), transport.runtimeID, expected, "y") }()
	select {
	case request := <-transport.requests:
		if request.Command != protocol.CmdInput {
			t.Fatalf("command=%s, want input", request.Command)
		}
		var input protocol.InputPayload
		if err := json.Unmarshal(request.Payload, &input); err != nil || input.Data != "y\n" {
			t.Fatalf("input=%+v err=%v", input, err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("initial response did not reach the transport")
	}
	if err := service.RespondToAttention(context.Background(), transport.runtimeID, expected, "y"); !errors.Is(err, ErrAttentionDeliveryUncertain) {
		t.Fatalf("concurrent retry should report pending delivery, got %v", err)
	}
	if err := service.RespondToAttention(context.Background(), transport.runtimeID, expected, "n"); !errors.Is(err, registry.ErrAttentionConflict) {
		t.Fatalf("competing answer must conflict while delivery is pending, got %v", err)
	}
	transport.once.Do(func() { close(transport.release) })
	if err := <-first; err != nil {
		t.Fatalf("initial response: %v", err)
	}
	if err := reg.UpdateAttentionMeta(transport.runtimeID, registry.AttentionUpdate{State: registry.StateRunning, Kind: "working"}); err != nil {
		t.Fatal(err)
	}
	_ = transport.close()
	restarted := NewRuntimeApplicationService(registry.NewRegistry(path), nil, nil)
	if err := restarted.RespondToAttention(context.Background(), transport.runtimeID, expected, "y"); err != nil {
		t.Fatalf("submitted retry after restart: %v", err)
	}
	if err := restarted.RespondToAttention(context.Background(), transport.runtimeID, expected, "n"); !errors.Is(err, registry.ErrAttentionConflict) {
		t.Fatalf("changed answer must conflict, got %v", err)
	}
	stale := expected
	stale.Revision++
	if err := restarted.RespondToAttention(context.Background(), transport.runtimeID, stale, "y"); !errors.Is(err, registry.ErrAttentionConflict) {
		t.Fatalf("changed revision must conflict, got %v", err)
	}
	if got := transport.count.Load(); got != 1 {
		t.Fatalf("transport dispatches=%d, want 1", got)
	}
}

func TestAttentionUncertainDeliveryRetryRemainsUncertainAfterRestart(t *testing.T) {
	transport := newAttentionTransportFixture(t, false)
	transport.once.Do(func() { close(transport.release) })
	path := t.TempDir() + "/runtimes.json"
	reg := registry.NewRegistry(path)
	session, expected := attentionRetrySession(transport.runtimeID, false)
	if err := reg.Register(session); err != nil {
		t.Fatal(err)
	}
	service := NewRuntimeApplicationService(reg, nil, nil)
	if err := service.RespondToAttention(context.Background(), transport.runtimeID, expected, "y"); !errors.Is(err, ErrAttentionDeliveryUncertain) {
		t.Fatalf("dropped transport acknowledgement must be uncertain, got %v", err)
	}
	_ = transport.close()
	restarted := NewRuntimeApplicationService(registry.NewRegistry(path), nil, nil)
	if err := restarted.RespondToAttention(context.Background(), transport.runtimeID, expected, "y"); !errors.Is(err, ErrAttentionDeliveryUncertain) {
		t.Fatalf("uncertain retry after restart: %v", err)
	}
	if got := transport.count.Load(); got != 1 {
		t.Fatalf("uncertain operation was resent: dispatches=%d", got)
	}
}

func TestAttentionPendingClaimAfterRestartNeverDispatches(t *testing.T) {
	transport := newAttentionTransportFixture(t, true)
	transport.once.Do(func() { close(transport.release) })
	path := t.TempDir() + "/runtimes.json"
	reg := registry.NewRegistry(path)
	session, expected := attentionRetrySession(transport.runtimeID, false)
	if err := reg.Register(session); err != nil {
		t.Fatal(err)
	}
	digest := attentionOperationDigest(expected, registry.AttentionOperationResponse, "y")
	if claim, err := reg.ClaimAttentionOperation(session.RuntimeID, expected, registry.AttentionOperationResponse, digest, ""); err != nil || !claim.Dispatch {
		t.Fatalf("initial durable reservation=%+v err=%v", claim, err)
	}
	restarted := NewRuntimeApplicationService(registry.NewRegistry(path), nil, nil)
	if err := restarted.RespondToAttention(context.Background(), session.RuntimeID, expected, "y"); !errors.Is(err, ErrAttentionDeliveryUncertain) {
		t.Fatalf("pending retry after restart: %v", err)
	}
	if got := transport.count.Load(); got != 0 {
		t.Fatalf("pending operation was dispatched after restart: %d requests", got)
	}
}

func TestAttentionStructuredDenyRetryUsesExactOptionWithoutResending(t *testing.T) {
	transport := newAttentionTransportFixture(t, true)
	transport.once.Do(func() { close(transport.release) })
	path := t.TempDir() + "/runtimes.json"
	reg := registry.NewRegistry(path)
	session, expected := attentionRetrySession(transport.runtimeID, true)
	if err := reg.Register(session); err != nil {
		t.Fatal(err)
	}
	service := NewRuntimeApplicationService(reg, nil, nil)
	if err := service.ChooseAttention(context.Background(), transport.runtimeID, expected, "deny"); err != nil {
		t.Fatalf("structured deny: %v", err)
	}
	request := <-transport.requests
	var choice protocol.ChooseInteractionPayload
	if err := json.Unmarshal(request.Payload, &choice); err != nil || request.Command != protocol.CmdChooseInteraction || choice.InteractionID != "req-1" || choice.OptionID != "deny" {
		t.Fatalf("structured choice command=%s choice=%+v err=%v", request.Command, choice, err)
	}
	_ = transport.close()
	restarted := NewRuntimeApplicationService(registry.NewRegistry(path), nil, nil)
	if err := restarted.ChooseAttention(context.Background(), transport.runtimeID, expected, "deny"); err != nil {
		t.Fatalf("structured deny retry after restart: %v", err)
	}
	if err := restarted.ChooseAttention(context.Background(), transport.runtimeID, expected, "allow_once"); !errors.Is(err, registry.ErrAttentionConflict) {
		t.Fatalf("different option must conflict, got %v", err)
	}
	if got := transport.count.Load(); got != 1 {
		t.Fatalf("structured choice dispatches=%d, want 1", got)
	}
}

func TestAttentionStructuredDenyUncertainRetryNeverResends(t *testing.T) {
	transport := newAttentionTransportFixture(t, false)
	transport.once.Do(func() { close(transport.release) })
	path := t.TempDir() + "/runtimes.json"
	reg := registry.NewRegistry(path)
	session, expected := attentionRetrySession(transport.runtimeID, true)
	if err := reg.Register(session); err != nil {
		t.Fatal(err)
	}
	service := NewRuntimeApplicationService(reg, nil, nil)
	if err := service.ChooseAttention(context.Background(), transport.runtimeID, expected, "deny"); !errors.Is(err, ErrAttentionDeliveryUncertain) {
		t.Fatalf("structured deny without acknowledgement: %v", err)
	}
	restarted := NewRuntimeApplicationService(registry.NewRegistry(path), nil, nil)
	if err := restarted.ChooseAttention(context.Background(), transport.runtimeID, expected, "deny"); !errors.Is(err, ErrAttentionDeliveryUncertain) {
		t.Fatalf("uncertain structured deny retry: %v", err)
	}
	if got := transport.count.Load(); got != 1 {
		t.Fatalf("uncertain structured deny dispatches=%d, want 1", got)
	}
}
