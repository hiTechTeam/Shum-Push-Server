package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

type Environment string

const (
	Sandbox    Environment = "sandbox"
	Production Environment = "production"
)

type Device struct {
	Token       string      `json:"token"`
	Environment Environment `json:"environment"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type snapshot struct {
	Version    int                 `json:"version"`
	Identities map[string][]Device `json:"identities"`
}

type Store struct {
	mu   sync.RWMutex
	path string
	data snapshot
}

func Open(path string) (*Store, error) {
	s := &Store{path: path, data: snapshot{Version: 1, Identities: map[string][]Device{}}}
	bytes, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read device store: %w", err)
	}
	if err := json.Unmarshal(bytes, &s.data); err != nil {
		return nil, fmt.Errorf("decode device store: %w", err)
	}
	if s.data.Version != 1 || s.data.Identities == nil {
		return nil, errors.New("unsupported device store format")
	}
	return s, nil
}

func (s *Store) Upsert(identity string, device Device) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	devices := s.data.Identities[identity]
	found := false
	for i := range devices {
		if devices[i].Token == device.Token {
			devices[i] = device
			found = true
			break
		}
	}
	if !found {
		devices = append(devices, device)
	}
	sort.Slice(devices, func(i, j int) bool { return devices[i].Token < devices[j].Token })
	s.data.Identities[identity] = devices
	return s.persistLocked()
}

func (s *Store) Remove(identity, token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	devices := s.data.Identities[identity]
	kept := devices[:0]
	for _, device := range devices {
		if device.Token != token {
			kept = append(kept, device)
		}
	}
	if len(kept) == 0 {
		delete(s.data.Identities, identity)
	} else {
		s.data.Identities[identity] = kept
	}
	return s.persistLocked()
}

func (s *Store) Devices(identity string) []Device {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Device(nil), s.data.Identities[identity]...)
}

func (s *Store) Count() (identities, devices int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	identities = len(s.data.Identities)
	for _, values := range s.data.Identities {
		devices += len(values)
	}
	return
}

func (s *Store) persistLocked() error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create device store directory: %w", err)
	}
	bytes, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return fmt.Errorf("encode device store: %w", err)
	}
	temporary, err := os.CreateTemp(dir, ".devices-*.json")
	if err != nil {
		return fmt.Errorf("create temporary device store: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(bytes); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryName, s.path); err != nil {
		return fmt.Errorf("replace device store: %w", err)
	}
	return nil
}
