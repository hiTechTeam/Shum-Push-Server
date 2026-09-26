package api

import (
	"crypto/sha256"
	"encoding/base64"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvitationPageIsPublicAndDoesNotReflectRequestData(t *testing.T) {
	handler := New(nil, nil, "write-api-secret", slog.New(slog.NewTextHandler(io.Discard, nil)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/invite?contact=UNTRUSTED_CONTACT", nil))
	if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("unexpected response: %d %v", response.Code, response.Header())
	}
	body := response.Body.String()
	if strings.Contains(body, "UNTRUSTED_CONTACT") || strings.Contains(body, "/*SHUM_INVITATION_SCRIPT*/") {
		t.Fatal("request data reflected or script not embedded")
	}
	if !strings.Contains(body, "shum://c2/") || !strings.Contains(body, "https://testflight.apple.com/join/Z4pTvZcR") {
		t.Fatal("missing app or installation destination")
	}
	digest := sha256.Sum256([]byte(invitationScript))
	if !strings.Contains(response.Header().Get("Content-Security-Policy"), "'sha256-"+base64.StdEncoding.EncodeToString(digest[:])+"'") {
		t.Fatal("CSP does not authorize the exact embedded script")
	}
	if response.Header().Get("Referrer-Policy") != "no-referrer" || response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing privacy headers")
	}
}

func TestInvitationPageRejectsWrites(t *testing.T) {
	handler := New(nil, nil, "", slog.Default())
	for _, method := range []string{http.MethodPost, http.MethodDelete} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(method, "/invite", nil))
		if response.Code != http.StatusMethodNotAllowed {
			t.Fatalf("%s: expected 405, got %d", method, response.Code)
		}
	}
}
