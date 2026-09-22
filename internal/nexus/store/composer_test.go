package store

import (
	"strings"
	"path/filepath"
	"testing"
)

func TestComposerSessionPersistsBriefTurnsSkillsAndPromptArtifacts(t *testing.T) {
	st, err := Open(filepath.Join(t.TempDir(), "composer.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	project, err := st.CreateProject(Project{Name: "Composer", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	session, err := st.CreateComposerSession(ComposerSession{ProjectID: project.ID, State: ComposerExploring, Title: "Test strategy", BriefJSON: `{"goal":"design test strategy"}`})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendComposerTurn(ComposerTurn{SessionID: session.ID, Role: ComposerUser, Content: "Help me define test coverage"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertComposerSkillProposal(ComposerSkillProposal{SessionID: session.ID, SkillID: "skill-webapp-testing", State: ComposerSkillSuggested, Reason: "Browser regression coverage"}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.UpsertComposerSkillProposal(ComposerSkillProposal{SessionID: session.ID, SkillID: "skill-legacy", State: "SELECTED", Reason: "Legacy persisted state"}); err != nil {
		t.Fatal(err)
	}
	artifact, err := st.CreatePromptArtifact(PromptArtifact{SessionID: session.ID, Content: "Implement the approved test strategy.", ContextJSON: `{}`, SkillIDsJSON: `["skill-webapp-testing"]`})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := st.GetComposerSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.BriefJSON != session.BriefJSON || loaded.ProjectID != project.ID {
		t.Fatalf("unexpected loaded session: %+v", loaded)
	}
	turns, err := st.ListComposerTurns(session.ID, 40)
	if err != nil || len(turns) != 1 || turns[0].Content != "Help me define test coverage" {
		t.Fatalf("unexpected turns: %+v, %v", turns, err)
	}
	skills, err := st.ListComposerSkillProposals(session.ID)
	if err != nil || len(skills) != 2 || skills[0].SkillID != "skill-legacy" || skills[0].State != ComposerSkillAccepted || skills[1].SkillID != "skill-webapp-testing" {
		t.Fatalf("unexpected skills: %+v, %v", skills, err)
	}
	if err := st.SetComposerSkillState(session.ID, "skill-webapp-testing", ComposerSkillApplied); err != nil {
		t.Fatal(err)
	}
	skills, err = st.ListComposerSkillProposals(session.ID)
	if err != nil || skills[1].State != ComposerSkillApplied {
		t.Fatalf("unexpected updated skills: %+v, %v", skills, err)
	}
	if artifact.Version != 1 || artifact.Hash == "" {
		t.Fatalf("immutable artifact missing identity: %+v", artifact)
	}
	artifacts, err := st.ListPromptArtifacts(session.ID)
	if err != nil || len(artifacts) != 1 || artifacts[0].ID != artifact.ID {
		t.Fatalf("unexpected artifacts: %+v, %v", artifacts, err)
	}
}

func TestPromptArtifactDoesNotSilentlyTruncate(t *testing.T) {
	st := openTestStore(t)
	project, err := st.CreateProject(Project{Name: "Prompt artifact size", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	session, err := st.CreateComposerSession(ComposerSession{ProjectID: project.ID, BriefJSON: `{"goal":"large prompt"}`})
	if err != nil {
		t.Fatal(err)
	}

	content := strings.Repeat("x", 32*1024)
	artifact, err := st.CreatePromptArtifact(PromptArtifact{SessionID: session.ID, Content: content})
	if err != nil {
		t.Fatal(err)
	}
	if artifact.Content != content {
		t.Fatalf("prompt artifact was silently modified: got %d bytes want %d", len(artifact.Content), len(content))
	}

	tooLarge := strings.Repeat("x", maxPromptArtifactBytes+1)
	if _, err := st.CreatePromptArtifact(PromptArtifact{SessionID: session.ID, Content: tooLarge}); err == nil {
		t.Fatal("expected oversized prompt artifact to fail explicitly")
	}
}

func TestComposerContextManifestAndSkillScoreRoundTrip(t *testing.T) {
	st := openTestStore(t)
	project, err := st.CreateProject(Project{Name: "Composer context manifest", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	session, err := st.CreateComposerSession(ComposerSession{
		ProjectID:           project.ID,
		BriefJSON:           `{"goal":"inspect context"}`,
		ContextManifestJSON: `{"version":"1","fingerprint":"ctx-fp"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := st.GetComposerSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ContextManifestJSON != session.ContextManifestJSON {
		t.Fatalf("context manifest did not round-trip: got %q want %q", loaded.ContextManifestJSON, session.ContextManifestJSON)
	}
	if _, err := st.UpsertComposerSkillProposal(ComposerSkillProposal{
		SessionID: session.ID, SkillID: "security", State: ComposerSkillSuggested,
		Reason: "security match", RecommendationScore: 42,
	}); err != nil {
		t.Fatal(err)
	}
	skills, err := st.ListComposerSkillProposals(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(skills) != 1 || skills[0].RecommendationScore != 42 {
		t.Fatalf("skill recommendation score did not round-trip: %+v", skills)
	}
	session.ContextManifestJSON = `{"version":"1","fingerprint":"ctx-fp-2"}`
	if err := st.UpdateComposerSession(*session); err != nil {
		t.Fatal(err)
	}
	loaded, err = st.GetComposerSession(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ContextManifestJSON != session.ContextManifestJSON {
		t.Fatalf("updated context manifest did not persist: got %q want %q", loaded.ContextManifestJSON, session.ContextManifestJSON)
	}
}

func TestComposerV2MigrationPersistsVariantsAndReceipts(t *testing.T) {
	st := openTestStore(t)
	project, err := st.CreateProject(Project{Name: "Composer v2", CanonicalPath: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	session, err := st.CreateComposerSession(ComposerSession{ProjectID: project.ID, BriefJSON: `{}`})
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := st.CreatePromptArtifact(PromptArtifact{SessionID: session.ID, Content: "prompt"})
	if err != nil {
		t.Fatal(err)
	}
	variant, err := st.CreatePromptVariant(PromptVariant{ArtifactID: artifact.ID, Variant: "GENERIC_PORTABLE", Content: "portable"})
	if err != nil || variant.Hash == "" {
		t.Fatalf("variant: %+v %v", variant, err)
	}
	if _, err := st.RecordComposerDestinationReceipt(ComposerDestinationReceipt{ArtifactID: artifact.ID, Destination: "COPY", Variant: "GENERIC_PORTABLE", Status: "RECORDED"}); err != nil {
		t.Fatal(err)
	}
	variants, err := st.ListPromptVariants(artifact.ID)
	if err != nil || len(variants) != 1 {
		t.Fatalf("variants: %+v %v", variants, err)
	}
}
