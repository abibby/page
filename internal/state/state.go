// Package state tracks which torrents have already been imported so the daemon
// does not re-add them on every poll.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store is a JSON-backed set of processed torrent hashes.
type Store struct {
	path string
	mu   sync.Mutex
	data map[string]string // hash -> RFC3339 import time
}

// Load reads the state file, creating an empty store if it does not exist.
func Load(path string) (*Store, error) {
	s := &Store{path: path, data: map[string]string{}}
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return nil, err
	}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &s.data); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// Has reports whether a torrent hash has already been processed.
func (s *Store) Has(hash string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.data[hash]
	return ok
}

// Mark records a torrent hash as processed and persists the store atomically.
func (s *Store) Mark(hash, when string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if when == "" {
		when = time.Now().UTC().Format(time.RFC3339)
	}
	s.data[hash] = when
	return s.save()
}

func (s *Store) save() error {
	b, err := json.MarshalIndent(s.data, "", "  ")
	if err != nil {
		return err
	}
	if dir := filepath.Dir(s.path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
