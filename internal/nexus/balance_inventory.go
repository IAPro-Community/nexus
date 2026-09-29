package nexus

import (
	"sort"
	"strings"

	"github.com/IAPro-Community/nexus/internal/core/quota"
)

// BalanceEntity is the routing-safe projection of a registered provider/profile.
//
// The source of truth is ListResources, which itself is built from profile.List()
// (the same registered identities exposed by nexus usage). Automatic balancing
// must never invent executors from PATH discovery, static provider tables or a
// model name remembered by the caller.
type BalanceEntity struct {
	ID             string             `json:"id"`
	Provider       string             `json:"provider"`
	Profile        string             `json:"profile"`
	DisplayName    string             `json:"display_name"`
	UsageStatus    string             `json:"usage_status"`
	UsageSource    string             `json:"usage_source,omitempty"`
	QuotaRemaining float64            `json:"quota_remaining"`
	Available      bool               `json:"available"`
	Health         string             `json:"health"`
	Eligibility    *quota.Eligibility `json:"eligibility,omitempty"`
	ModelGroups    []BalanceModelGroup `json:"model_groups,omitempty"`
}

// BalanceModelGroup is the bounded capacity summary used by routing surfaces.
// It intentionally exposes no credentials, provider-home paths or raw usage
// payloads.
type BalanceModelGroup struct {
	Key       string  `json:"key,omitempty"`
	Name      string  `json:"name"`
	Remaining float64 `json:"remaining"`
	Known     bool    `json:"known"`
	Available bool    `json:"available"`
}

// BuildBalanceInventory projects only registered resources for which Nexus has
// measured quota evidence. UNKNOWN/UNSUPPORTED entities remain visible in
// nexus usage/ListResources for diagnostics and manual setup, but they never
// enter this automatic-balancing inventory.
func BuildBalanceInventory(accounts []ProviderAccount) []BalanceEntity {
	entities := make([]BalanceEntity, 0, len(accounts))
	for _, account := range accounts {
		if !hasMeasuredBalanceEvidence(account) {
			continue
		}

		view := account.QuotaView
		entity := BalanceEntity{
			ID:             account.ID,
			Provider:       account.Provider,
			Profile:        account.Profile,
			DisplayName:    account.DisplayName,
			UsageStatus:    strings.ToUpper(strings.TrimSpace(view.Status)),
			UsageSource:    strings.TrimSpace(view.Source),
			QuotaRemaining: account.QuotaRemaining,
			Available:      account.Available,
			Health:         account.Health,
			Eligibility:    account.Eligibility,
			ModelGroups:    make([]BalanceModelGroup, 0, len(view.ModelGroups)),
		}
		for _, group := range view.ModelGroups {
			remaining, known := group.GroupRemaining()
			entity.ModelGroups = append(entity.ModelGroups, BalanceModelGroup{
				Key:       group.Key,
				Name:      group.Name,
				Remaining: remaining,
				Known:     known,
				Available: known && remaining > 0,
			})
		}
		entities = append(entities, entity)
	}

	sort.SliceStable(entities, func(i, j int) bool {
		if entities[i].Provider != entities[j].Provider {
			return entities[i].Provider < entities[j].Provider
		}
		return entities[i].Profile < entities[j].Profile
	})
	return entities
}

// BalanceCandidateAccounts returns the ProviderAccount objects that are safe to
// feed into automatic cross-provider selection. Manual selection deliberately
// continues to use the complete ListResources result.
func BalanceCandidateAccounts(accounts []ProviderAccount) []ProviderAccount {
	out := make([]ProviderAccount, 0, len(accounts))
	for _, account := range accounts {
		if hasMeasuredBalanceEvidence(account) {
			out = append(out, account)
		}
	}
	return out
}

func hasMeasuredBalanceEvidence(account ProviderAccount) bool {
	if strings.TrimSpace(account.Provider) == "" || strings.TrimSpace(account.Profile) == "" {
		return false
	}
	if !account.Authenticated || account.QuotaView == nil {
		return false
	}

	view := account.QuotaView
	status := strings.ToUpper(strings.TrimSpace(view.Status))
	switch status {
	case "LIVE", "CACHED", "ESTIMATED", "RATE_LIMITED", "OK":
		// RATE_LIMITED is still measured evidence. Eligibility will reject it,
		// but retaining it in the inventory lets the UI explain why it lost.
		// OK is the bounded compatibility status used by older measured
		// aggregate snapshots and is treated as cached evidence elsewhere.
	default:
		return false
	}

	// A measured status without any scorable group/window is not enough to
	// claim that Nexus knows this resource's limit.
	for _, group := range view.ModelGroups {
		if _, ok := group.GroupRemaining(); ok {
			return true
		}
	}

	// Some providers expose a measured aggregate without model-group detail.
	// Preserve that compatibility only when a normalized aggregate is present.
	return account.QuotaTotal > 0 && account.QuotaRemaining >= 0
}
