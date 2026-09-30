package api

import (
	"context"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/hiTechTeam/Shum-Push-Server/internal/apns"
	"github.com/hiTechTeam/Shum-Push-Server/internal/identity"
	"github.com/hiTechTeam/Shum-Push-Server/internal/store"
)

type PushSender interface {
	Send(context.Context, store.Device, apns.Notification) error
}

type Server struct {
	devices       *store.Store
	push          PushSender
	verifier      *identity.Verifier
	gatewaySecret string
	logger        *slog.Logger
	limiter       *limiter
}

func New(devices *store.Store, push PushSender, gatewaySecret string, logger *slog.Logger) http.Handler {
	s := &Server{
		devices: devices, push: push, verifier: identity.NewVerifier(),
		gatewaySecret: gatewaySecret, logger: logger,
		limiter: newLimiter(60, time.Minute),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.health)
	mux.HandleFunc("GET /readyz", s.ready)
	mux.HandleFunc("GET /invite", invitationPage)
	mux.HandleFunc("POST /v1/devices", s.register)
	mux.HandleFunc("DELETE /v1/devices", s.unregister)
	mux.HandleFunc("POST /v1/notifications", s.notify)
	return s.securityHeaders(s.recover(mux))
}

type deviceRequest struct {
	Card        identity.ContactCard `json:"card"`
	DeviceToken string               `json:"device_token"`
	Environment store.Environment    `json:"environment"`
	AppVersion  string               `json:"app_version,omitempty"`
}

type notificationRequest struct {
	Card        identity.ContactCard `json:"card"`
	RecipientID string               `json:"recipient_id"`
	EventID     string               `json:"event_id"`
	Kind        string               `json:"kind"`
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (s *Server) ready(w http.ResponseWriter, _ *http.Request) {
	identities, devices := s.devices.Count()
	status := http.StatusOK
	state := "ready"
	if s.push == nil {
		status = http.StatusServiceUnavailable
		state = "waiting_for_apns_credentials"
	}
	writeJSON(w, status, map[string]any{"status": state, "identities": identities, "devices": devices})
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	if !s.allowGateway(r) || !s.limiter.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "request rejected")
		return
	}
	body, err := readBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var request deviceRequest
	if json.Unmarshal(body, &request) != nil || !validDeviceToken(request.DeviceToken) || (request.Environment != store.Sandbox && request.Environment != store.Production) {
		writeError(w, http.StatusBadRequest, "invalid device registration")
		return
	}
	identityID, key, err := request.Card.Validate()
	if err == nil {
		err = s.verifier.Verify(r, body, key)
	}
	if err != nil {
		s.logger.Info("device registration rejected", "reason", err.Error())
		writeError(w, http.StatusUnauthorized, "authentication failed")
		return
	}
	device := store.Device{Token: strings.ToLower(request.DeviceToken), Environment: request.Environment, UpdatedAt: time.Now().UTC()}
	if err := s.devices.Upsert(identityID, device); err != nil {
		s.logger.Error("device registration persistence failed", "error", err)
		writeError(w, http.StatusInternalServerError, "registration failed")
		return
	}
	s.logger.Info("device registered", "environment", request.Environment)
	writeJSON(w, http.StatusCreated, map[string]any{"status": "registered"})
}

