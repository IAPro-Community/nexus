package nexus

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/events"
	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

func TestMissionNotificationDeliveryIsIdempotentAndRestartSafe(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "mission-notifications.db")
	st, run, repo := newMissionNotificationFixture(t, dbPath, "mission-notification-restart")
	now := time.Date(2026, 10, 25, 13, 0, 0, 0, time.UTC)
	transport := &missionNotificationTransport{available: true}
	service := NewMissionNotificationService(st, transport)
	service.now = func() time.Time { return now }
	if err := service.ObserveRun(run); err != nil {
		t.Fatal(err)
	}

	run.State = runner.StateBlockedNeedsUser
	run.NeedsHuman = &runner.HumanIntervention{ID: "intervention-notify", Version: 1, ReasonCode: "SECURITY"}
	run.UpdatedAt = now.Add(time.Second)
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	if err := service.ObserveRun(run); err != nil {
		t.Fatal(err)
	}
	if err := service.ObserveRun(run); err != nil {
		t.Fatal(err)
	}
	if len(transport.payloads) != 1 {
		t.Fatalf("same blocker should deliver once, got %d payloads", len(transport.payloads))
	}
	payload := transport.payloads[0]
	if payload.Tag != runner.AttentionDedupKey(run) {
		t.Fatalf("notification tag is not tied to the current intervention: %+v", payload)
	}
	for _, forbidden := range []string{"intervention-notify", "SECURITY", "http://", "https://", "token", "secret"} {
		if strings.Contains(strings.ToLower(payload.Title+" "+payload.Body), strings.ToLower(forbidden)) {
			t.Fatalf("notification payload leaked %q: %+v", forbidden, payload)
		}
	}
	delivery, err := st.GetMissionNotificationDelivery(payload.Tag, store.MissionNotificationNative)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Status != store.MissionNotificationDelivered || delivery.Attempts != 1 {
		t.Fatalf("native delivery was not durably recorded: %+v", delivery)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	transportAfterRestart := &missionNotificationTransport{}
	serviceAfterRestart := NewMissionNotificationService(st, transportAfterRestart)
	serviceAfterRestart.now = func() time.Time { return now.Add(time.Minute) }
	if err := serviceAfterRestart.Recover(ctx, []*runner.MissionRun{run}); err != nil {
		t.Fatal(err)
	}
	if len(transportAfterRestart.payloads) != 0 {
		t.Fatalf("delivered intent repeated after restart: %+v", transportAfterRestart.payloads)
	}
}

func TestMissionNotificationFailureRetriesAfterRestart(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "mission-notifications.db")
	st, run, repo := newMissionNotificationFixture(t, dbPath, "mission-notification-retry")
	start := time.Date(2026, 10, 25, 14, 0, 0, 0, time.UTC)
	firstTransport := &missionNotificationTransport{available: true, err: errors.New("temporary desktop transport failure: Authorization: Bearer test-secret-token-value")}
	service := NewMissionNotificationService(st, firstTransport)
	service.now = func() time.Time { return start }
	if err := service.ObserveRun(run); err != nil {
		t.Fatal(err)
	}
	run.State = runner.StateBlockedNeedsUser
	run.NeedsHuman = &runner.HumanIntervention{ID: "intervention-retry", Version: 1, ReasonCode: "SECURITY"}
	run.UpdatedAt = start.Add(time.Second)
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return start.Add(time.Second) }
	if err := service.ObserveRun(run); err != nil {
		t.Fatal(err)
	}
	key := runner.AttentionDedupKey(run)
	delivery, err := st.GetMissionNotificationDelivery(key, store.MissionNotificationNative)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Status != store.MissionNotificationPending || delivery.Attempts != 1 || !strings.Contains(delivery.LastError, "temporary") {
		t.Fatalf("failed transport was not kept retryable: %+v", delivery)
	}
	if strings.Contains(delivery.LastError, "test-secret-token-value") {
		t.Fatalf("notification delivery error persisted a credential: %q", delivery.LastError)
	}
	if err := st.Close(); err != nil {
		t.Fatal(err)
	}

	st, err = store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	secondTransport := &missionNotificationTransport{available: true}
	serviceAfterRestart := NewMissionNotificationService(st, secondTransport)
	serviceAfterRestart.now = func() time.Time { return start.Add(2 * time.Second) }
	if err := serviceAfterRestart.Recover(ctx, []*runner.MissionRun{run}); err != nil {
		t.Fatal(err)
	}
	if len(secondTransport.payloads) != 1 || secondTransport.payloads[0].Tag != key {
		t.Fatalf("pending native delivery did not recover exactly once: %+v", secondTransport.payloads)
	}
	delivery, err = st.GetMissionNotificationDelivery(key, store.MissionNotificationNative)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Status != store.MissionNotificationDelivered || delivery.Attempts != 2 {
		t.Fatalf("successful retry was not recorded: %+v", delivery)
	}
}

