package quota

import (
	"time"

	sdkquota "github.com/IAPro-Community/IAPro-SDK/quota"
)

// sdkView adapts the public Nexus projection to the SDK's shared quota rules.
// AccountScope is intentionally omitted: the Nexus view is serialized by
// product APIs and must not acquire new identity fields through this adapter.
func sdkView(view QuotaView) sdkquota.QuotaView {
	result := sdkquota.QuotaView{
		Provider:          view.Provider,
		Profile:           view.Profile,
		Account:           view.Account,
		Plan:              view.Plan,
		Status:            view.Status,
		Source:            view.Source,
		DiagnosticCode:    view.DiagnosticCode,
		DiagnosticMessage: view.DiagnosticMessage,
		FetchedAt:         view.FetchedAt,
		Available:         view.Available,
		AvailReasons: sdkquota.AvailReasons{
			ExhaustedWindows: append([]string(nil), view.AvailReasons.ExhaustedWindows...),
			RateLimited:      view.AvailReasons.RateLimited,
			UnknownQuota:     view.AvailReasons.UnknownQuota,
			AuthRequired:     view.AvailReasons.AuthRequired,
			AllOK:            view.AvailReasons.AllOK,
		},
		ModelGroups: view.ModelGroups,
	}
	return result
}

func applySDKAvailability(view *QuotaView, now time.Time) {
	shared := sdkView(*view)
	shared.ComputeAvailabilityAt(now)
	view.Available = shared.Available
	view.AvailReasons = AvailReasons{
		ExhaustedWindows: append([]string(nil), shared.AvailReasons.ExhaustedWindows...),
		RateLimited:      shared.AvailReasons.RateLimited,
		UnknownQuota:     shared.AvailReasons.UnknownQuota,
		AuthRequired:     shared.AvailReasons.AuthRequired,
		AllOK:            shared.AvailReasons.AllOK,
	}
}
