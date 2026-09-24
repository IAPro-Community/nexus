package agentprojection

import (
	"crypto/sha256"
	"encoding/hex"
)

type OwnershipState string

const (
	OwnershipMissing           OwnershipState = "MISSING"
	OwnershipManagedUnmodified OwnershipState = "MANAGED_UNMODIFIED"
	OwnershipManagedModified   OwnershipState = "MANAGED_MODIFIED"
	OwnershipUnmanaged         OwnershipState = "UNMANAGED"
)

type OwnershipDecision string

const (
	DecisionCreate   OwnershipDecision = "CREATE"
	DecisionUpdate   OwnershipDecision = "UPDATE"
	DecisionConflict OwnershipDecision = "CONFLICT"
)

// ResolveOwnership is deliberately fail-closed. A provider adapter may create a
// missing file or update a Nexus-managed file whose observed hash still matches
// the previous receipt. Everything else is a conflict.
func ResolveOwnership(exists bool, managed bool, observedHash, receiptHash string) (OwnershipState, OwnershipDecision) {
	if !exists {
		return OwnershipMissing, DecisionCreate
	}
	if !managed {
		return OwnershipUnmanaged, DecisionConflict
	}
	if observedHash != "" && receiptHash != "" && observedHash == receiptHash {
		return OwnershipManagedUnmodified, DecisionUpdate
	}
	return OwnershipManagedModified, DecisionConflict
}

func ContentHash(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
