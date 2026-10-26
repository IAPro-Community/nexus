package nexus

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/core/security"
	"github.com/IAPro-Community/nexus/internal/nexus/runner"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

const (
	missionNotificationKindBlocked = "MISSION_BLOCKED"
	missionNotificationKindFailed  = "MISSION_FAILED"
	missionNotificationRetryMax    = 5 * time.Minute
	missionNotificationUnavailable = 5 * time.Minute
)

// MissionNotificationService delivers only durable transport intents. Mission
// state and the in-app Attention Center remain derived from MissionRun.
type MissionNotificationService struct {
	store    *store.Store
	notifier notify.Notifier
	now      func() time.Time
}

func NewMissionNotificationService(st *store.Store, notifier notify.Notifier) *MissionNotificationService {
	if notifier == nil {
		notifier = notify.Default()
	}
	return &MissionNotificationService{store: st, notifier: notifier, now: func() time.Time { return time.Now().UTC() }}
}

// Recover observes persisted runs after host start and retries due deliveries.
// Existing runs with no cursor establish a quiet baseline; a transition that
// happened after an earlier observation is re-enqueued from the current state.
func (s *MissionNotificationService) Recover(ctx context.Context, runs []*runner.MissionRun) error {
	if s == nil || s.store == nil {
		return nil
	}
	for _, run := range runs {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := s.observe(run); err != nil {
			return err
		}
	}
	return s.DeliverPending(ctx)
}

// ObserveRun records a new current-state projection and attempts eligible
// native delivery. Errors are returned for diagnostics; callers must not let a
// notification transport failure affect the Mission operation.
func (s *MissionNotificationService) ObserveRun(run *runner.MissionRun) error {
	if s == nil || s.store == nil || run == nil {
		return nil
	}
	if err := s.observe(run); err != nil {
		return err
	}
	return s.DeliverPending(context.Background())
}

func (s *MissionNotificationService) observe(run *runner.MissionRun) error {
	if run == nil {
		return nil
	}
	intent := missionNotificationIntentForRun(run)
	intent.CreatedAt = s.currentTime()
	_, err := s.store.ObserveMissionNotification(intent, missionNativeNotificationEligible(run), intent.CreatedAt)
	return err
}

func (s *MissionNotificationService) DeliverPending(ctx context.Context) error {
	if s == nil || s.store == nil {
		return nil
	}
	now := s.currentTime()
	deliveries, err := s.store.ListPendingMissionNotificationDeliveries(now, 50)
	if err != nil {
		return err
	}
	for _, delivery := range deliveries {
		if err := ctx.Err(); err != nil {
			return err
		}
		if delivery.Channel != store.MissionNotificationNative {
			continue
		}
		if !notify.IsAvailable(s.notifier) {
			if err := s.store.DeferMissionNotificationDelivery(delivery.IntentKey, delivery.Channel,
				"native desktop notification channel unavailable", now.Add(missionNotificationUnavailable), now); err != nil {
				return err
			}
			continue
		}
		err := s.notifier.Notify(notify.Payload{Title: delivery.Title, Body: delivery.Body, Tag: delivery.IntentKey})
		now = s.currentTime()
		if err == nil {
			if err := s.store.RecordMissionNotificationAttempt(delivery.IntentKey, delivery.Channel,
				store.MissionNotificationDelivered, "", now, now); err != nil {
				return err
			}
			continue
		}
		attempt := delivery.Attempts + 1
		retryAt := now.Add(missionNotificationRetryDelay(attempt))
		if persistErr := s.store.RecordMissionNotificationAttempt(delivery.IntentKey, delivery.Channel,
			store.MissionNotificationPending, security.Redact(err.Error()), retryAt, now); persistErr != nil {
			return fmt.Errorf("record native notification failure: %w", persistErr)
		}
	}
	return nil
}

func (s *MissionNotificationService) currentTime() time.Time {
	if s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

func missionNotificationIntentForRun(run *runner.MissionRun) store.MissionNotificationIntent {
	intent := store.MissionNotificationIntent{
		Key: runner.AttentionDedupKey(run), RunID: run.ID, ProjectID: run.ProjectID, SourceID: run.ID,
		Kind: string(run.State),
	}
	if run.NeedsHuman != nil {
		intent.SourceID = run.NeedsHuman.ID
		intent.SourceVersion = run.NeedsHuman.Version
	}
	switch run.State {
	case runner.StateBlockedNeedsUser:
		intent.Kind = missionNotificationKindBlocked
		intent.Title = "Nexus — missão aguardando você"
		intent.Body = "Uma missão precisa de uma decisão. Abra o Centro de Atenção."
	case runner.StateFailedNoProgress, runner.StateFailedVerification, runner.StateFailedBudgetExceeded, runner.StateFailed:
		intent.Kind = missionNotificationKindFailed
		intent.Title = "Nexus — missão interrompida"
		intent.Body = "Uma missão terminou sem concluir a verificação. Abra o Centro de Atenção."
	case runner.StateCompletedVerified:
		intent.Kind = "MISSION_COMPLETED"
		intent.Title = "Nexus — missão verificada"
		intent.Body = "Uma missão terminou e passou pela verificação."
	default:
		intent.Kind = string(run.State)
	}
	return intent
}

func missionNativeNotificationEligible(run *runner.MissionRun) bool {
	if run == nil {
		return false
	}
	switch run.State {
	case runner.StateBlockedNeedsUser:
		if run.NeedsHuman == nil || strings.TrimSpace(run.NeedsHuman.ID) == "" || run.NeedsHuman.Version <= 0 {
			return false
		}
		reason := strings.ToUpper(strings.TrimSpace(run.NeedsHuman.ReasonCode))
		for _, internal := range []string{"QUOTA", "RATE_LIMIT", "RETRY", "REMEDIATION"} {
			if strings.Contains(reason, internal) {
				return false
			}
		}
		return true
	case runner.StateFailedNoProgress, runner.StateFailedVerification, runner.StateFailedBudgetExceeded, runner.StateFailed:
		return run.Contract.EscalateOnFailure
	default:
		return false
	}
}

func missionNotificationRetryDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return time.Second
	}
	delay := time.Second << min(attempt-1, 9)
	if delay > missionNotificationRetryMax {
		return missionNotificationRetryMax
	}
	return delay
}
