package update

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrUntrustedKeyID    = errors.New("key ID is not trusted or recognized in keyring")
	ErrInvalidSignature  = errors.New("ed25519 signature verification failed for manifest")
	ErrInvalidSchema     = errors.New("unsupported manifest schema version")
	ErrManifestMalformed = errors.New("manifest JSON is malformed")
)

const (
	// ProductionTrustRootKeyID is the canonical key ID used for production Nexus releases.
	ProductionTrustRootKeyID = "nexus-signing-key-2026-v1"
)

var (
	// ProductionTrustRootHex is injected only by a protected release build.
	// Local/dev binaries intentionally have no production trust root.
	ProductionTrustRootHex string
	ProductionTrustRoot    ed25519.PublicKey
)

func init() {
	hexKey := strings.TrimSpace(ProductionTrustRootHex)
	if hexKey == "" {
		return
	}
	pub, err := hex.DecodeString(hexKey)
	if err != nil || len(pub) != ed25519.PublicKeySize {
		panic("nexus update: injected production trust root is invalid — refuse to build an unverifiable updater")
	}
	ProductionTrustRoot = ed25519.PublicKey(pub)
}

type KeyRing struct {
	keys map[string]ed25519.PublicKey
}

func NewKeyRing() *KeyRing {
	kr := &KeyRing{
		keys: make(map[string]ed25519.PublicKey),
	}
	// Always include the production trust root when available
	if ProductionTrustRoot != nil {
		kr.keys[ProductionTrustRootKeyID] = ProductionTrustRoot
	}
	return kr
}

func (kr *KeyRing) AddKey(keyID string, pub ed25519.PublicKey) {
	kr.keys[keyID] = pub
}

// RemoveKey revokes a trust-root (rotation completion or compromise).
// Manifests signed by the removed key fail with ErrUntrustedKeyID while
// manifests from remaining keys keep verifying: rotation never needs a
// flag day.
func (kr *KeyRing) RemoveKey(keyID string) {
	delete(kr.keys, keyID)
}

func (kr *KeyRing) VerifyManifest(manifestBytes []byte, sigHex string) (*Manifest, error) {
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrManifestMalformed, err)
	}

	if err := manifest.Validate(ManifestPolicy{}); err != nil {
		if manifest.SchemaVersion != 1 {
			return nil, fmt.Errorf("%w: version %d", ErrInvalidSchema, manifest.SchemaVersion)
		}
		return nil, err
	}

	pubKey, ok := kr.keys[manifest.KeyID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUntrustedKeyID, manifest.KeyID)
	}

	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return nil, fmt.Errorf("%w: malformed hex signature: %v", ErrInvalidSignature, err)
	}

	if !ed25519.Verify(pubKey, manifestBytes, sigBytes) {
		return nil, ErrInvalidSignature
	}

	return &manifest, nil
}
