package promotion

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

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
	Action                 Action                     `json:"action"`
	Mode                   Mode                       `json:"mode"`
	Reason                 string                     `json:"reason"`
	Intent                 nexus.IntentDecision       `json:"intent"`
	Delegation             nexus.DelegationDecision   `json:"delegation"`
	MissionTeam            nexus.MissionTeamSelection `json:"mission_team"`
	ExplicitMissionRequest bool                       `json:"explicit_mission_request,omitempty"`
}

type IntentDecider func(context.Context, string, string) (nexus.IntentDecision, error)

// Promoter reuses Nexus' canonical policy functions. The injectable decider
// keeps this package deterministic in tests while production uses the live
// project intelligence path.
type Promoter struct {
	Nexus        *nexus.Nexus
	DecideIntent IntentDecider
}

// InteractiveDecisionTimeout is the shared latency budget for deciding
// whether Enter stays with the Lead or promotes the prompt to a Mission.
const InteractiveDecisionTimeout = 250 * time.Millisecond

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

// Decide classifies one complete line. Interactive use is provider-first:
// OFF, DIRECT, clarifying work and ordinary single-topology plans remain with
// the existing Lead. A Mission is promoted only when the request explicitly
// asks for one, requires decomposition, or benefits from delegated workstreams.
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
	explicitRequest := IsExplicitMissionRequest(line)
	result := Decision{
		Action:                 KeepWithLead,
		Mode:                   mode,
		Intent:                 intent,
		Delegation:             delegation,
		MissionTeam:            selection,
		ExplicitMissionRequest: explicitRequest,
	}
	if intent.Strategy == nexus.IntentClarify {
		result.Reason = "intent requires clarification before promotion"
		return result, nil
	}
	if intent.Strategy == nexus.IntentDirect && !explicitRequest {
		result.Reason = "direct intent stays with the interactive Lead"
		return result, nil
	}

	missionWorthy := delegation.Delegate || intent.TaskRequirements.RequiresDecomposition || explicitRequest
	if !missionWorthy {
		result.Reason = "interactive plan stays with the Lead; no delegation or decomposition is required"
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
	} else if intent.TaskRequirements.RequiresDecomposition {
		result.Reason = "task requirements require decomposition"
	} else {
		result.Reason = "user explicitly requested Mission execution"
	}
	return result, nil
}

// IsExplicitMissionRequest reports whether the user explicitly asked for
// Mission/autonomous execution. Callers use this to distinguish a user-owned
// Mission request from a heuristic promotion that may safely fall back to the
// interactive Lead when admission cannot be satisfied.
func IsExplicitMissionRequest(line string) bool {
	tokens := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(line)), func(r rune) bool {
		// Keep hyphenated product/domain terms together so "mission-critical"
		// is not reinterpreted as an explicit Mission request.
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-'
	})
	if len(tokens) == 0 {
		return false
	}
	isRequestVerb := func(token string) bool {
		switch token {
		case "execute", "executar", "rode", "rodar", "run", "start", "inicie", "iniciar",
			"crie", "criar", "create", "promova", "promover", "promote",
			"transforme", "transformar", "use", "usar", "faça", "faca", "quero", "want":
			return true
		default:
			return false
		}
	}
	isMissionToken := func(token string) bool {
		switch token {
		case "mission", "missions", "missão", "missao", "missões", "missoes":
			return true
		default:
			return false
		}
	}
	for i, token := range tokens {
		explicitTarget := isMissionToken(token) ||
			strings.HasPrefix(token, "autônom") ||
			strings.HasPrefix(token, "autonom")
		if !explicitTarget {
			continue
		}
		// "Mission" / "como Mission" are concise explicit commands.
		if len(tokens) <= 2 {
			return true
		}
		start := i - 5
		if start < 0 {
			start = 0
		}
		for j := start; j < i; j++ {
			if isRequestVerb(tokens[j]) {
				return true
			}
		}
	}
	return false
}
