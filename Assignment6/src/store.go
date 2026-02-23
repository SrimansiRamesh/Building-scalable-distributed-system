package main

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"
)

// SearchStore provides thread-safe storage for searchable products using sync.Map.
// sync.Map is optimized for cases where keys are stable (written once, read many times),
// which fits our use case of loading 100K products at startup and then only reading.
type SearchStore struct {
	products sync.Map // map[int]*SearchProduct
	count    int      // total number of products
}

// NewSearchStore creates a new SearchStore and generates numProducts products at startup.
func NewSearchStore(numProducts int) *SearchStore {
	ss := &SearchStore{count: numProducts}
	ss.generateProducts(numProducts)
	return ss
}

// generateProducts creates sample products with predictable, varied data.
func (ss *SearchStore) generateProducts(n int) {
	brands := []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta"}
	categories := []string{"Electronics", "Books", "Home", "Garden", "Sports", "Toys", "Clothing", "Food", "Health", "Automotive"}
	descriptions := []string{
		"High quality product with excellent features",
		"Best seller in its category",
		"Premium grade with warranty included",
		"Affordable and reliable everyday item",
		"Top rated by customers worldwide",
	}

	for i := 1; i <= n; i++ {
		brand := brands[i%len(brands)]
		p := &SearchProduct{
			ID:          i,
			Name:        fmt.Sprintf("Product %s %d", brand, i),
			Category:    categories[i%len(categories)],
			Description: descriptions[i%len(descriptions)],
			Brand:       brand,
		}
		ss.products.Store(i, p)
	}
}

// Search checks exactly maxCheck products for case-insensitive matches in name and category.
// Returns up to maxResults matching products and the total number of matches found.
func (ss *SearchStore) Search(query string, maxCheck int, maxResults int) ([]SearchProduct, int, time.Duration) {
	start := time.Now()
	queryLower := strings.ToLower(query)

	var results []SearchProduct
	totalFound := 0
	checked := 0

	// Iterate through products by ID, checking exactly maxCheck products
	for i := 1; i <= ss.count && checked < maxCheck; i++ {
		val, ok := ss.products.Load(i)
		if !ok {
			continue
		}
		p := val.(*SearchProduct)

		// Count EVERY product checked, not just matches
		checked++

		// Simulate fixed-time computation per product (like an AI model or video processing).
		// Runs multiple SHA-256 rounds to create realistic CPU load per check.
		data := []byte(fmt.Sprintf("%s-%s-%d", p.Name, p.Category, p.ID))
		for r := 0; r < 300; r++ {
			hash := sha256.Sum256(data)
			data = hash[:]
		}

		// Case-insensitive search on name and category
		nameLower := strings.ToLower(p.Name)
		categoryLower := strings.ToLower(p.Category)

		if strings.Contains(nameLower, queryLower) || strings.Contains(categoryLower, queryLower) {
			totalFound++
			if len(results) < maxResults {
				results = append(results, *p)
			}
		}
	}

	elapsed := time.Since(start)
	return results, totalFound, elapsed
}