func (s *Server) unregister(w http.ResponseWriter, r *http.Request) {
	if !s.allowGateway(r) || !s.limiter.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "request rejected")
		return
	}
	body, err := readBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var request deviceRequest
	if json.Unmarshal(body, &request) != nil || !validDeviceToken(request.DeviceToken) {
		writeError(w, http.StatusBadRequest, "invalid device registration")
		return
	}
	identityID, key, err := request.Card.Validate()
	if err != nil || s.verifier.Verify(r, body, key) != nil {
		writeError(w, http.StatusUnauthorized, "authentication failed")
		return
	}
	if err := s.devices.Remove(identityID, strings.ToLower(request.DeviceToken)); err != nil {
		s.logger.Error("device removal persistence failed", "error", err)
		writeError(w, http.StatusInternalServerError, "removal failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) notify(w http.ResponseWriter, r *http.Request) {
	if !s.allowGateway(r) || !s.limiter.allow(clientIP(r)) {
		writeError(w, http.StatusTooManyRequests, "request rejected")
		return
	}
	if s.push == nil {
		writeError(w, http.StatusServiceUnavailable, "push delivery is not configured")
		return
	}
	body, err := readBody(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	var request notificationRequest
	if json.Unmarshal(body, &request) != nil || !validIdentity(request.RecipientID) || !validEventID(request.EventID) || (request.Kind != "message" && request.Kind != "invitation" && request.Kind != "reaction") {
		s.logger.Info("notification rejected", "reason", "invalid request")
		writeError(w, http.StatusBadRequest, "invalid notification request")
		return
	}
	_, key, err := request.Card.Validate()
	if err == nil {
		err = s.verifier.Verify(r, body, key)
	}
	if err != nil {
		s.logger.Info("notification rejected", "reason", err.Error())
		writeError(w, http.StatusUnauthorized, "authentication failed")
		return
	}
	devices := s.devices.Devices(request.RecipientID)
	sent, removed, failed := 0, 0, 0
	for _, device := range devices {
		err := s.push.Send(r.Context(), device, apns.Notification{EventID: request.EventID, Kind: request.Kind, Sender: request.Card.Name})
		if err == nil {
			sent++
			continue
		}
		var failure *apns.Error
		if errors.As(err, &failure) && failure.InvalidToken() {
			removed++
			s.logger.Warn("APNs rejected device token", "environment", device.Environment, "reason", err)
			if removeErr := s.devices.Remove(request.RecipientID, device.Token); removeErr != nil {
				s.logger.Error("invalid device cleanup failed", "error", removeErr)
			}
			continue
		}
		failed++
		s.logger.Warn("APNs delivery failed", "reason", err)
	}
	s.logger.Info("notification processed", "kind", request.Kind, "devices", len(devices),
		"sent", sent, "invalid_tokens", removed, "failed", failed)
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted"})
}

func (s *Server) allowGateway(r *http.Request) bool {
	if s.gatewaySecret == "" {
		return true
	}
	value := r.Header.Get("X-Shum-Gateway")
	return len(value) == len(s.gatewaySecret) && subtle.ConstantTimeCompare([]byte(value), []byte(s.gatewaySecret)) == 1
}

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if value := recover(); value != nil {
				s.logger.Error("request panic", "value", value)
				writeError(w, http.StatusInternalServerError, "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	bytes, err := io.ReadAll(io.LimitReader(r.Body, 16*1024+1))
	if err != nil {
		return nil, err
	}
	if len(bytes) > 16*1024 {
		return nil, errors.New("request body is too large")
	}
	return bytes, nil
}

func validDeviceToken(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func validIdentity(value string) bool { return len(value) == 64 && validDeviceToken(value) }

func validEventID(value string) bool {
	if len(value) < 16 || len(value) > 64 {
		return false
	}
	for _, character := range value {
		if !(character == '-' || character == '_' || character >= '0' && character <= '9' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z') {
			return false
		}
	}
	return true
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-For"), ",")[0]); forwarded != "" {
		return forwarded
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

type limiter struct {
	mu        sync.Mutex
	entries   map[string][]time.Time
	limit     int
	duration  time.Duration
	now       func() time.Time
	lastSweep time.Time
}

func newLimiter(limit int, duration time.Duration) *limiter {
	return &limiter{entries: map[string][]time.Time{}, limit: limit, duration: duration, now: time.Now}
}

// allow keeps a client key only while it has requests inside the window, so
// client IP addresses do not outlive rate limiting.
func (l *limiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	cutoff := now.Add(-l.duration)
	if !now.Before(l.lastSweep.Add(l.duration)) {
		for entry, values := range l.entries {
			if len(values) == 0 || !values[len(values)-1].After(cutoff) {
				delete(l.entries, entry)
			}
		}
		l.lastSweep = now
	}
	values := l.entries[key]
	kept := values[:0]
	for _, value := range values {
		if value.After(cutoff) {
			kept = append(kept, value)
		}
	}
	if len(kept) >= l.limit {
		l.entries[key] = kept
		return false
	}
	l.entries[key] = append(kept, now)
	return true
}
