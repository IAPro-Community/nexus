package nexus

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// signApproval builds the canonical payload (sorted keys, JSON strings) and
// signs it — the same construction as scripts/sign-release-approval.mjs.
func signApproval(t *testing.T, privateKey ed25519.PrivateKey, payload map[string]any) []byte {
	t.Helper()
	keys := make([]string, 0, len(payload))
	for key := range payload {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var canonical []byte
	canonical = append(canonical, '{')
	for i, key := range keys {
		if i > 0 {
			canonical = append(canonical, ',')
		}
		keyJSON, err := json.Marshal(key)
		if err != nil {
			t.Fatal(err)
		}
		canonical = append(canonical, keyJSON...)
		canonical = append(canonical, ':')
		switch value := payload[key].(type) {
		case string:
			valueJSON, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			canonical = append(canonical, valueJSON...)
		case int:
			canonical = append(canonical, strconv.Itoa(value)...)
		}
	}
	canonical = append(canonical, '}')
	return ed25519.Sign(privateKey, canonical)
}

const approvalTestSHA = "0123456789abcdef0123456789abcdef01234567"

// approvalFixture writes a trust store and returns the bundle path, trust
// path and the generated signing key.
func approvalFixture(t *testing.T) (string, string, ed25519.PrivateKey) {
	t.Helper()
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	pemBlock, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	trust := map[string]any{
		"schemaVersion": 1,
		"reviewers": []map[string]string{
			{
				"id":           "owner",
				"publicKeyPem": string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pemBlock})),
			},
		},
	}
	trustRaw, err := json.Marshal(trust)
	if err != nil {
		t.Fatal(err)
	}
	trustPath := filepath.Join(t.TempDir(), "release-reviewers.json")
	if err := os.WriteFile(trustPath, trustRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	approvalPath := filepath.Join(t.TempDir(), "release-approval.json")
	return approvalPath, trustPath, privateKey
}

func writeApprovalBundle(t *testing.T, approvalPath string, bundle map[string]any) {
	t.Helper()
	raw, err := json.Marshal(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(approvalPath, raw, 0o600); err != nil {
		t.Fatal(err)
	}
}

func signedBundle(t *testing.T, privateKey ed25519.PrivateKey, payload map[string]any, keyID string) map[string]any {
	t.Helper()
	signature := base64.StdEncoding.EncodeToString(signApproval(t, privateKey, payload))
	bundle := map[string]any{}
	for key, value := range payload {
		bundle[key] = value
	}
	bundle["signature"] = map[string]any{"keyId": keyID, "base64": signature}
	return bundle
}

func standardPayload(approvalID, approvedAtUtc, approver string) map[string]any {
	return map[string]any{
		"approvalId":    approvalID,
		"approvedAtUtc": approvedAtUtc,
		"approver":      approver,
		"schemaVersion": 1,
		"sourceSha":     approvalTestSHA,
	}
}

func TestVerifyReleaseApprovalRoundTrip(t *testing.T) {
	approvalPath, trustPath, privateKey := approvalFixture(t)
	writeApprovalBundle(t, approvalPath, signedBundle(t, privateKey,
		standardPayload("GO-2026-001", "2026-11-10T12:00:00Z", "maestro"), "owner"))
	record, reason := LoadReleaseApproval(approvalPath, trustPath, approvalTestSHA)
	if reason != "" || record == nil {
		t.Fatalf("valid approval must verify, got %s", reason)
	}
	if record.Approver != "maestro" || record.ApprovalID != "GO-2026-001" || record.SHA != approvalTestSHA {
		t.Fatalf("approval binding lost: %+v", record)
	}
}

func TestVerifyReleaseApprovalFailsClosed(t *testing.T) {
	approvalPath, trustPath, privateKey := approvalFixture(t)
	now := time.Date(2026, 11, 10, 12, 1, 0, 0, time.UTC)
	valid := signedBundle(t, privateKey, standardPayload("GO-2026-001", "2026-11-10T12:00:00Z", "maestro"), "owner")

	// Tampered payload (approvalId changed after signing).
	tampered := signedBundle(t, privateKey, standardPayload("GO-2026-001", "2026-11-10T12:00:00Z", "maestro"), "owner")
	tampered["approvalId"] = "GO-2026-999"
	writeApprovalBundle(t, approvalPath, tampered)
	if _, err := VerifyReleaseApproval(mustRead(t, approvalPath), mustRead(t, trustPath), approvalTestSHA, now); err == nil {
		t.Fatal("tampered payload must fail")
	}

	// SHA mismatch: signed for one SHA, checked against another.
	writeApprovalBundle(t, approvalPath, valid)
	otherSHA := strings.Repeat("f", 40)
	if _, err := VerifyReleaseApproval(mustRead(t, approvalPath), mustRead(t, trustPath), otherSHA, now); err == nil ||
		!strings.Contains(err.Error(), "release head is") {
		t.Fatalf("sha mismatch must be explicit, got %v", err)
	}

	// Stale approval (91 days old).
	stale := now.Add(-91 * 24 * time.Hour).UTC().Format("2006-01-02T15:04:05Z")
	writeApprovalBundle(t, approvalPath, signedBundle(t, privateKey,
		standardPayload("GO-2026-001", stale, "maestro"), "owner"))
	if _, err := VerifyReleaseApproval(mustRead(t, approvalPath), mustRead(t, trustPath), approvalTestSHA, now); err == nil ||
		!strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale approval must fail, got %v", err)
	}

	// Unenrolled signer with a self-consistent signature.
	writeApprovalBundle(t, approvalPath, signedBundle(t, privateKey,
		standardPayload("X", "2026-11-10T12:00:00Z", "attacker"), "not-enrolled"))
	if _, err := VerifyReleaseApproval(mustRead(t, approvalPath), mustRead(t, trustPath), approvalTestSHA, now); err == nil ||
		!strings.Contains(err.Error(), "not enrolled") {
		t.Fatalf("unenrolled signer must fail, got %v", err)
	}

	// Missing file fails closed with a reason.
	if _, reason := LoadReleaseApproval(filepath.Join(t.TempDir(), "missing.json"), trustPath, approvalTestSHA); reason == "" {
		t.Fatal("missing approval must yield a reason")
	}
}

func TestVerifyReleaseApprovalClockSkewTolerated(t *testing.T) {
	approvalPath, trustPath, privateKey := approvalFixture(t)
	future := time.Now().UTC().Add(3 * time.Minute).Format("2006-01-02T15:04:05Z")
	writeApprovalBundle(t, approvalPath, signedBundle(t, privateKey,
		standardPayload("GO-2026-001", future, "maestro"), "owner"))
	if _, err := VerifyReleaseApproval(mustRead(t, approvalPath), mustRead(t, trustPath), approvalTestSHA, time.Now()); err != nil {
		t.Fatalf("skew within tolerance must pass: %v", err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
