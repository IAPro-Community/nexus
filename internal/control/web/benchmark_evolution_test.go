package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IAPro-Community/nexus/internal/control/driver"
	"github.com/IAPro-Community/nexus/internal/core/provider/versionprobe"
)

func BenchmarkHandleProvidersCached(b *testing.B) {
	b.Setenv("HOME", b.TempDir())
	b.Setenv("NEXUS_REAL_HOME", "")
	versionprobe.InvalidateCache()

	handler := &APIHandler{drivers: driver.DefaultRegistry()}

	// Warm the binary version cache once, mirroring steady-state desktop/web runtime.
	warmRec := httptest.NewRecorder()
	warmReq := httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil)
	handler.handleProviders(warmRec, warmReq)
	if warmRec.Code != http.StatusOK {
		b.Fatalf("warm handleProviders status=%d", warmRec.Code)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/providers", nil)
		handler.handleProviders(rec, req)
		if rec.Code != http.StatusOK {
			b.Fatalf("handleProviders status=%d", rec.Code)
		}
	}
}
