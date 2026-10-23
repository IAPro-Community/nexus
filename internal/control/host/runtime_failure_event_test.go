package host

import (
	"errors"
	"strings"
	"testing"

	"github.com/IAPro-Community/nexus/internal/core/model"
)

func TestRuntimeFailureEventMetadataClassifiesWithoutLeakingOutput(t *testing.T) {
	tests := []struct {
		name       string
		output     string
		wantKind   model.FailureKind
		wantRetry  string
	}{
		{name: "rate limit", output: "HTTP 429 Too Many Requests. Try again in 37 minutes.", wantKind: model.FailureRateLimit, wantRetry: "37m0s"},
		{name: "quota exhausted", output: "Insufficient credits", wantKind: model.FailureQuota},
		{name: "authentication", output: "HTTP 401 Unauthorized", wantKind: model.FailureAuth},
		{name: "network", output: "connection refused", wantKind: model.FailureNetwork},
		{name: "unknown", output: "unexpected failure", wantKind: model.FailureUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const marker = "secret-marker-never-publish"
			got := runtimeFailureEventMetadata(errors.New("process exited"), []byte(tt.output+" "+marker))
			if got["failure_kind"] != string(tt.wantKind) {
				t.Fatalf("failure kind = %v, want %q", got["failure_kind"], tt.wantKind)
			}
			if value := got["retry_after"]; value != nil && value != tt.wantRetry {
				t.Fatalf("retry_after = %v, want %q", value, tt.wantRetry)
			}
			if tt.wantRetry != "" && got["retry_after"] != tt.wantRetry {
				t.Fatalf("retry_after = %v, want %q", got["retry_after"], tt.wantRetry)
			}
			for _, value := range got {
				if str, ok := value.(string); ok && strings.Contains(str, marker) {
					t.Fatalf("raw terminal output escaped into runtime event: %q", str)
				}
			}
		})
	}
}

func TestRuntimeFailureEventMetadataIgnoresSuccessfulExit(t *testing.T) {
	if got := runtimeFailureEventMetadata(nil, []byte("HTTP 429 Too Many Requests")); got != nil {
		t.Fatalf("successful exit must not publish failure metadata: %+v", got)
	}
}
