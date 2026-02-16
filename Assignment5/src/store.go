package main

import "sync"

// Store provides thread-safe in-memory storage for products using a hashmap.
type Store struct {
	mu       sync.RWMutex
	products map[int]*Product
}

// NewStore creates and returns a new empty Store.
func NewStore() *Store {
	return &Store{
		products: make(map[int]*Product),
	}
}

// Get retrieves a product by its ID. Returns the product and true if found,
// or nil and false if not found.
func (s *Store) Get(id int) (*Product, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.products[id]
	return p, ok
}

// Set stores or updates a product by its ID.
func (s *Store) Set(id int, p *Product) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.products[id] = p
}