func TestMissionNotificationUnavailableChannelDefersWithoutAttempt(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "mission-notifications.db")
	st, run, repo := newMissionNotificationFixture(t, dbPath, "mission-notification-capability")
	defer st.Close()
	now := time.Date(2026, 10, 25, 15, 0, 0, 0, time.UTC)
	transport := &missionNotificationTransport{available: false}
	service := NewMissionNotificationService(st, transport)
	service.now = func() time.Time { return now }
	if err := service.ObserveRun(run); err != nil {
		t.Fatal(err)
	}
	run.State = runner.StateBlockedNeedsUser
	run.NeedsHuman = &runner.HumanIntervention{ID: "intervention-capability", Version: 1, ReasonCode: "SECURITY"}
	run.UpdatedAt = now.Add(time.Second)
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	if err := service.ObserveRun(run); err != nil {
		t.Fatal(err)
	}
	key := runner.AttentionDedupKey(run)
	delivery, err := st.GetMissionNotificationDelivery(key, store.MissionNotificationNative)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Status != store.MissionNotificationPending || delivery.Attempts != 0 || len(transport.payloads) != 0 {
		t.Fatalf("unavailable channel should remain pending without a send attempt: delivery=%+v payloads=%+v", delivery, transport.payloads)
	}
	transport.available = true
	now = now.Add(missionNotificationUnavailable + time.Second)
	service.now = func() time.Time { return now }
	if err := service.DeliverPending(ctx); err != nil {
		t.Fatal(err)
	}
	delivery, err = st.GetMissionNotificationDelivery(key, store.MissionNotificationNative)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Status != store.MissionNotificationDelivered || len(transport.payloads) != 1 {
		t.Fatalf("native delivery did not resume when capability became available: delivery=%+v payloads=%+v", delivery, transport.payloads)
	}
}

func TestMissionNotificationRetryReachesTransportThroughGlobalThrottle(t *testing.T) {
	ctx := context.Background()
	st, run, repo := newMissionNotificationFixture(t, filepath.Join(t.TempDir(), "mission-notification-throttle.db"), "mission-notification-throttle")
	defer st.Close()
	start := time.Date(2026, 10, 25, 16, 0, 0, 0, time.UTC)
	transport := &missionNotificationFlakyTransport{failuresRemaining: 1}
	service := NewMissionNotificationService(st, notify.NewThrottled(transport, time.Minute))
	service.now = func() time.Time { return start }
	if err := service.ObserveRun(run); err != nil {
		t.Fatal(err)
	}
	run.State = runner.StateBlockedNeedsUser
	run.NeedsHuman = &runner.HumanIntervention{ID: "intervention-throttle", Version: 1, ReasonCode: "SECURITY"}
	run.UpdatedAt = start.Add(time.Second)
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return start.Add(time.Second) }
	if err := service.ObserveRun(run); err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return start.Add(2 * time.Second) }
	if err := service.DeliverPending(ctx); err != nil {
		t.Fatal(err)
	}
	if transport.calls != 2 {
		t.Fatalf("retry was suppressed by the process-wide throttle: calls=%d", transport.calls)
	}
	delivery, err := st.GetMissionNotificationDelivery(runner.AttentionDedupKey(run), store.MissionNotificationNative)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Status != store.MissionNotificationDelivered || delivery.Attempts != 2 {
		t.Fatalf("retry through throttle was not recorded as delivered: %+v", delivery)
	}
}

