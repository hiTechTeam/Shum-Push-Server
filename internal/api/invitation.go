package api

import (
	"crypto/sha256"
	_ "embed"
	"encoding/base64"
	"io"
	"net/http"
	"strings"
)

//go:embed invitation.html
var invitationHTML string

//go:embed invitation.js
var invitationScript string

func invitationPage(w http.ResponseWriter, r *http.Request) {
	// All contact data lives in the URL fragment and is handled by the browser.
	// Neither the server nor link previews need access to the contact locator.
	digest := sha256.Sum256([]byte(invitationScript))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; script-src 'sha256-"+base64.StdEncoding.EncodeToString(digest[:])+"'; style-src 'unsafe-inline'; base-uri 'none'; frame-ancestors 'none'; form-action 'none'")
	if r.Method == http.MethodHead {
		return
	}
	_, _ = io.WriteString(w, strings.Replace(invitationHTML, "/*SHUM_INVITATION_SCRIPT*/", invitationScript, 1))
}
