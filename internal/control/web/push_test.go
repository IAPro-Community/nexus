package web

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// decryptPushPayload mirrors what a browser does with an aes128gcm push body.
// A roundtrip here proves the record layout and the key derivation, which are
// the parts that silently break delivery in production.
func decryptPushPayload(t *testing.T, clientKey *ecdh.PrivateKey, authSecret, body []byte) []byte {
	t.Helper()
	if len(body) < 16+4+1 {
		t.Fatalf("body too short: %d", len(body))
	}
	salt := body[:16]
	recordSize := binary.BigEndian.Uint32(body[16:20])
	if recordSize != pushRecordSize {
		t.Fatalf("record size = %d, want %d", recordSize, pushRecordSize)
	}
	idLen := int(body[20])
	keyID := body[21 : 21+idLen]
	ciphertext := body[21+idLen:]

	asPublic, err := ecdh.P256().NewPublicKey(keyID)
	if err != nil {
		t.Fatalf("as_public: %v", err)
	}
	shared, err := clientKey.ECDH(asPublic)
	if err != nil {
		t.Fatalf("ecdh: %v", err)
	}
	keyInfo := append([]byte("WebPush: info\x00"), clientKey.PublicKey().Bytes()...)
	keyInfo = append(keyInfo, keyID...)
	ikm, err := hkdf.Key(sha256.New, shared, authSecret, string(keyInfo), 32)
	if err != nil {
		t.Fatalf("ikm: %v", err)
	}
	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		t.Fatalf("cek: %v", err)
	}
	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		t.Fatalf("nonce: %v", err)
	}
	block, err := aes.NewCipher(cek)
	if err != nil {
		t.Fatalf("aes: %v", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		t.Fatalf("gcm: %v", err)
	}
	padded, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if len(padded) == 0 || padded[len(padded)-1] != 0x02 {
		t.Fatalf("missing last-record delimiter")
	}
	return padded[:len(padded)-1]
}

func TestPushPayloadRoundTrip(t *testing.T) {
	clientKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	authSecret := make([]byte, 16)
	if _, err := rand.Read(authSecret); err != nil {
		t.Fatal(err)
	}
	p256dh := base64.RawURLEncoding.EncodeToString(clientKey.PublicKey().Bytes())
	auth := base64.RawURLEncoding.EncodeToString(authSecret)
	payload := []byte(`{"title":"Nexus","body":"atenção necessária"}`)

	body, err := encryptPushPayload(p256dh, auth, payload)
	if err != nil {
		t.Fatalf("encryptPushPayload: %v", err)
	}
	got := decryptPushPayload(t, clientKey, authSecret, body)
	if string(got) != string(payload) {
		t.Fatalf("payload = %q, want %q", got, payload)
	}
}

func TestPushStoreSubscriptions(t *testing.T) {
	svc, err := newPushService(t.TempDir())
	if err != nil {
		t.Fatalf("newPushService: %v", err)
	}
	clientKey, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	authSecret := make([]byte, 16)
	if _, err := rand.Read(authSecret); err != nil {
		t.Fatal(err)
	}
	endpoint := "https://push.example.test/sub/abc"
	if err := svc.Subscribe(
		endpoint,
		base64.RawURLEncoding.EncodeToString(clientKey.PublicKey().Bytes()),
		base64.RawURLEncoding.EncodeToString(authSecret),
	); err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if svc.Count() != 1 {
		t.Fatalf("count = %d, want 1", svc.Count())
	}
	// Re-subscribing the same endpoint must not duplicate the entry.
	if err := svc.Subscribe(
		endpoint,
		base64.RawURLEncoding.EncodeToString(clientKey.PublicKey().Bytes()),
		base64.RawURLEncoding.EncodeToString(authSecret),
	); err != nil {
		t.Fatalf("re-Subscribe: %v", err)
	}
	if svc.Count() != 1 {
		t.Fatalf("count after re-subscribe = %d, want 1", svc.Count())
	}

	// The private key and the subscription must survive a restart.
	reloaded, err := newPushService(svc.storeDir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Count() != 1 {
		t.Fatalf("reloaded count = %d, want 1", reloaded.Count())
	}
	if reloaded.PublicKey() != svc.PublicKey() {
		t.Fatal("VAPID public key must be stable across restarts")
	}

	if err := svc.Unsubscribe(endpoint); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}
	if svc.Count() != 0 {
		t.Fatalf("count after unsubscribe = %d, want 0", svc.Count())
	}
}

func TestPushSubscribeRejectsInvalidInput(t *testing.T) {
	svc, err := newPushService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	valid := base64.RawURLEncoding.EncodeToString(make([]byte, 65))
	for name, tc := range map[string][3]string{
		"missing endpoint": {"", valid, "AAAA"},
		"not https":        {"http://", valid, "AAAA"},
		"bad p256dh":       {"https://push.example.test/x", "not-a-key", "AAAA"},
		"missing auth":     {"https://push.example.test/x", valid, ""},
	} {
		if err := svc.Subscribe(tc[0], tc[1], tc[2]); err == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
}

func TestPushVAPIDTokenShape(t *testing.T) {
	svc, err := newPushService(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	token, err := svc.vapidToken("https://fcm.googleapis.com/wp/abc")
	if err != nil {
		t.Fatalf("vapidToken: %v", err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("jwt parts = %d, want 3", len(parts))
	}
	claimsJSON, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("claims: %v", err)
	}
	var claims struct {
		Aud string `json:"aud"`
		Exp int64  `json:"exp"`
		Sub string `json:"sub"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		t.Fatalf("claims json: %v", err)
	}
	if claims.Aud != "https://fcm.googleapis.com" {
		t.Fatalf("aud = %q", claims.Aud)
	}
	if claims.Sub != pushVAPIDSubject {
		t.Fatalf("sub = %q", claims.Sub)
	}
	if claims.Exp <= time.Now().Unix() {
		t.Fatal("exp must be in the future")
	}
	if sig, err := base64.RawURLEncoding.DecodeString(parts[2]); err != nil || len(sig) != 64 {
		t.Fatalf("signature must be 64 raw bytes, got %d (err %v)", len(sig), err)
	}
}
