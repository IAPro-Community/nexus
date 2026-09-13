package update

import (
	"testing"
)

func TestProductionTrustRootIsEmptyByDefault(t *testing.T) {
	if len(ProductionTrustRoot) != 0 {
		t.Fatalf("production trust root must be empty in local/test builds, got %d bytes", len(ProductionTrustRoot))
	}
	kr := NewKeyRing()
	if _, err := kr.VerifyManifest([]byte(`{"schema_version":1}`), "00"); err == nil {
		t.Fatal("malformed manifest must still fail")
	}
	if _, ok := kr.keys[ProductionTrustRootKeyID]; ok {
		t.Fatalf("local keyring must not contain the production key")
	}
}
