package promotion

import (
	"context"
	"fmt"
	"strings"

	"github.com/IAPro-Community/nexus/internal/nexus"
)

// Mode controls whether a composed prompt can leave the interactive Lead. It
// aliases Nexus' canonical DelegationMode so callers cannot accidentally
// create a second policy vocabulary at the transport boundary.
type Mode = nexus.DelegationMode

const (
	ModeAuto Mode = nexus.DelegationAuto
	ModeAsk  Mode = nexus.DelegationAsk
	ModeOff  Mode = nexus.DelegationOff
)

// Action is the only decision the terminal adapter needs to act on.
type Action string

const (
	KeepWithLead Action = "KEEP_WITH_LEAD"
	Promote      Action = "PROMOTE"
	Ask          Action = "ASK"
)

const (
	ActionKeepWithLead = KeepWithLead
	ActionPromote      = Promote
	ActionAsk          = Ask
)

// Decision contains the explainable inputs used to make the action decision.
type Decision struct {
	Action      Action                     `json:"action"`
	Mode        Mode                       `json:"mode"`
	Reason      string                     `json:"reason"`
	Intent      nexus.IntentDecision       `json:"intent"`
	Delegation  nexus.DelegationDecision   `json:"delegation"`
	MissionTeam nexus.MissionTeamSelection `json:"mission_team"`
}

type IntentDecider func(context.Context, string, string) (nexus.IntentDecision, error)

// Promoter reuses Nexus' canonical policy functions. The injectable decider
// keeps this package deterministic in tests while production uses the live
// project intelligence path.
type Promoter struct {
	Nexus        *nexus.Nexus
	DecideIntent IntentDecider
}

func NewPromoter(n *nexus.Nexus) Promoter { return Promoter{Nexus: n} }

func NormalizeMode(mode Mode) Mode {
	switch Mode(strings.ToUpper(strings.TrimSpace(string(mode)))) {
	case ModeAsk:
		return ModeAsk
	case ModeOff:
		return ModeOff
	default:
		return ModeAuto
	}
}

// Decide classifies one complete line. OFF, DIRECT and clarifying work remain
// with the existing Lead. PLAN intent is Mission-worthy even when no fan-out is
// beneficial: delegation shapes the Mission topology, it does not decide
// whether durable Mission/Flow history should exist.
func (p Promoter) Decide(ctx context.Context, projectID, line string, mode Mode) (Decision, error) {
	mode = NormalizeMode(mode)
	line = strings.TrimSpace(line)
	if line == "" {
		return Decision{Action: KeepWithLead, Mode: mode, Reason: "empty prompt"}, nil
	}
	if mode == ModeOff {
		return Decision{Action: KeepWithLead, Mode: mode, Reason: "delegation mode is OFF"}, nil
	}
	var intent nexus.IntentDecision
	var err error
	if p.DecideIntent != nil {
		intent, err = p.DecideIntent(ctx, projectID, line)
	} else if p.Nexus != nil && strings.TrimSpace(projectID) != "" {
		intent, err = p.Nexus.DecideIntentForProject(ctx, projectID, line)
	} else {
		intent = nexus.DecideIntent(line, nil)
	}
	if err != nil {
		return Decision{}, fmt.Errorf("decide prompt intent: %w", err)
	}
	delegation := nexus.DecideDelegation(line, mode)
	selection := nexus.SelectMissionTeam(line, intent)
	result := Decision{Action: KeepWithLead, Mode: mode, Intent: intent, Delegation: delegation, MissionTeam: selection}
	if intent.Strategy == nexus.IntentClarify {
		result.Reason = "intent requires clarification before promotion"
		return result, nil
	}
	if intent.Strategy == nexus.IntentDirect {
		result.Reason = "direct intent stays with the interactive Lead"
		return result, nil
	}
	if mode == ModeAsk {
		result.Action = Ask
		result.Reason = "mission-worthy plan requires confirmation before promotion"
		return result, nil
	}
	result.Action = Promote
	if delegation.Delegate {
		result.Reason = delegation.Reason
	} else {
		result.Reason = "plan intent is Mission-worthy; delegation remains a topology decision"
	}
	return result, nil
}
