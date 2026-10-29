package app

import (
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestOMPPreviewFailsClosedWithoutOptIn(t *testing.T) {
	t.Setenv("NEXUS_EXPERIMENTAL_HARNESSES", "")
	if err := executeOMPPreview("default", nil); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("OMP preview must require explicit opt-in: %v", err)
	}
}

func TestOMPPreviewRejectsUncertifiedLaunchModes(t *testing.T) {
	t.Setenv("NEXUS_EXPERIMENTAL_HARNESSES", "1")
	for _, arg := range []string{"--direct", "--lead", "--yolo", "-y"} {
		if err := executeOMPPreview("default", []string{arg}); err == nil || !strings.Contains(err.Error(), "supervised PTY only") {
			t.Errorf("OMP mode %s must fail closed: %v", arg, err)
		}
	}
}

func TestSelectOMPPreviewProfile(t *testing.T) {
	profiles := []model.Profile{
		{Provider: "omp", Name: "first"},
		{Provider: "omp", Name: "disabled", Disabled: true},
		{Provider: "codex", Name: "other"},
	}
	cases := []struct {
		requested, defaultProfile, want string
		errorContains                   string
	}{
		{want: "first"},
		{requested: "auto", want: "first"},
		{requested: "first", want: "first"},
		{requested: "disabled", errorContains: "disabled"},
		{requested: "missing", errorContains: "missing"},
		{defaultProfile: "first", want: "first"},
	}
	for _, tc := range cases {
		got, err := selectOMPPreviewProfile(profiles, tc.requested, tc.defaultProfile)
		if tc.errorContains != "" {
			if err == nil || !strings.Contains(err.Error(), tc.errorContains) {
				t.Errorf("select(%q,%q): expected %q error, got %v", tc.requested, tc.defaultProfile, tc.errorContains, err)
			}
		} else if err != nil || got != tc.want {
			t.Errorf("select(%q,%q): got %q, %v; want %q", tc.requested, tc.defaultProfile, got, err, tc.want)
		}
	}
	multiple := append(profiles, model.Profile{Provider: "omp", Name: "second"})
	if _, err := selectOMPPreviewProfile(multiple, "", ""); err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("ambiguous OMP profiles must fail closed: %v", err)
	}
	if _, err := selectOMPPreviewProfile(nil, "", ""); err == nil || !strings.Contains(err.Error(), "no enabled") {
		t.Fatalf("empty OMP profiles must fail closed: %v", err)
	}
}
