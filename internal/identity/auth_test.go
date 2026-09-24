package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

func TestContactCardValidation(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	noise := make([]byte, 32)
	if _, err := rand.Read(noise); err != nil {
		t.Fatal(err)
	}
	nostr := make([]byte, 32)
	if _, err := rand.Read(nostr); err != nil {
		t.Fatal(err)
	}
	card := ContactCard{
		Version:    1,
		NoiseKey:   base64.RawURLEncoding.EncodeToString(noise),
		SigningKey: base64.RawURLEncoding.EncodeToString(public),
		NostrKey:   hex.EncodeToString(nostr), Name: "Test", Bio: "",
	}
	bytes, err := card.swiftSigningBytes(noise, public)
	if err != nil {
		t.Fatal(err)
	}
	card.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(private, bytes))
	identity, key, err := card.Validate()
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(noise)
	if identity != hex.EncodeToString(digest[:]) {
		t.Fatalf("unexpected identity %s", identity)
	}
	if string(key) != string(public) {
		t.Fatal("unexpected signing key")
	}
}
