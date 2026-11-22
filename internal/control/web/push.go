package web

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Web Push (VAPID + aes128gcm) is implemented on the standard library only:
// the repo must not gain a dependency that pulls in a second HTTP stack just
// to notify a phone. The payload is encrypted per RFC 8291 so the relay never
// sees the notification text.

const (
	pushFileName       = "push.json"
	pushRecordSize     = 4096
	pushMaxSubscribers = 16
	pushVAPIDSubject   = "mailto:nexus@localhost"
	pushVAPIDLifetime  = 12 * time.Hour
	pushSendTimeout    = 10 * time.Second
)

// PushSubscription is one browser subscription. The endpoint is the delivery
// address issued by the browser vendor; p256dh/auth are the client's public
// key material used to encrypt the payload.
type PushSubscription struct {
	ID        string    `json:"id"`
	Endpoint  string    `json:"endpoint"`
	P256DH    string    `json:"p256dh"`
	Auth      string    `json:"auth"`
	CreatedAt time.Time `json:"created_at"`
}

type persistedPushStore struct {
	PrivateKey    string             `json:"private_key"`
	Subscriptions []PushSubscription `json:"subscriptions"`
}

// pushService owns the VAPID key pair and the subscription list. Subscriptions
// are stored on the host only: a paired phone cannot read or alter them.
type pushService struct {
	mu       sync.Mutex
	storeDir string
	private  *ecdh.PrivateKey
	subs     map[string]PushSubscription
	client   *http.Client
	now      func() time.Time
}

func newPushService(storeDir string) (*pushService, error) {
	p := &pushService{
		storeDir: storeDir,
		subs:     map[string]PushSubscription{},
		client:   &http.Client{Timeout: pushSendTimeout},
		now:      time.Now,
	}
	if err := p.load(); err != nil {
		return nil, err
	}
	if p.private == nil {
		key, err := ecdh.P256().GenerateKey(rand.Reader)
		if err != nil {
			return nil, err
		}
		p.private = key
		if err := p.persistLocked(); err != nil {
			return nil, err
		}
	}
	return p, nil
}

// PublicKey is the uncompressed VAPID public key, base64url, as required by
// PushManager.subscribe({applicationServerKey}).
func (p *pushService) PublicKey() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return base64.RawURLEncoding.EncodeToString(p.private.PublicKey().Bytes())
}

func (p *pushService) Count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.subs)
}

func (p *pushService) Subscribe(endpoint, p256dh, auth string) error {
	endpoint = strings.TrimSpace(endpoint)
	p256dh = strings.TrimSpace(p256dh)
	auth = strings.TrimSpace(auth)
	if endpoint == "" || p256dh == "" || auth == "" {
		return fmt.Errorf("incomplete push subscription")
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return fmt.Errorf("invalid push endpoint")
	}
	keyBytes, err := base64.RawURLEncoding.DecodeString(p256dh)
	if err != nil {
		return fmt.Errorf("invalid p256dh key")
	}
	if _, err := ecdh.P256().NewPublicKey(keyBytes); err != nil {
		return fmt.Errorf("invalid p256dh key")
	}
	if _, err := base64.RawURLEncoding.DecodeString(auth); err != nil {
		return fmt.Errorf("invalid auth secret")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	id := pushSubscriptionID(endpoint)
	if _, exists := p.subs[id]; !exists && len(p.subs) >= pushMaxSubscribers {
		return fmt.Errorf("too many push subscriptions")
	}
	p.subs[id] = PushSubscription{
		ID:        id,
		Endpoint:  endpoint,
		P256DH:    p256dh,
		Auth:      auth,
		CreatedAt: p.now().UTC(),
	}
	return p.persistLocked()
}

func (p *pushService) Unsubscribe(endpoint string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.subs, pushSubscriptionID(strings.TrimSpace(endpoint)))
	return p.persistLocked()
}

// PushResult reports one delivery attempt. Expired subscriptions are dropped.
type PushResult struct {
	Attempted int `json:"attempted"`
	Delivered int `json:"delivered"`
	Pruned    int `json:"pruned"`
}

// Notify encrypts payload for every subscription and delivers it. A payload is
// capped so a single record never needs multiple aes128gcm chunks.
func (p *pushService) Notify(payload []byte) PushResult {
	if len(payload) > pushRecordSize-128 {
		payload = payload[:pushRecordSize-128]
	}
	p.mu.Lock()
	targets := make([]PushSubscription, 0, len(p.subs))
	for _, sub := range p.subs {
		targets = append(targets, sub)
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].CreatedAt.Before(targets[j].CreatedAt) })
	p.mu.Unlock()

	result := PushResult{Attempted: len(targets)}
	stale := make([]string, 0, 1)
	for _, sub := range targets {
		status, err := p.deliver(sub, payload)
		switch {
		case err == nil:
			result.Delivered++
		case status == http.StatusNotFound || status == http.StatusGone:
			stale = append(stale, sub.Endpoint)
		}
	}
	if len(stale) > 0 {
		p.mu.Lock()
		for _, endpoint := range stale {
			delete(p.subs, pushSubscriptionID(endpoint))
		}
		result.Pruned = len(stale)
		_ = p.persistLocked()
		p.mu.Unlock()
	}
	return result
}

