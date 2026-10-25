package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/core/model"
)

type delayedProviderDetectionDriver struct {
	*driver.FakeDriver
	id    string
	delay time.Duration
}

func (d *delayedProviderDetectionDriver) ProviderID() string { return d.id }

func (d *delayedProviderDetectionDriver) Detect(ctx context.Context) (model.DetectionResult, error) {
	timer := time.NewTimer(d.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return model.DetectionResult{}, ctx.Err()
	case <-timer.C:
		return model.DetectionResult{Installed: true, Version: "fixture"}, nil
	}
}

func TestHandleProvidersAllowsSlowLocalCLIProbeToComplete(t *testing.T) {
	drivers := driver.NewRegistry()
	drivers.Register(&delayedProviderDetectionDriver{
		FakeDriver: driver.NewFakeDriver(),
		id:         "gemini",
		delay:      2500 * time.Millisecond,
	})
	handler := &APIHandler{drivers: drivers}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil)
	handler.handleProviders(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("provider endpoint returned %d", recorder.Code)
	}
	var providers []struct {
		ID        string `json:"id"`
		Installed bool   `json:"installed"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &providers); err != nil {
		t.Fatalf("decode providers: %v", err)
	}
	if len(providers) != 1 || providers[0].ID != "gemini" || !providers[0].Installed {
		t.Fatalf("slow installed provider was reported missing: %+v", providers)
	}
}
