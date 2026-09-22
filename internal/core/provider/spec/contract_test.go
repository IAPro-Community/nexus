package spec

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

var updateGoldens = flag.Bool("update", false, "update provider contract goldens")

type contractSnapshot struct {
	ID               string      `json:"id"`
	BinaryCandidates []string    `json:"binary_candidates"`
	EnvKeys          []string    `json:"env_keys"`
	EnvDelete        []string    `json:"env_delete"`
	Kickoff          []string    `json:"kickoff"`
	Resume           []string    `json:"resume"`
	AutoApproval     []string    `json:"auto_approval"`
	ReviewMode          []string           `json:"review_mode"`
	AutonomousIsolation IsolationGuarantee `json:"autonomous_isolation"`
	ReviewIsolation     IsolationGuarantee `json:"review_isolation"`
	Quota               QuotaSource        `json:"quota"`
	KnownVersions    []string    `json:"known_versions"`
}

func TestContractGoldens(t *testing.T) {
	update := *updateGoldens || os.Getenv("UPDATE_GOLDENS") == "1"
	for _, s := range All() {
		t.Run(s.ID, func(t *testing.T) {
			got := snapshotOf(s)
			path := filepath.Join("testdata", s.ID, "contract.golden")
			if update {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				raw, err := json.MarshalIndent(got, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				raw = append(raw, '\n')
				if err := os.WriteFile(path, raw, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden: %v (run with -update or UPDATE_GOLDENS=1)", err)
			}
			var want contractSnapshot
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				gotRaw, _ := json.MarshalIndent(got, "", "  ")
				t.Fatalf("contract drift for %s\ngot:\n%s\nwant file: %s", s.ID, gotRaw, path)
			}
		})
	}
}

func snapshotOf(s Spec) contractSnapshot {
	env := s.EnvKeys("{home}", "{binDir}", "{profile}")
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	deleteKeys := append([]string(nil), s.EnvDelete...)
	sort.Strings(deleteKeys)
	kickoff := []string{}
	if s.Kickoff != nil {
		kickoff = s.Kickoff("{prompt}")
	}
	resume := []string{}
	if s.Resume != nil {
		resume = s.Resume("{session}")
	}
	return contractSnapshot{
		ID:               s.ID,
		BinaryCandidates: append([]string(nil), s.BinaryCandidates...),
		EnvKeys:          keys,
		EnvDelete:        deleteKeys,
		Kickoff:          kickoff,
		Resume:           resume,
		AutoApproval:     append([]string(nil), s.AutoApproval...),
		ReviewMode:          append([]string(nil), s.ReviewMode...),
		AutonomousIsolation: s.AutonomousIsolationLevel(),
		ReviewIsolation:     s.ReviewIsolationLevel(),
		Quota:               s.Quota,
		KnownVersions:    append([]string(nil), s.KnownVersions...),
	}
}
