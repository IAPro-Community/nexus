package nexus

import (
	"github.com/IAPro-Community/nexus/internal/control/notify"
	"github.com/IAPro-Community/nexus/internal/nexus/store"
)

// ApplyNativeNotificationPreferences keeps native delivery aligned with the
// singleton preferences persisted by the local Nexus installation.
func ApplyNativeNotificationPreferences(preferences store.CapacityPreferences) {
	notify.SetCategoryPolicy(notify.Policy{
		Categories: map[notify.Category]bool{
			notify.CategoryActionRequired:     preferences.NotifyActionRequired,
			notify.CategoryAuthPermission:     preferences.NotifyAuthPermission,
			notify.CategoryCriticalFailure:    preferences.NotifyCriticalFailure,
			notify.CategoryQuotaExhausted:     preferences.NotifyQuotaExhausted,
			notify.CategoryQuotaLow:           preferences.NotifyQuotaLow,
			notify.CategoryCompletionRecovery: preferences.NotifyCompletionRecovery,
			notify.CategoryRuntimeConnection:  preferences.NotifyRuntimeConnection,
		},
		QuietHoursEnabled:         preferences.QuietHoursEnabled,
		QuietHoursStartUTC:        preferences.QuietHoursStartUTC,
		QuietHoursEndUTC:          preferences.QuietHoursEndUTC,
		ExhaustedBypassQuietHours: preferences.ExhaustedBypassQuietHours,
	})
}