func TestMissionNotificationsExcludeQuotaAutomationAndKeepCompletionInApp(t *testing.T) {
	for _, reason := range []string{"QUOTA_EXHAUSTED", "RATE_LIMITED", "RETRY_PENDING", "REMEDIATION_REQUIRED"} {
		blocked := &runner.MissionRun{
			ID: "auto-block-" + reason, State: runner.StateBlockedNeedsUser,
			NeedsHuman: &runner.HumanIntervention{ID: "intv-" + reason, Version: 1, ReasonCode: reason},
		}
		if missionNativeNotificationEligible(blocked) {
			t.Errorf("internal %s transition must not create an interrupting native notification", reason)
		}
	}
	for _, state := range []runner.State{runner.StateExecuting, runner.StateCompletedVerified, runner.StateFailedNoProgress} {
		run := &runner.MissionRun{ID: "mission-" + string(state), State: state}
		if state == runner.StateFailedNoProgress {
			run.Contract.EscalateOnFailure = false
		}
		if missionNativeNotificationEligible(run) {
			t.Errorf("non-blocker state %s should not notify without an escalation contract", state)
		}
	}
	completed := &runner.MissionRun{ID: "mission-completed", State: runner.StateCompletedVerified}
	if item := runner.AttentionFromRun(completed); item == nil || item.State != runner.StateCompletedVerified {
		t.Fatalf("verified completion should remain in the in-app Attention projection: %+v", item)
	}
	escalatedFailure := &runner.MissionRun{
		ID: "mission-escalated-failure", State: runner.StateFailedNoProgress,
		Contract: runner.AutonomyContract{EscalateOnFailure: true},
	}
	if !missionNativeNotificationEligible(escalatedFailure) {
		t.Fatal("an explicitly escalated terminal Mission failure should notify")
	}
}

func TestRunApplicationPublishesMissionBlockerThroughDurableNotificationOutbox(t *testing.T) {
	ctx := context.Background()
	st, run, repo := newMissionNotificationFixture(t, filepath.Join(t.TempDir(), "mission-notification-app.db"), "mission-notification-app")
	defer st.Close()
	transport := &missionNotificationTransport{available: true}
	notifications := NewMissionNotificationService(st, transport)
	service := &RunApplicationService{bus: events.NewBus(10), notifications: notifications}
	now := time.Now().UTC()
	notifications.now = func() time.Time { return now }
	service.publishRunEvent(run, events.EventMissionStarted, "Mission run started", nil)

	run.State = runner.StateBlockedNeedsUser
	run.NeedsHuman = &runner.HumanIntervention{ID: "intervention-published", Version: 3, ReasonCode: "SECURITY"}
	run.UpdatedAt = now.Add(time.Second)
	if err := repo.SaveRun(ctx, run); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	service.publishRunEvent(run, events.EventHumanInterventionCreated, "Human intervention created", nil)
	if len(transport.payloads) != 1 {
		t.Fatalf("application event did not trigger exactly one native notification: %+v", transport.payloads)
	}
	delivery, err := st.GetMissionNotificationDelivery(runner.AttentionDedupKey(run), store.MissionNotificationNative)
	if err != nil {
		t.Fatal(err)
	}
	if delivery.Status != store.MissionNotificationDelivered {
		t.Fatalf("application event did not persist delivery receipt: %+v", delivery)
	}
}

func newMissionNotificationFixture(t *testing.T, dbPath, runID string) (*store.Store, *runner.MissionRun, *storeRunRepository) {
	t.Helper()
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	project, err := st.CreateProject(store.Project{Name: "Mission notification fixture", CanonicalPath: t.TempDir()})
	if err != nil {
		st.Close()
		t.Fatal(err)
	}
	plan, err := st.CreateWorkPlan(store.WorkPlan{ProjectID: project.ID, Title: "Mission notification fixture"})
	if err != nil {
		st.Close()
		t.Fatal(err)
	}
	run := &runner.MissionRun{ID: runID, ProjectID: project.ID, PlanID: plan.ID, State: runner.StateExecuting, UpdatedAt: time.Now().UTC()}
	repo := newStoreRunRepository(st).(*storeRunRepository)
	if err := repo.SaveRun(context.Background(), run); err != nil {
		st.Close()
		t.Fatal(err)
	}
	return st, run, repo
}

type missionNotificationTransport struct {
	available bool
	err       error
	payloads  []notify.Payload
}

func (n *missionNotificationTransport) Available() bool { return n.available }
func (n *missionNotificationTransport) Notify(payload notify.Payload) error {
	n.payloads = append(n.payloads, payload)
	return n.err
}

type missionNotificationFlakyTransport struct {
	failuresRemaining int
	calls             int
}

func (*missionNotificationFlakyTransport) Available() bool { return true }
func (n *missionNotificationFlakyTransport) Notify(notify.Payload) error {
	n.calls++
	if n.failuresRemaining > 0 {
		n.failuresRemaining--
		return errors.New("temporary desktop transport failure")
	}
	return nil
}
