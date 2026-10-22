package quota

import (
	"testing"
	"time"
)

func TestEvaluateEligibilityGates(t *testing.T) {
	live := QuotaView{Status: "LIVE", Available: true}

	cases := []struct {
		name string
		in   EligibilityInput
		want EligibilityReason
		ok   bool
	}{
		{
			name: "live ok",
			in:   EligibilityInput{View: live, Authenticated: true, Health: "healthy"},
			want: ReasonOK,
			ok:   true,
		},
		{
			name: "outdated estimated capacity cannot steer automatic routing",
			in: EligibilityInput{
				View:          QuotaView{Status: "ESTIMATED", Available: true, FetchedAt: time.Now().Add(-25 * 24 * time.Hour)},
				Authenticated: true,
				Health:        "healthy",
			},
			want: ReasonUnknownQuota,
			ok:   false,
		},
		{
			name: "outdated official rate limit is no longer a current block",
			in: EligibilityInput{
				View:          QuotaView{Status: "RATE_LIMITED", Available: false, FetchedAt: time.Now().Add(-25 * 24 * time.Hour), AvailReasons: AvailReasons{RateLimited: true}},
				Authenticated: true,
				Health:        "healthy",
			},
			want: ReasonUnknownQuota,
			ok:   false,
		},
		{
			name: "fresh estimate still eligible when all other gates pass",
			in: EligibilityInput{
				View:          QuotaView{Status: "ESTIMATED", Available: true, FetchedAt: time.Now().Add(-1 * time.Minute)},
				Authenticated: true,
				Health:        "healthy",
			},
			want: ReasonOK,
			ok:   true,
		},
		{
			name: "unknown auto",
			in: EligibilityInput{
				View:          QuotaView{Status: "UNKNOWN", Available: false, AvailReasons: AvailReasons{UnknownQuota: true}},
				Authenticated: true,
				Health:        "healthy",
			},
			want: ReasonUnknownQuota,
			ok:   false,
		},
		{
			name: "unknown explicit available",
			in: EligibilityInput{
				View:              QuotaView{Status: "UNKNOWN", Available: true, AvailReasons: AvailReasons{UnknownQuota: true}},
				Authenticated:     true,
				Health:            "healthy",
				ExplicitSelection: true,
			},
			want: ReasonUnknownQuota,
			ok:   true,
		},
		{
			name: "exhausted unavailable",
			in: EligibilityInput{
				View: QuotaView{
					Status:       "LIVE",
					Available:    false,
					AvailReasons: AvailReasons{ExhaustedWindows: []string{"weekly", "claude_weekly"}},
				},
				Authenticated: true,
				Health:        "healthy",
			},
			want: ReasonExhausted,
			ok:   false,
		},
		{
			name: "partial exhaustion remains eligible",
			in: EligibilityInput{
				View: QuotaView{
					Status:       "CACHED",
					Available:    true,
					AvailReasons: AvailReasons{ExhaustedWindows: []string{"5h"}},
				},
				Authenticated: true,
				Health:        "healthy",
			},
			want: ReasonOK,
			ok:   true,
		},
		{
			name: "unsupported auto remains operationally eligible",
			in: EligibilityInput{
				View:          QuotaView{Status: "UNSUPPORTED", Available: false, AvailReasons: AvailReasons{UnknownQuota: true}},
				Authenticated: true,
				Health:        "healthy",
			},
			want: ReasonUnsupportedQuota,
			ok:   true,
		},
		{
			name: "unavailable",
			in:   EligibilityInput{View: live, Authenticated: true, Health: "unavailable"},
			want: ReasonUnavailable,
			ok:   false,
		},
		{
			name: "rate limited",
			in: EligibilityInput{
				View:          QuotaView{Status: "RATE_LIMITED", Available: false, AvailReasons: AvailReasons{RateLimited: true}},
				Authenticated: true,
				Health:        "healthy",
			},
			want: ReasonRateLimited,
			ok:   false,
		},
		{
			name: "degraded still eligible",
			in:   EligibilityInput{View: live, Authenticated: true, Health: "degraded"},
			want: ReasonOK,
			ok:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateEligibility(tc.in)
			if got.Reason != tc.want || got.Eligible != tc.ok {
				t.Fatalf("got eligible=%v reason=%s want eligible=%v reason=%s", got.Eligible, got.Reason, tc.ok, tc.want)
			}
			if tc.name == "unsupported auto remains operationally eligible" && got.CapacityEvidence != "OPAQUE" {
				t.Fatalf("unsupported capacity evidence=%q want OPAQUE", got.CapacityEvidence)
			}
			if tc.name == "unknown auto" && got.CapacityEvidence != "UNKNOWN" {
				t.Fatalf("unknown capacity evidence=%q want UNKNOWN", got.CapacityEvidence)
			}
		})
	}
}
