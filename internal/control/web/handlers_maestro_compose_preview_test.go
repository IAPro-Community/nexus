package web

import (
	"strings"
	"testing"
)

func TestMaestroComposeRequestRequiresConsentAndBoundedFields(t *testing.T) {
	for _, raw := range []string{
		`{}`,
		`{"artifact_id":"artifact-1"}`,
		`{"artifact_id":"artifact-1","approved_model_use":false}`,
		`{"artifact_id":"artifact-1","approved_model_use":true,"workspace":"/other/project"}`,
		`{"artifact_id":"artifact-1","approved_model_use":true}{}`,
		`{"artifact_id":"artifact-1","approved_model_use":"true"}`,
	} {
		if _, err := parseMaestroComposeHTTPBody(strings.NewReader(raw)); err == nil {
			t.Fatalf("unsafe compose request accepted: %s", raw)
		}
	}
	body, err := parseMaestroComposeHTTPBody(strings.NewReader(
		`{"artifact_id":"artifact-1","model":"default","approved_model_use":true}`))
	if err != nil || body.ArtifactID != "artifact-1" || body.Model != "default" || !body.ApprovedModelUse {
		t.Fatalf("valid compose request rejected: %+v %v", body, err)
	}
}
