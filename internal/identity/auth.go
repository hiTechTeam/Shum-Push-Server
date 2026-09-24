package identity

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	headerPublicKey = "X-Shum-Public-Key"
	headerTimestamp = "X-Shum-Timestamp"
	headerNonce     = "X-Shum-Nonce"
	headerSignature = "X-Shum-Signature"
)

type ContactCard struct {
	Version    int    `json:"version"`
	NoiseKey   string `json:"noise_key"`
	SigningKey string `json:"signing_key"`
	NostrKey   string `json:"nostr_key"`
	Name       string `json:"name"`
	Bio        string `json:"bio"`
	Signature  string `json:"signature"`
}

func (c ContactCard) Validate() (identityID string, signingKey ed25519.PublicKey, err error) {
	noise, err := decodeBase64URL(c.NoiseKey, 32)
	if err != nil || allZero(noise) {
		return "", nil, errors.New("invalid noise key")
	}
	key, err := decodeBase64URL(c.SigningKey, ed25519.PublicKeySize)
	if err != nil {
		return "", nil, errors.New("invalid signing key")
	}
	signature, err := decodeBase64URL(c.Signature, ed25519.SignatureSize)
	if err != nil {
		return "", nil, errors.New("invalid contact signature")
	}
	if c.Version != 1 || !validHex(c.NostrKey, 64) || strings.TrimSpace(c.Name) == "" || len([]byte(c.Name)) > 64 || utf8.RuneCountInString(c.Bio) > 72 {
		return "", nil, errors.New("invalid contact card")
	}
	signed, err := c.swiftSigningBytes(noise, key)
	if err != nil || !ed25519.Verify(ed25519.PublicKey(key), signed, signature) {
		return "", nil, errors.New("contact signature verification failed")
	}
	digest := sha256.Sum256(noise)
	return hex.EncodeToString(digest[:]), ed25519.PublicKey(key), nil
}

func (c ContactCard) swiftSigningBytes(noise, signing []byte) ([]byte, error) {
	value := map[string]any{
		"bio":        c.Bio,
		"name":       c.Name,
		"noiseKey":   base64.StdEncoding.EncodeToString(noise),
		"nostrKey":   c.NostrKey,
		"signature":  "",
		"signingKey": base64.StdEncoding.EncodeToString(signing),
		"version":    c.Version,
	}
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(value); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buffer.Bytes(), []byte("\n")), nil
}

type Verifier struct {
	mu     sync.Mutex
	nonces map[string]time.Time
	now    func() time.Time
}

func NewVerifier() *Verifier {
	return &Verifier{nonces: map[string]time.Time{}, now: time.Now}
}

func (v *Verifier) Verify(r *http.Request, body []byte, expected ed25519.PublicKey) error {
	publicKey, err := decodeBase64URL(r.Header.Get(headerPublicKey), ed25519.PublicKeySize)
	if err != nil || subtle.ConstantTimeCompare(publicKey, expected) != 1 {
		return errors.New("request key does not match contact card")
	}
	timestampText := r.Header.Get(headerTimestamp)
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil {
		return errors.New("invalid request timestamp")
	}
	now := v.now()
	delta := now.Sub(time.Unix(timestamp, 0))
	if delta < -5*time.Minute || delta > 5*time.Minute {
		return errors.New("request timestamp is outside the allowed window")
	}
	nonce := r.Header.Get(headerNonce)
	if len(nonce) < 16 || len(nonce) > 128 {
		return errors.New("invalid request nonce")
	}
	signature, err := decodeBase64URL(r.Header.Get(headerSignature), ed25519.SignatureSize)
	if err != nil {
		return errors.New("invalid request signature")
	}
	canonical := Canonical(r.Method, r.URL.EscapedPath(), timestampText, nonce, body)
	if !ed25519.Verify(expected, canonical, signature) {
		return errors.New("request signature verification failed")
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	for value, expiry := range v.nonces {
		if !expiry.After(now) {
			delete(v.nonces, value)
		}
	}
	nonceKey := base64.RawURLEncoding.EncodeToString(expected) + ":" + nonce
	if _, exists := v.nonces[nonceKey]; exists {
		return errors.New("request nonce was already used")
	}
	v.nonces[nonceKey] = now.Add(10 * time.Minute)
	return nil
}

func Canonical(method, path, timestamp, nonce string, body []byte) []byte {
	digest := sha256.Sum256(body)
	return []byte(fmt.Sprintf("SHUM1\n%s\n%s\n%s\n%s\n%s", strings.ToUpper(method), path, timestamp, nonce, hex.EncodeToString(digest[:])))
}

func decodeBase64URL(value string, expected int) ([]byte, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(decoded) != expected {
		return nil, errors.New("invalid base64url value")
	}
	return decoded, nil
}

func validHex(value string, length int) bool {
	if len(value) != length || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func allZero(value []byte) bool {
	var result byte
	for _, item := range value {
		result |= item
	}
	return result == 0
}
