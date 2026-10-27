package codex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/IAPro-Community/nexus/internal/core/config"
	"github.com/IAPro-Community/nexus/internal/core/model"
	"github.com/IAPro-Community/nexus/internal/core/security"
)

// authSourceFileName stores credential provenance inside the profile root.
const authSourceFileName = "auth-source.json"

// hostAuthFile is the Codex CLI login file owned by the real user home.
func hostAuthFile() string {
	hostHome := security.FindHostHome()
	if hostHome == "" {
		return ""
	}
	return filepath.Join(hostHome, ".codex", "auth.json")
}

// HostAuthAccountID returns the chatgpt_account_id of the host CLI login,
// without exposing the credential. Empty when the host is not logged in.
func HostAuthAccountID() string {
	path := hostAuthFile()
	if path == "" {
		return ""
	}
	return readCodexAuthAccountID(path)
}

// profileAuthPath reports the profile credential, if any, and whether it
// already exists. Both the canonical and the legacy location are accepted.
func profileAuthPath(home string) (string, bool) {
	candidates := []string{
		filepath.Join(home, "auth.json"),
		filepath.Join(home, ".codex", "auth.json"),
	}
	for _, candidate := range candidates {
		if st, err := os.Stat(candidate); err == nil && st.Size() > 0 {
			return candidate, true
		}
	}
	return candidates[0], false
}

// AdoptHostAuth copies the host Codex CLI login into a profile that has no
// credential of its own and records where it came from.
//
// The profile keeps its own CODEX_HOME: quota probes still require the
// app-server to report the same chatgpt_account_id stored in this file, so a
// stale copy is rejected instead of being attributed to the wrong account.
// Existing credentials are never overwritten; switching accounts stays an
// explicit user action.
func AdoptHostAuth(ctx context.Context, p model.Profile) (model.AuthSource, bool) {
	var src model.AuthSource
	if p.Provider != "codex" {
		return src, false
	}
	home, err := config.ProfileHome("codex", p.Name)
	if err != nil || strings.TrimSpace(home) == "" {
		return src, false
	}
	if _, exists := profileAuthPath(home); exists {
		return src, false
	}
	source := hostAuthFile()
	if source == "" {
		return src, false
	}
	data, err := os.ReadFile(source)
	if err != nil || len(data) == 0 {
		return src, false
	}
	var auth hostAuthFileShape
	if err := json.Unmarshal(data, &auth); err != nil {
		return src, false
	}
	// Only a ChatGPT subscription login carries the account-scoped quota. An
	// API-key login would authenticate a different identity and must not be
	// presented as subscription capacity.
	if !strings.EqualFold(strings.TrimSpace(auth.AuthMode), "chatgpt") {
		return src, false
	}
	if strings.TrimSpace(auth.Tokens.AccessToken) == "" || strings.TrimSpace(auth.Tokens.RefreshToken) == "" {
		return src, false
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return src, false
	}
	target := filepath.Join(home, "auth.json")
	if err := os.WriteFile(target, data, 0600); err != nil {
		return src, false
	}
	_ = os.Chmod(target, 0600)

	accountID := strings.TrimSpace(auth.Tokens.AccountID)
	if accountID != "" {
		recordHostAuthObservation(accountID)
	}
	plan := "ChatGPT"
	if info := New().InspectAuth(ctx, p); info.Authenticated && strings.TrimSpace(info.Plan) != "" {
		plan = info.Plan
	}
	src = model.AuthSource{
		Origin:        model.AuthSourceOriginHost,
		LinkedAt:      time.Now().UTC(),
		AccountIDHash: hashAccountID(accountID),
		Plan:          plan,
	}
	writeAuthSource(p.Name, src)
	return src, true
}

type hostAuthFileShape struct {
	AuthMode string `json:"auth_mode"`
	Tokens   struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		AccountID    string `json:"account_id"`
	} `json:"tokens"`
}

func hashAccountID(accountID string) string {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(accountID))
	return hex.EncodeToString(sum[:6])
}

func writeAuthSource(name string, src model.AuthSource) {
	root, err := config.ProfileRoot("codex", name)
	if err != nil || strings.TrimSpace(root) == "" {
		return
	}
	data, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(root, authSourceFileName), append(data, '\n'), 0600)
}
