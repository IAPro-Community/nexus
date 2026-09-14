package handoff

import "github.com/IAPro-Community/nexus/internal/control/host"

func init() {
	host.PerformAccountHandoff = PerformAccountHandoff
	host.PerformContextHandoff = PerformContextHandoff
}
