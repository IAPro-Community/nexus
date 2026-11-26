package app

import (
	"testing"
	"time"

	"github.com/IAPro-Community/nexus/internal/conversation"
)

func TestLatestConversationForProfileUsesHarnessAndSelectedProfile(t *testing.T) {
	newer := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)
	conversations := []conversation.Conversation{
		{Provider: "agy", Profile: "agy1", ID: "agy-old", LastModified: newer.Add(-time.Hour)},
		{Provider: "agy", Profile: "agy2", ID: "agy-newer-other-profile", LastModified: newer.Add(time.Hour)},
		{Provider: "codex", Profile: "agy1", ID: "different-harness", LastModified: newer.Add(2 * time.Hour)},
		{Provider: "agy", Profile: "agy1", ID: "agy-selected", LastModified: newer},
	}

	got := latestConversationForProfile(conversations, "agy", "agy1", "/workspace")
	if got.ID != "agy-selected" || got.Profile != "agy1" {
		t.Fatalf("latest selected profile conversation = %+v", got)
	}

	got = latestConversationForProfile(conversations, "agy", "", "/workspace")
	if got.ID != "agy-newer-other-profile" || got.Profile != "agy2" {
		t.Fatalf("latest conversation across profiles = %+v", got)
	}
}
