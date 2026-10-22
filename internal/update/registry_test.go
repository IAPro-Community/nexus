package update

import (
	"strings"
	"testing"
)

func TestChannelPointerTag(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"stable": "channel-stable",
		"STABLE": "channel-stable",
		"":       "channel-stable",
		"beta":   "channel-beta",
		"Beta":   "channel-beta",
	}
	for in, want := range cases {
		if got := ChannelPointerTag(in); got != want {
			t.Fatalf("ChannelPointerTag(%q)=%q want %q", in, got, want)
		}
	}
}

func TestDefaultManifestURL(t *testing.T) {
	t.Parallel()
	got := DefaultManifestURL("IAPro-Community/nexus", "stable")
	want := "https://github.com/IAPro-Community/nexus/releases/download/channel-stable/update-manifest.json"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	gotBeta := DefaultManifestURL("IAPro-Community/nexus", "beta")
	wantBeta := "https://github.com/IAPro-Community/nexus/releases/download/channel-beta/update-manifest.json"
	if gotBeta != wantBeta {
		t.Fatalf("got %q want %q", gotBeta, wantBeta)
	}
	if DefaultManifestURL("", "stable") != DefaultUpdateRegistryURL {
		t.Fatalf("empty repo must fall back to default %q", DefaultUpdateRegistryRepo)
	}
}

func TestResolveRegistryURLPrecedence(t *testing.T) {
	t.Setenv("NEXUS_UPDATE_REGISTRY_URL", "")
	t.Setenv("NEXUS_RELEASE_REPO", "")

	if got := ResolveRegistryURL("", "stable"); got != DefaultUpdateRegistryURL {
		t.Fatalf("default stable=%q want %q", got, DefaultUpdateRegistryURL)
	}
	if !strings.Contains(ResolveRegistryURL("", "beta"), "channel-beta") {
		t.Fatalf("beta channel URL missing channel-beta: %s", ResolveRegistryURL("", "beta"))
	}

	t.Setenv("NEXUS_RELEASE_REPO", "acme/fork")
	got := ResolveRegistryURL("", "stable")
	want := "https://github.com/acme/fork/releases/download/channel-stable/update-manifest.json"
	if got != want {
		t.Fatalf("repo override got %q want %q", got, want)
	}

	t.Setenv("NEXUS_UPDATE_REGISTRY_URL", "https://example.invalid/manifest.json")
	if got := ResolveRegistryURL("", "stable"); got != "https://example.invalid/manifest.json" {
		t.Fatalf("full URL override got %q", got)
	}

	if got := ResolveRegistryURL("https://explicit.invalid/m.json", "stable"); got != "https://explicit.invalid/m.json" {
		t.Fatalf("explicit arg must win, got %q", got)
	}
}

func TestDefaultUpdateRegistryURLIsIAPROStable(t *testing.T) {
	t.Parallel()
	want := "https://github.com/IAPro-Community/nexus/releases/download/channel-stable/update-manifest.json"
	if DefaultUpdateRegistryURL != want {
		t.Fatalf("DefaultUpdateRegistryURL=%q want %q", DefaultUpdateRegistryURL, want)
	}
}

func TestDefaultRegistryPointsAtSourceRepo(t *testing.T) {
	t.Parallel()
	if DefaultUpdateRegistryRepo != "IAPro-Community/nexus" {
		t.Fatalf("default registry must match the source repo, got %q", DefaultUpdateRegistryRepo)
	}
}