func (p *pushService) deliver(sub PushSubscription, payload []byte) (int, error) {
	body, err := encryptPushPayload(sub.P256DH, sub.Auth, payload)
	if err != nil {
		return 0, err
	}
	token, err := p.vapidToken(sub.Endpoint)
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequest(http.MethodPost, sub.Endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("TTL", "86400")
	req.Header.Set("Urgency", "high")
	req.Header.Set("Authorization", "vapid t="+token+", k="+p.PublicKey())
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<12))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp.StatusCode, nil
	}
	return resp.StatusCode, fmt.Errorf("push endpoint returned %d", resp.StatusCode)
}

// vapidToken builds the ES256 JWT required by RFC 8292.
func (p *pushService) vapidToken(endpoint string) (string, error) {
	audience, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	p.mu.Lock()
	key := p.private
	p.mu.Unlock()

	header := base64.RawURLEncoding.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256"}`))
	claims, err := json.Marshal(map[string]any{
		"aud": audience.Scheme + "://" + audience.Host,
		"exp": p.now().Add(pushVAPIDLifetime).Unix(),
		"sub": pushVAPIDSubject,
	})
	if err != nil {
		return "", err
	}
	signingInput := header + "." + base64.RawURLEncoding.EncodeToString(claims)
	digest := sha256.Sum256([]byte(signingInput))
	ecdsaKey := ecdsaKeyFromECDH(key)
	r, s, err := ecdsa.Sign(rand.Reader, ecdsaKey, digest[:])
	if err != nil {
		return "", err
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(signature), nil
}

// ecdsaKeyFromECDH converts an ECDH P-256 key into the ECDSA form required by
// the VAPID JWT signature. The public point is read from the uncompressed
// encoding produced by crypto/ecdh, so no deprecated low-level curve math is
// needed.
func ecdsaKeyFromECDH(key *ecdh.PrivateKey) *ecdsa.PrivateKey {
	d := new(big.Int).SetBytes(key.Bytes())
	pub := key.PublicKey().Bytes()
	return &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: elliptic.P256(),
			X:     new(big.Int).SetBytes(pub[1:33]),
			Y:     new(big.Int).SetBytes(pub[33:65]),
		},
		D: d,
	}
}

// encryptPushPayload implements RFC 8291 with the aes128gcm content coding of
// RFC 8188. Layout: salt(16) | rs(4) | idlen(1) | as_public(65) | ciphertext.
func encryptPushPayload(p256dh, auth string, plaintext []byte) ([]byte, error) {
	clientKey, err := base64.RawURLEncoding.DecodeString(p256dh)
	if err != nil {
		return nil, err
	}
	authSecret, err := base64.RawURLEncoding.DecodeString(auth)
	if err != nil {
		return nil, err
	}
	uaPublic, err := ecdh.P256().NewPublicKey(clientKey)
	if err != nil {
		return nil, err
	}
	asPrivate, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	asPublic := asPrivate.PublicKey().Bytes()
	shared, err := asPrivate.ECDH(uaPublic)
	if err != nil {
		return nil, err
	}

	keyInfo := append([]byte("WebPush: info\x00"), clientKey...)
	keyInfo = append(keyInfo, asPublic...)
	ikm, err := hkdf.Key(sha256.New, shared, authSecret, string(keyInfo), 32)
	if err != nil {
		return nil, err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}
	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, err
	}
	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	// 0x02 marks the last (and only) record so the client does not expect more.
	padded := append(append([]byte{}, plaintext...), 0x02)
	ciphertext := gcm.Seal(nil, nonce, padded, nil)

	header := make([]byte, 0, 16+4+1+len(asPublic))
	header = append(header, salt...)
	recordSize := make([]byte, 4)
	binary.BigEndian.PutUint32(recordSize, pushRecordSize)
	header = append(header, recordSize...)
	header = append(header, byte(len(asPublic)))
	header = append(header, asPublic...)
	return append(header, ciphertext...), nil
}

func pushSubscriptionID(endpoint string) string {
	sum := sha256.Sum256([]byte(endpoint))
	return hex.EncodeToString(sum[:12])
}

func (p *pushService) load() error {
	if p.storeDir == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(p.storeDir, pushFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var stored persistedPushStore
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil
	}
	if raw, err := base64.RawURLEncoding.DecodeString(stored.PrivateKey); err == nil && len(raw) > 0 {
		if key, err := ecdh.P256().NewPrivateKey(raw); err == nil {
			p.private = key
		}
	}
	for _, sub := range stored.Subscriptions {
		if sub.Endpoint == "" {
			continue
		}
		p.subs[sub.ID] = sub
	}
	return nil
}

func (p *pushService) persistLocked() error {
	if p.storeDir == "" || p.private == nil {
		return nil
	}
	if err := os.MkdirAll(p.storeDir, webAuthDirMode); err != nil {
		return err
	}
	payload := persistedPushStore{
		PrivateKey:    base64.RawURLEncoding.EncodeToString(p.private.Bytes()),
		Subscriptions: make([]PushSubscription, 0, len(p.subs)),
	}
	for _, sub := range p.subs {
		payload.Subscriptions = append(payload.Subscriptions, sub)
	}
	sort.Slice(payload.Subscriptions, func(i, j int) bool {
		return payload.Subscriptions[i].CreatedAt.Before(payload.Subscriptions[j].CreatedAt)
	})
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return atomicWriteFile(filepath.Join(p.storeDir, pushFileName), data)
}
