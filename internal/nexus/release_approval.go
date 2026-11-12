package nexus

import (
	"bytes"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// Signed release approval (release-desk evidence path).
//
// The approval artifact is produced by scripts/sign-release-approval.mjs:
// a canonical JSON payload {approvalId, approvedAtUtc, approver, schemaVersion,
// sourceSha} signed with an Ed25519 key enrolled in the release trust store.
// The verifier in this file is fail-closed: any parse, signature, SHA-binding
// or freshness failure leaves the release without an approval, so the desk
// keeps merge/publish blocked with an explicit reason.

const (
	// maxApprovalBundleBytes matches the evidence bundle metadata cap.
	maxApprovalBundleBytes = 2 * 1024 * 1024
	// approvalFreshness bounds the signed approval lifetime (90 days, like
	// the offline evidence timestamps) and tolerates +5 minutes of clock skew.
	approvalFreshness = 90 * 24 * time.Hour
	approvalSkew      = 5 * time.Minute
)

var releaseApprovalSHA = regexp.MustCompile(`^[0-9a-f]{40}$`)
var releaseApprovalTimestamp = regexp.MustCompile(`^\d{4}-\d\d-\d\dT\d\d:\d\d:\d\d(?:\.\d+)?Z$`)

// ReleaseApprovalBundle mirrors the signed JSON artifact.
type ReleaseApprovalBundle struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Approver      string                   `json:"approver"`
	ApprovalID    string                   `json:"approvalId"`
	SourceSha     string                   `json:"sourceSha"`
	ApprovedAtUtc string                   `json:"approvedAtUtc"`
	Signature     ReleaseApprovalSignature `json:"signature"`
}

// ReleaseApprovalSignature is the detached Ed25519 signature block.
type ReleaseApprovalSignature struct {
	KeyID  string `json:"keyId"`
	Base64 string `json:"base64"`
}

type releaseTrustStore struct {
	Reviewers []struct {
		ID           string `json:"id"`
		PublicKeyPem string `json:"publicKeyPem"`
	} `json:"reviewers"`
}

// canonicalJSONString escapes exactly like JavaScript JSON.stringify for the
// characters allowed here; control characters and DEL are rejected outright
// so the two canonicalizers cannot drift on exotic input.
func canonicalJSONString(value string) ([]byte, error) {
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			return nil, fmt.Errorf("approval payload contains control character U+%04X", r)
		}
	}
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// LoadReleaseApproval reads the signed approval artifact and the trust store
// from the project repo, verifies the Ed25519 signature and the SHA binding,
// and returns the approval bound to headSHA. Any problem returns (nil, reason)
// — never a partially trusted approval.
func LoadReleaseApproval(approvalPath, trustPath, headSHA string) (*ReleaseApproval, string) {
	bundleRaw, err := readCappedFile(approvalPath, maxApprovalBundleBytes)
	if err != nil {
		return nil, "approval artifact unavailable: " + err.Error()
	}
	trustRaw, err := readCappedFile(trustPath, maxApprovalBundleBytes)
	if err != nil {
		return nil, "trust store unavailable: " + err.Error()
	}
	record, err := VerifyReleaseApproval(bundleRaw, trustRaw, headSHA, time.Now())
	if err != nil {
		return nil, err.Error()
	}
	return &ReleaseApproval{
		Approver:   record.Approver,
		ApprovalID: record.ApprovalID,
		SHA:        record.SHA,
		ApprovedAt: record.ApprovedAt,
	}, ""
}

