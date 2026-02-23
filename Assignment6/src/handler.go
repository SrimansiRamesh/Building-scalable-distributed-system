package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	searchMaxCheck   = 100 // Check exactly 100 products per search (bounded iteration)
	searchMaxResults = 20  // Return at most 20 matching products
)

// Handler holds the dependencies for HTTP handlers.
type Handler struct {
	searchStore *SearchStore
}

// NewHandler creates a new Handler with the given search store.
func NewHandler(searchStore *SearchStore) *Handler {
	return &Handler{searchStore: searchStore}
}

// SearchProducts handles GET /products/search?q={query}
// Searches name and category fields with bounded iteration (checks 100 products).
// Returns up to 20 matching results with total match count and search time.
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

	results, totalFound, elapsed := h.searchStore.Search(query, searchMaxCheck, searchMaxResults)

	c.JSON(http.StatusOK, SearchResponse{
		Products:   results,
		TotalFound: totalFound,
		SearchTime: elapsed.String(),
	})
}
