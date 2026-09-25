package apns

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hiTechTeam/Shum-Push-Server/internal/store"
)

type Notification struct {
	EventID string
	Kind    string
}

type Client struct {
	sandbox    *environmentClient
	production *environmentClient
	bundleID   string
}

type environmentClient struct {
	keyID   string
	teamID  string
	key     *ecdsa.PrivateKey
	http    *http.Client
	mu      sync.Mutex
	jwt     string
	jwtTime time.Time
}

type Error struct {
	Status int
	Reason string
}

func (e *Error) Error() string { return fmt.Sprintf("APNs returned %d: %s", e.Status, e.Reason) }
func (e *Error) InvalidToken() bool {
	return e.Status == http.StatusGone || e.Reason == "BadDeviceToken" || e.Reason == "DeviceTokenNotForTopic" || e.Reason == "Unregistered"
}

func New(sandboxKeyID, sandboxKeyFile, productionKeyID, productionKeyFile, teamID, bundleID string) (*Client, error) {
	sandbox, err := newEnvironmentClient(sandboxKeyID, teamID, sandboxKeyFile)
	if err != nil {
		return nil, fmt.Errorf("configure sandbox APNs client: %w", err)
	}
	production, err := newEnvironmentClient(productionKeyID, teamID, productionKeyFile)
	if err != nil {
		return nil, fmt.Errorf("configure production APNs client: %w", err)
	}
	return &Client{sandbox: sandbox, production: production, bundleID: bundleID}, nil
}

func newEnvironmentClient(keyID, teamID, keyFile string) (*environmentClient, error) {
	bytes, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("read APNs private key: %w", err)
	}
	block, _ := pem.Decode(bytes)
	if block == nil {
		return nil, errors.New("APNs private key is not PEM")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse APNs private key: %w", err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || key.Curve.Params().Name != "P-256" {
		return nil, errors.New("APNs private key must use P-256")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.ForceAttemptHTTP2 = true
	return &environmentClient{
		keyID: keyID, teamID: teamID, key: key,
		http: &http.Client{Transport: transport, Timeout: 15 * time.Second},
	}, nil
}

func (c *Client) Send(ctx context.Context, device store.Device, notification Notification) error {
	payload := map[string]any{
		"aps": map[string]any{
			"alert":             map[string]any{"title": "Shum", "loc-key": "PUSH_NEW_MESSAGE_BODY"},
			"sound":             "default",
			"content-available": 1,
		},
		"shum": map[string]any{"event_id": notification.EventID, "kind": notification.Kind},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	host := "https://api.push.apple.com"
	environment := c.production
	if device.Environment == store.Sandbox {
		host = "https://api.sandbox.push.apple.com"
		environment = c.sandbox
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, host+"/3/device/"+device.Token, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	token, err := environment.providerToken()
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "bearer "+token)
	req.Header.Set("apns-topic", c.bundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("apns-expiration", fmt.Sprintf("%d", time.Now().Add(24*time.Hour).Unix()))
	if notification.EventID != "" {
		req.Header.Set("apns-collapse-id", notification.EventID)
	}
	response, err := environment.http.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		_, _ = io.Copy(io.Discard, response.Body)
		return nil
	}
	var failure struct {
		Reason string `json:"reason"`
	}
	_ = json.NewDecoder(io.LimitReader(response.Body, 4096)).Decode(&failure)
	return &Error{Status: response.StatusCode, Reason: failure.Reason}
}

func (c *environmentClient) providerToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	if c.jwt != "" && now.Sub(c.jwtTime) < 50*time.Minute {
		return c.jwt, nil
	}
	header, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": c.keyID})
	claims, _ := json.Marshal(map[string]any{"iss": c.teamID, "iat": now.Unix()})
	unsigned := encode(header) + "." + encode(claims)
	digest := sha256.Sum256([]byte(unsigned))
	r, s, err := ecdsa.Sign(rand.Reader, c.key, digest[:])
	if err != nil {
		return "", err
	}
	signature := make([]byte, 64)
	r.FillBytes(signature[:32])
	s.FillBytes(signature[32:])
	c.jwt = unsigned + "." + encode(signature)
	c.jwtTime = now
	return c.jwt, nil
}

func encode(value []byte) string { return base64.RawURLEncoding.EncodeToString(value) }
