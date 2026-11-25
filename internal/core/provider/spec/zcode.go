package spec

import (
	"path/filepath"

	sdkprovider "github.com/IAPro-Community/IAPro-SDK/providers/spec"
	"github.com/IAPro-Community/nexus/internal/runtime"
)

var zcodeSpec = Spec{Spec: sdkprovider.Spec{
	ClearLine:        "\x15",
	ID:               "zcode",
	DisplayName:      "Z Code",
	ShortBadge:       "ZCD",
	AuthKind:         sdkprovider.AuthKindAPIKey,
	BinaryCandidates: []string{"zcode"},
	EnvKeys: func(profileHome, binDir, profileName string) map[string]string {
		return map[string]string{
			"HOME":            profileHome,
			"USERPROFILE":     profileHome,
			"ZCODE_HOME":      profileHome,
			"XDG_CONFIG_HOME": filepath.Join(profileHome, ".config"),
			"AI_PROFILE":      profileName,
			"AI_PROVIDER":     "zcode",
			"PATH":            runtime.EnhancedPATH(binDir),
		}
	},
	Kickoff: func(prompt string) []string {
		return []string{"-p", prompt}
	},
	Resume: func(sessionID string) []string {
		return []string{"--resume", sessionID}
	},
	AutoApproval:  []string{"--mode", "yolo"},
	ReviewMode:    []string{"--mode", "guarded"},
	Quota:         QuotaUnsupported,
	KnownVersions: []string{"3.17.2-44"},
	HistorySyncPolicy: sdkprovider.HistorySyncPolicy{
		Mode: sdkprovider.HistorySyncNone,
	},
}}

func init() {
	_ = sdkprovider.Register(zcodeSpec.Spec)
}
