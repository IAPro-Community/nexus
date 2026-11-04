package nexus

import "testing"

func TestValidateSingleTurnExecutionMode(t *testing.T) {
	cases := []struct {
		name     string
		readOnly bool
		mode     string
		wantMode string
		wantErr  bool
	}{
		{name: "missing mode defaults to safe for writes", mode: "", wantMode: "SAFE", wantErr: true},
		{name: "safe blocks writes", mode: "safe", wantMode: "SAFE", wantErr: true},
		{name: "plan blocks writes", mode: "PLAN", wantMode: "PLAN", wantErr: true},
		{name: "yolo allows writes", mode: "yolo", wantMode: "YOLO"},
		{name: "safe allows read only review", readOnly: true, mode: "SAFE", wantMode: "SAFE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := validateSingleTurnExecutionMode(tc.readOnly, tc.mode)
			if got != tc.wantMode {
				t.Fatalf("normalized mode=%q, want %q", got, tc.wantMode)
			}
			if (err != nil) != tc.wantErr {
				t.Fatalf("error=%v, wantErr=%t", err, tc.wantErr)
			}
		})
	}
}
