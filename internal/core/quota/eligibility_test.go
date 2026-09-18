package quota

import "testing"

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
			name: "unsupported auto",
			in: EligibilityInput{
				View:          QuotaView{Status: "UNSUPPORTED", Available: true},
				Authenticated: true,
				Health:        "healthy",
			},
			want: ReasonUnsupportedQuota,
			ok:   false,
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
		})
	}
}
