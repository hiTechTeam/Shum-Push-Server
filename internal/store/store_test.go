package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStorePersistsAndRemovesDevices(t *testing.T) {
	path := filepath.Join(t.TempDir(), "devices.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	device := Device{Token: "abc", Environment: Sandbox, UpdatedAt: time.Unix(1, 0).UTC()}
	if err := s.Upsert("owner", device); err != nil {
		t.Fatal(err)
	}
	reloaded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	values := reloaded.Devices("owner")
	if len(values) != 1 || values[0].Token != "abc" {
		t.Fatalf("unexpected devices: %#v", values)
	}
	if err := reloaded.Remove("owner", "abc"); err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Devices("owner")) != 0 {
		t.Fatal("device was not removed")
	}
}
