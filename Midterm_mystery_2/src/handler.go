package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	searchMaxCheck   = 100 // Check exactly 100 products per search (bounded iteration)
	searchMaxResults = 20  // Return at most 20 matching products
	recommendTimeout = 500 * time.Millisecond // Fail fast after 500ms
)

// Handler holds the dependencies for HTTP handlers.
type Handler struct {
	searchStore *SearchStore
	cb          *CircuitBreaker
}

// NewHandler creates a new Handler with the given search store.
func NewHandler(searchStore *SearchStore) *Handler {
	return &Handler{
		searchStore: searchStore,
		// Circuit breaker: open after 5 consecutive failures, reset after 30s
		cb: NewCircuitBreaker(5, 30*time.Second),
	}
}

// SearchProducts handles GET /products/search?q={query}
// Phase 1 (PROTECTION_ENABLED=false): calls recommend with NO timeout or circuit breaker → crashes
// Phase 2 (PROTECTION_ENABLED=true):  calls recommend with timeout + circuit breaker → recovers
func (h *Handler) SearchProducts(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Missing search query",
			Details: "Query parameter 'q' is required",
		})
		return
	}

	// Step 1: fast bounded search (always runs)
	results, totalFound, elapsed := h.searchStore.Search(query, searchMaxCheck, searchMaxResults)

	// Collect product IDs to pass to recommend service
	productIDs := make([]int, len(results))
	for i, p := range results {
		productIDs[i] = p.ID
	}

	// Step 2: call recommend service
	var recs []string
	var recStatus string

	protectionEnabled := os.Getenv("PROTECTION_ENABLED") == "true"

	if protectionEnabled {
		// ── FIXED: fail fast timeout + circuit breaker ──────────────────
		ctx, cancel := context.WithTimeout(c.Request.Context(), recommendTimeout)
		defer cancel()
		recs, recStatus = getRecommendationsProtected(ctx, h.cb, productIDs)
	} else {
		// ── BROKEN: no timeout, no circuit breaker ──────────────────────
		// If recommend is slow, this entire handler blocks.
		// 20 concurrent users = 20 goroutines hanging here simultaneously.
		var err error
		recs, err = getRecommendationsFlaky(productIDs)
		if err != nil {
			recStatus = "error"
		} else {
			recStatus = "ok"
		}
	}

	c.JSON(http.StatusOK, SearchResponse{
		Products:        results,
		Recommendations: recs,
		TotalFound:      totalFound,
		SearchTime:      elapsed.String(),
		RecommendStatus: recStatus,
	})
}
