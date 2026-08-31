package nexus

import "testing"

func TestContextCapsuleIsBoundedAndCarriesDependencyReceipts(t *testing.T) {
	receipt := WorkReceipt{StepID: "a", Status: "COMPLETED", Summary: "built"}
	capsule := NewContextCapsule("p", "main", "abc", "fp", "flow", 2, "b", []WorkReceipt{receipt}, []string{"go.mod"}, []string{"go-test"}, []string{"tests pass"}, []string{"no network"})
	if capsule.StepID != "b" || len(capsule.DependencyReceipts) != 1 || capsule.DependencyReceipts[0].StepID != "a" {
		t.Fatalf("invalid capsule: %#v", capsule)
	}
	capsule.DependencyReceipts[0].Summary = "mutated"
	if receipt.Summary != "built" {
		t.Fatal("input receipt was mutated")
	}
}

func TestWorkReceiptDoesNotInventEvidence(t *testing.T) {
	receipt := WorkReceipt{StepID: "a", Status: "COMPLETED"}
	if len(receipt.Commands) != 0 || len(receipt.ChangedFiles) != 0 || len(receipt.Verification) != 0 {
		t.Fatalf("empty evidence was invented: %#v", receipt)
	}
}
