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
	Products   []SearchProduct `json:"products"`
	TotalFound int             `json:"total_found"`
	SearchTime string          `json:"search_time"`
}

// ErrorResponse represents the standard error format.
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}
