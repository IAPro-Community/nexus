package web

import "testing"

func TestProjectSubrouteClassification(t *testing.T) {
	cases := map[string]string{
		"/api/v1/projects/proj-1/layout":                    "layout",
		"/api/v1/projects/proj-1/agents":                    "agents",
		"/api/v1/projects/proj-1/missions":                  "missions",
		"/api/v1/projects/proj-1/plans":                     "plans",
		"/api/v1/projects/proj-1/open-os":                   "open-os",
		"/api/v1/projects/proj-1/git/branches":              "git-branches",
		"/api/v1/projects/proj-1/git/checkout":              "git-checkout",
		"/api/v1/projects/proj-1/context-readiness":         "context-readiness",
		"/api/v1/projects/proj-1/context-readiness/hydrate": "context-hydrate",
		"/api/v1/projects/proj-1/context-readiness/refresh": "context-refresh",
		"/api/v1/projects/proj-1":                           "detail",
	}
	for path, want := range cases {
		if got := projectSubroute(path); got != want {
			t.Fatalf("path=%s got=%s want=%s", path, got, want)
		}
	}
}
