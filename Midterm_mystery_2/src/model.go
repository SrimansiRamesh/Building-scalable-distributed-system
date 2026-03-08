package main

// SearchProduct represents a product in the search catalog.
type SearchProduct struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Brand       string `json:"brand"`
}

// SearchResponse is the JSON response from the search endpoint.
type SearchResponse struct {
	Products        []SearchProduct `json:"products"`
	Recommendations []string        `json:"recommendations"`
	TotalFound      int             `json:"total_found"`
	SearchTime      string          `json:"search_time"`
	RecommendStatus string          `json:"recommend_status"` // "ok", "timeout", "circuit_open"
}

// ErrorResponse represents the standard error format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
