package host

import (
	"fmt"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/registry"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/quota"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// SlashResult represents the outcome of processing an input line through the Nexus Router.
type SlashResult struct {
	Intercepted      bool   // True if command was a /nexus command consumed by Nexus Control
	Response         string // Formatted response to display to the user
	ForwardToProcess string // Text to forward to child process (if any, e.g. unescaped //ai text)
	Action           string // Special action e.g. "detach", "stop", "handoff", "continue"
	ActionArg        string // Argument for special action (e.g. target profile)
}

// StripANSI removes ANSI escape sequences and non-printable control codes from input strings.
func StripANSI(str string) string {
	var b strings.Builder
	inEsc := false
	for i := 0; i < len(str); i++ {
		c := str[i]
		if c == 0x1b { // ESC
			inEsc = true
			continue
		}
		if inEsc {
			if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '~' {
				inEsc = false
			}
			continue
		}
		if c >= 32 || c == '\t' {
			b.WriteByte(c)
		}
	}
	return b.String()
}

// RouteSlashCommand is a deprecated compatibility parser for explicit
// CmdSlash control requests. It must never be called for CmdInput terminal
// bytes; the PTY path is intentionally transparent.
func RouteSlashCommand(input string, session registry.RuntimeSession) SlashResult {
	clean := StripANSI(input)
	trimmed := strings.TrimSpace(clean)

	// 1. Check for the Nexus escape prefix ("//nexus", "::nexus")
	if strings.HasPrefix(trimmed, "//nexus") || strings.HasPrefix(trimmed, "::nexus") {
		escaped := "/nexus" + trimmed[7:]
		if strings.HasSuffix(input, "\n") {
			escaped += "\n"
		}
		return SlashResult{
			Intercepted:      false,
			ForwardToProcess: escaped,
		}
	}
	// 2. Check for the reserved Nexus control prefix ("/nexus", ":nexus")
	isNexus := strings.HasPrefix(trimmed, "/nexus") || strings.HasPrefix(trimmed, ":nexus")
	if !isNexus {
		// Normal input: forward untouched
		return SlashResult{
			Intercepted:      false,
			ForwardToProcess: input,
		}
	}

	prefix := "/nexus"
	// Use the actual prefix from input for display consistency
	if strings.HasPrefix(trimmed, ":nexus") {
		prefix = ":nexus"
	}

	// It is a slash command: intercept completely
	parts := strings.Fields(trimmed)
	subcmd := ""
	if len(parts) > 1 {
		subcmd = parts[1]
	}

	switch subcmd {
	case "help", "?", "":
		resp := fmt.Sprintf(`
╔══════════════════════════════════════════════════════════════════╗
║             NEXUS CONTROL — UNIVERSAL SLASH COMMANDS            ║
╠══════════════════════════════════════════════════════════════════╣
║  %s status               Show current runtime status & quota   ║
║  %s accounts             List available accounts & quotas      ║
║  %s usage                Display honest usage metrics snapshot ║
║  %s handoff <profile>    Same-provider account handoff         ║
║  %s continue <provider>  Cross-provider context handoff        ║
║  %s control              Open Control Center TUI instructions  ║
║  %s detach               Detach from session (keeps alive)     ║
║  %s stop                 Gracefully terminate runtime session  ║
║  /%s <text>              Escape prefix to send literal command ║
╚══════════════════════════════════════════════════════════════════╝
`, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix, prefix)
		return SlashResult{Intercepted: true, Response: resp}

	case "status":
		qv := profile.GetQuotaView(session.ProviderID, session.ProfileID, "", "")
		usageStr := formatQuotaViewSummary(qv)
		availLabel := qv.AvailabilityLabel()
		if !qv.IsAvailable() {
			availLabel = "⚠ " + availLabel
		}

		resp := fmt.Sprintf(`
┌─ NEXUS CONTROL STATUS ───────────────────────────────────────┐
│  Runtime ID:       %-41s │
│  Provider:         %-41s │
│  Profile:          %-41s │
│  Provider Session: %-41s │
│  State:            %-41s │
│  Control Level:    %-41s │
│  Workspace:        %-41s │
│  Usage / Quota:    %-41s │
│  Disponibilidade:  %-41s │
└──────────────────────────────────────────────────────────────┘
`, session.RuntimeID, strings.ToUpper(session.ProviderID), session.ProfileID,
			fallbackUnknown(session.ProviderSessionID), session.State, session.ControlLevel,
			truncateStr(session.Workspace, 41), usageStr, availLabel)
		return SlashResult{Intercepted: true, Response: resp}

	case "accounts":
		profs, _ := profile.List()
		var lines []string
		lines = append(lines, fmt.Sprintf("=== Accounts & Quotas for Provider: %s ===", strings.ToUpper(session.ProviderID)))
		lines = append(lines, fmt.Sprintf("%-16s %-12s %-14s %-14s %-14s %s", "PROFILE", "STATUS", "CAPACITY", "AVAIL", "FRESHNESS", "RESET"))
		lines = append(lines, strings.Repeat("─", 80))

		found := 0
		allUnavailable := true
		for _, p := range profs {
			if p.Provider == session.ProviderID {
				found++
				activeMarker := "  "
				if p.Name == session.ProfileID {
					activeMarker = "> "
				}
				info := profile.GetAccountInfo(p.Provider, p.Name)
				qv := profile.GetQuotaView(p.Provider, p.Name, "", "")

				authStatus := "AUTH_OK"
				if !info.Authenticated {
					authStatus = "NO_AUTH"
				}

				capStr := "UNKNOWN"
				if summary := qv.CompactGroupSummary(); summary != "" && (qv.Status == string(model.UsageLive) || qv.Status == string(model.UsageCached) || qv.Status == string(model.UsageEstimated)) {
					capStr = summary
				} else if qv.Status == string(model.UsageLive) || qv.Status == string(model.UsageCached) || qv.Status == string(model.UsageEstimated) {
					if rem, ok := qv.BestGroupRemaining(); ok {
						capStr = fmt.Sprintf("%.0f%% left", rem)
					}
				}
				if qv.Status == string(model.UsageRateLimited) {
					capStr = "429 LIMITED"
				}

				availLabel := qv.AvailabilityLabel()
				if qv.IsAvailable() {
					allUnavailable = false
				}

				freshStr := qv.Status
				if !qv.FetchedAt.IsZero() {
					freshStr = quota.FormatFreshness(qv.FetchedAt)
				}
				resetStr := "-"
				for _, g := range qv.ModelGroups {
					for _, w := range g.Windows {
						if w.ResetDesc != "" {
							resetStr = w.ResetDesc
							break
						}
					}
					if resetStr != "-" {
						break
					}
				}

				lines = append(lines, fmt.Sprintf("%s%-14s %-12s %-14s %-14s %-14s %s",
					activeMarker, p.Name, authStatus, capStr, availLabel, freshStr, resetStr,
				))
			}
		}
		if found == 0 {
			lines = append(lines, "  (No profiles configured for this provider)")
		}
		if allUnavailable && found > 0 {
			lines = append(lines, "")
			lines = append(lines, fmt.Sprintf("⚠  ALERTA: Nenhuma conta disponível para %s!", strings.ToUpper(session.ProviderID)))
			lines = append(lines, "   Aguarde o reset da quota ou use outro provider.")
		}
		return SlashResult{Intercepted: true, Response: strings.Join(lines, "\n") + "\n"}

	case "usage":
		qv := profile.GetQuotaView(session.ProviderID, session.ProfileID, "", "")
		var sb strings.Builder
		fmt.Fprintf(&sb, "\n=== Quota Snapshot: %s:%s ===\n", strings.ToUpper(session.ProviderID), session.ProfileID)
		fmt.Fprintf(&sb, "Status:       %s\n", qv.Status)
		fmt.Fprintf(&sb, "Source:       %s\n", qv.Source)
		if len(qv.ModelGroups) > 0 {
			for _, group := range qv.ModelGroups {
				if qv.HasMultipleGroups() && group.Name != "" {
					fmt.Fprintf(&sb, "\n  %s:\n", group.Name)
				}
				for _, w := range group.Windows {
					fmt.Fprintf(&sb, "  Window [%s]:  Remaining: %.1f%% | Reset: %s\n", w.Kind, w.Remaining, w.ResetDesc)
				}
			}
		} else {
			sb.WriteString("Remaining:    UNKNOWN (No authentic quota metric exposed)\n")
		}
		if !qv.FetchedAt.IsZero() {
			fmt.Fprintf(&sb, "Last Check:   %s (%s ago)\n", qv.FetchedAt.Format(time.RFC3339), time.Since(qv.FetchedAt).Round(time.Second))
		}
		return SlashResult{Intercepted: true, Response: sb.String() + "\n"}

	case "detach":
		return SlashResult{
			Intercepted: true,
			Action:      "detach",
			Response:    "\n[Nexus Control] Detached from runtime session. Host process remains running in background.\n",
		}

	case "stop":
		return SlashResult{
			Intercepted: true,
			Action:      "stop",
			Response:    "\n[Nexus Control] Stopping runtime session...\n",
		}

	case "handoff":
		if len(parts) < 3 {
			return SlashResult{
				Intercepted: true,
				Response:    fmt.Sprintf("Usage: %s handoff <target-profile> (e.g. %s handoff codex:work or %s handoff work)\n", prefix, prefix, prefix),
			}
		}
		target := parts[2]
		return SlashResult{
			Intercepted: true,
			Action:      "handoff",
			ActionArg:   target,
			Response:    fmt.Sprintf("\n[Nexus Control] Initiating same-provider account handoff to %s...\n", target),
		}

	case "continue":
		if len(parts) < 3 {
			return SlashResult{
				Intercepted: true,
				Response:    fmt.Sprintf("Usage: %s continue <target-provider[:profile]> (e.g. %s continue claude or %s continue claude:work)\n", prefix, prefix, prefix),
			}
		}
		target := parts[2]
		return SlashResult{
			Intercepted: true,
			Action:      "continue",
			ActionArg:   target,
			Response:    fmt.Sprintf("\n[Nexus Control] Creating cross-provider context envelope for %s...\n", target),
		}

	case "control", "ui":
		return SlashResult{
			Intercepted: true,
			Response:    fmt.Sprintf("\n[Nexus Control] To open the full Workspace OS, detach (%s detach) and run: nexus\n", prefix),
		}

	default:
		return SlashResult{
			Intercepted: true,
			Response:    fmt.Sprintf("Unknown command %q. Type %s help for available commands.\n", subcmd, prefix),
		}
	}
}

func fallbackUnknown(s string) string {
	if strings.TrimSpace(s) == "" {
		return "UNKNOWN"
	}
	return s
}

func truncateStr(s string, max int) string {
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}

// formatQuotaViewSummary returns a compact one-line summary from a QuotaView.
func formatQuotaViewSummary(qv quota.QuotaView) string {
	status := qv.Status
	if status == "" {
		status = "UNKNOWN"
	}
	if summary := qv.CompactGroupSummary(); summary != "" {
		fresh := ""
		if !qv.FetchedAt.IsZero() {
			fresh = " · " + quota.FormatFreshness(qv.FetchedAt)
		}
		return fmt.Sprintf("%s (%s)%s", summary, status, fresh)
	}
	bottleneck, _ := qv.Bottleneck()
	return fmt.Sprintf("%.0f%% remaining (%s)", bottleneck, status)
}
