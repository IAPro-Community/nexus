package nexus

import (
	"strings"
	"time"

	"github.com/kivervinicius/ai-cli/internal/core/model"
	"github.com/kivervinicius/ai-cli/internal/core/quota"
)

func accountEligibilityInput(acc ProviderAccount, explicit bool) quota.EligibilityInput {
	view := quota.QuotaView{}
	if acc.QuotaView != nil {
		view = *acc.QuotaView
	} else {
		// Legacy accounts without a QuotaView remain routable when Available;
		// do not invent UNKNOWN (that would hard-block AUTO allocation).
		if acc.Available {
			view.Status = string(model.UsageCached)
		}
	}
	// Account-level availability/reasons are authoritative for routing. QuotaView
	// fixtures often omit ComputeAvailability(), leaving Available=false.
	view.Available = acc.Available
	if acc.AvailReasons != nil {
		view.AvailReasons = *acc.AvailReasons
	}
	if strings.TrimSpace(view.Status) == "" && acc.Available {
		view.Status = string(model.UsageCached)
	}
	inCooldown := acc.CooldownUntil != nil && time.Now().Before(*acc.CooldownUntil)
	if acc.RateLimited {
		view.AvailReasons.RateLimited = true
		if strings.TrimSpace(view.Status) == "" || strings.EqualFold(view.Status, string(model.UsageCached)) || strings.EqualFold(view.Status, string(model.UsageLive)) {
			view.Status = string(model.UsageRateLimited)
		}
	}
	return quota.EligibilityInput{
		View:              view,
		Authenticated:     acc.Authenticated,
		Health:            acc.Health,
		NetworkError:      acc.NetworkError,
		InCooldown:        inCooldown,
		ExplicitSelection: explicit,
	}
}
