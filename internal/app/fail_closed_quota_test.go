package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

// staleQuotaSelector documents intent: stubSelector with a stale-quota error.
// The alias is intentionally inlined at call sites below.

func TestResolveLaunchProfileFailsClosedOnUnverifiedQuota(t *testing.T) {
	previousArgs := os.Args
	os.Args = []string{"nexus", "codex"}
	t.Cleanup(func() { os.Args = previousArgs })

	sel := &stubSelector{err: errors.New("no usable codex profiles available: profile desatualizada (UNKNOWN); inelegível para alocação automática")}
	accounts := map[string]model.AccountInfo{
		"default": {Authenticated: true, Health: model.HealthHealthy},
	}
	candidates := []model.Profile{{Name: "default", Provider: "codex"}}
	var stderr bytes.Buffer

	selected, err := resolveLaunchProfile(
		context.Background(), "codex", "", "/tmp", "default", candidates, accounts, sel, nil, &stderr,
	)
	if err == nil {
		t.Fatalf("unverified quota must fail closed; selected=%q stderr=%q", selected, stderr.String())
	}
	if !strings.Contains(stderr.String(), "fail-closed") {
		t.Fatalf("expected fail-closed notice, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "--accept-unverified-quota") {
		t.Fatalf("expected explicit override hint, got %q", stderr.String())
	}
}

func TestResolveLaunchProfileOverrideAcceptsFallback(t *testing.T) {
	previousArgs := os.Args
	os.Args = []string{"nexus", "codex", "--accept-unverified-quota"}
	t.Cleanup(func() { os.Args = previousArgs })

	sel := &stubSelector{err: errors.New("no usable codex profiles available: profile desatualizada (UNKNOWN); inelegível para alocação automática")}
	accounts := map[string]model.AccountInfo{
		"default": {Authenticated: true, Health: model.HealthHealthy},
	}
	candidates := []model.Profile{{Name: "default", Provider: "codex"}}
	var stderr bytes.Buffer

	selected, err := resolveLaunchProfile(
		context.Background(), "codex", "", "/tmp", "default", candidates, accounts, sel, nil, &stderr,
	)
	if err != nil {
		t.Fatalf("explicit override should launch fallback: %v", err)
	}
	if selected != "default" {
		t.Fatalf("selected=%q want default via explicit override", selected)
	}
}

func TestResolveLaunchProfileExplicitTargetNeverFailsClosed(t *testing.T) {
	// Explicit profiles bypass quota ranking entirely (existing policy): the
	// fail-closed gate must not make explicit choices impossible.
	sel := &stubSelector{err: errors.New("selector should not be consulted")}
	accounts := map[string]model.AccountInfo{}
	candidates := []model.Profile{{Name: "mine", Provider: "codex"}}

	selected, err := resolveLaunchProfile(
		context.Background(), "codex", "mine", "/tmp", "", candidates, accounts, sel, nil, nil,
	)
	if err != nil {
		t.Fatalf("explicit profile launch: %v", err)
	}
	if selected != "mine" {
		t.Fatalf("selected=%q want mine", selected)
	}
}
