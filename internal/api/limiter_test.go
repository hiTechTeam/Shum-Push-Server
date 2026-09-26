package api

import (
	"testing"
	"time"
)

func TestLimiterRejectsRequestsOverLimitInsideWindow(t *testing.T) {
	now := time.Unix(1_000, 0)
	l := newLimiter(2, time.Minute)
	l.now = func() time.Time { return now }

	if !l.allow("192.0.2.1") || !l.allow("192.0.2.1") {
		t.Fatal("requests within the limit must be allowed")
	}
	if l.allow("192.0.2.1") {
		t.Fatal("request over the limit must be rejected")
	}
	now = now.Add(time.Minute + time.Second)
	if !l.allow("192.0.2.1") {
		t.Fatal("request after the window must be allowed")
	}
}

func TestLimiterForgetsClientAddressesAfterWindow(t *testing.T) {
	now := time.Unix(1_000, 0)
	l := newLimiter(60, time.Minute)
	l.now = func() time.Time { return now }

	l.allow("192.0.2.1")
	l.allow("198.51.100.7")
	now = now.Add(time.Minute + time.Second)
	l.allow("203.0.113.9")

	if _, ok := l.entries["192.0.2.1"]; ok {
		t.Fatal("expired address 192.0.2.1 is still kept in memory")
	}
	if _, ok := l.entries["198.51.100.7"]; ok {
		t.Fatal("expired address 198.51.100.7 is still kept in memory")
	}
	if len(l.entries) != 1 {
		t.Fatalf("expected only the active address, got %d entries", len(l.entries))
	}
}
