package provider

import (
	"context"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

type installationTestProvider struct{}

func (installationTestProvider) ID() model.ProviderID             { return "test" }
func (installationTestProvider) Name() string                     { return "Test" }
func (installationTestProvider) Capabilities() model.Capabilities { return model.Capabilities{} }
func (installationTestProvider) Detect(context.Context) model.DetectionResult {
	return model.DetectionResult{Installed: true, Version: "1.2.3", BinaryPath: "/tmp/test-cli"}
}
func (installationTestProvider) Prepare(context.Context, model.Profile) error { return nil }
func (installationTestProvider) Run(context.Context, model.Profile, []string) (model.Failure, error) {
	return model.Failure{}, nil
}

func TestInstallationRegistrySeparatesDiscoveryFromRegistration(t *testing.T) {
	t.Setenv("NEXUS_DATA_DIR", t.TempDir())
	providers := NewRegistry()
	if err := providers.Register(installationTestProvider{}); err != nil {
		t.Fatal(err)
	}
	registry := NewInstallationRegistry()
	records, err := registry.Refresh(providers, context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var found InstallationRecord
	for _, record := range records {
		if record.Provider == "test" {
			found = record
			break
		}
	}
	if found.State != InstalledUnregistered {
		t.Fatalf("unexpected installation state: %+v", records)
	}
	if found.Path != "/tmp/test-cli" || found.Version != "1.2.3" {
		t.Fatalf("discovery metadata lost: %+v", found)
	}
}
