package tui

import (
	"os"

	"github.com/IAPro-Community/nexus/internal/conversation"
	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/profile"
)

// ShowMenu launches the unified interactive Quotas & Runtimes TUI.
// This bridges legacy invocations into the modern unified usage interface.
func ShowMenu() (*SelectionResult, error) {
	cfg, _ := config.LoadConfig()
	profs, _ := profile.List()

	accs := make(map[string]model.AccountInfo)
	rows := make([]UsageTableRow, 0, len(profs))

	for _, p := range profs {
		acc := profile.GetAccountInfo(p.Provider, p.Name)
		accs[p.Provider+":"+p.Name] = acc
		qv := profile.GetQuotaView(p.Provider, p.Name, acc.Plan, acc.Email)
		rows = append(rows, buildUsageRows(p.Provider, p.Name, qv, acc, cfg.Defaults)...)
	}

	cwd, _ := os.Getwd()
	convs := conversation.ListRecent(30, cwd)

	opts := UnifiedUsageOptions{
		Rows:        rows,
		Sessions:    convs,
		Accounts:    accs,
		Defaults:    cfg.Defaults,
		Workspace:   cwd,
		InitialMode: ModeSafe,
	}

	return RunUnifiedUsage(opts)
}
