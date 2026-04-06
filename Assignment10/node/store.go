package main

import "sync"

// Entry holds a value and its logical version number.
// Version starts at 0 and increments on every write.
// Version numbers are how we detect stale reads across nodes.
type Entry struct {
	Value   string
	Version int64
}

// Store is a thread-safe in-memory key-value store.
// sync.RWMutex allows multiple concurrent readers but only one writer.
type Store struct {
	mu   sync.RWMutex
	data map[string]Entry
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]Entry),
	}
}

// LocalSet writes directly to this node's memory with the given version.
// Called by the leader/coordinator after receiving a write from a client,
// and also by followers when receiving a replication message.
func (s *Store) LocalSet(key, value string, version int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Only update if incoming version is newer than what we have.
	// This prevents stale replication messages from overwriting newer data.
	if existing, ok := s.data[key]; ok && existing.Version >= version {
		return
	}
	s.data[key] = Entry{Value: value, Version: version}
}

// LocalGet reads directly from this node's memory.
// Returns the Entry and a boolean indicating if the key exists.
func (s *Store) LocalGet(key string) (Entry, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.data[key]
	return e, ok
}

// NextVersion returns the next version number for a key.
// If the key doesn't exist, starts at version 1.
func (s *Store) NextVersion(key string) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.data[key]; ok {
		return e.Version + 1
	}
	return 1
}