// VerifyReleaseApproval validates the bundle against the trust store and the
// expected head SHA.
func VerifyReleaseApproval(bundleRaw, trustRaw []byte, headSHA string, now time.Time) (ReleaseApproval, error) {
	var bundle ReleaseApprovalBundle
	decoder := json.NewDecoder(bytes.NewReader(bundleRaw))
	if err := decoder.Decode(&bundle); err != nil {
		return ReleaseApproval{}, fmt.Errorf("approval bundle is not valid JSON: %w", err)
	}
	if bundle.SchemaVersion != 1 {
		return ReleaseApproval{}, fmt.Errorf("approval bundle schemaVersion must be 1, got %d", bundle.SchemaVersion)
	}
	approver := strings.TrimSpace(bundle.Approver)
	approvalID := strings.TrimSpace(bundle.ApprovalID)
	if approver == "" || approvalID == "" {
		return ReleaseApproval{}, fmt.Errorf("approval bundle requires approver and approvalId")
	}
	if !releaseApprovalSHA.MatchString(strings.ToLower(bundle.SourceSha)) {
		return ReleaseApproval{}, fmt.Errorf("approval sourceSha must be a 40-char hex sha")
	}
	if !strings.EqualFold(bundle.SourceSha, strings.TrimSpace(headSHA)) {
		return ReleaseApproval{}, fmt.Errorf("approval is bound to %s but release head is %s", shortSHA(bundle.SourceSha), shortSHA(headSHA))
	}
	if !releaseApprovalTimestamp.MatchString(bundle.ApprovedAtUtc) {
		return ReleaseApproval{}, fmt.Errorf("approval timestamp must be explicit UTC")
	}
	approvedAt, err := time.Parse(time.RFC3339, bundle.ApprovedAtUtc)
	if err != nil {
		return ReleaseApproval{}, fmt.Errorf("approval timestamp is invalid: %w", err)
	}
	if approvedAt.After(now.Add(approvalSkew)) {
		return ReleaseApproval{}, fmt.Errorf("approval timestamp is in the future")
	}
	if now.Sub(approvedAt) > approvalFreshness {
		return ReleaseApproval{}, fmt.Errorf("approval is stale (older than 90 days); re-approval required")
	}
	if bundle.Signature.KeyID == "" || bundle.Signature.Base64 == "" {
		return ReleaseApproval{}, fmt.Errorf("approval bundle requires a detached signature")
	}
	signature, err := base64.StdEncoding.DecodeString(bundle.Signature.Base64)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return ReleaseApproval{}, fmt.Errorf("approval signature is not a valid Ed25519 signature")
	}
	publicKey, err := enrolledApprovalKey(trustRaw, bundle.Signature.KeyID)
	if err != nil {
		return ReleaseApproval{}, err
	}
	var payloadCanonical bytes.Buffer
	payloadCanonical.WriteByte('{')
	fields := []struct {
		name  string
		value any
	}{
		{"approvalId", approvalID},
		{"approvedAtUtc", bundle.ApprovedAtUtc},
		{"approver", approver},
		{"schemaVersion", bundle.SchemaVersion},
		{"sourceSha", strings.ToLower(bundle.SourceSha)},
	}
	// Build the canonical payload from the *validated* fields, not from a
	// re-serialization of arbitrary JSON, so hidden keys cannot influence it.
	for i, field := range fields {
		if i > 0 {
			payloadCanonical.WriteByte(',')
		}
		key, err := canonicalJSONString(field.name)
		if err != nil {
			return ReleaseApproval{}, err
		}
		payloadCanonical.Write(key)
		payloadCanonical.WriteByte(':')
		switch value := field.value.(type) {
		case string:
			encoded, err := canonicalJSONString(value)
			if err != nil {
				return ReleaseApproval{}, err
			}
			payloadCanonical.Write(encoded)
		case int:
			fmt.Fprintf(&payloadCanonical, "%d", value)
		}
	}
	payloadCanonical.WriteByte('}')
	if !ed25519.Verify(publicKey, payloadCanonical.Bytes(), signature) {
		return ReleaseApproval{}, fmt.Errorf("approval signature verification failed")
	}
	return ReleaseApproval{
		Approver:   approver,
		ApprovalID: approvalID,
		SHA:        strings.ToLower(bundle.SourceSha),
		ApprovedAt: approvedAt.UTC(),
	}, nil
}

// enrolledApprovalKey parses the trust store and returns the Ed25519 public
// key enrolled under keyID.
func enrolledApprovalKey(trustRaw []byte, keyID string) (ed25519.PublicKey, error) {
	var trust releaseTrustStore
	if err := json.Unmarshal(trustRaw, &trust); err != nil {
		return nil, fmt.Errorf("trust store is not valid JSON: %w", err)
	}
	for _, reviewer := range trust.Reviewers {
		if reviewer.ID != keyID {
			continue
		}
		block, _ := pem.Decode([]byte(reviewer.PublicKeyPem))
		if block == nil {
			return nil, fmt.Errorf("trust store key %s has an unreadable PEM block", keyID)
		}
		parsed, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("trust store key %s is not a valid public key: %w", keyID, err)
		}
		key, ok := parsed.(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("trust store key %s is not Ed25519", keyID)
		}
		return key, nil
	}
	return nil, fmt.Errorf("signer %s is not enrolled in the release trust store", keyID)
}

func readCappedFile(path string, cap int) ([]byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(raw) > cap {
		return nil, fmt.Errorf("file exceeds %d bytes", cap)
	}
	return raw, nil
}
