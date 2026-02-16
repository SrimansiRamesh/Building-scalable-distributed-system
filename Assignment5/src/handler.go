package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// Handler holds the dependencies for HTTP handlers.
type Handler struct {
	store *Store
}

// NewHandler creates a new Handler with the given store.
func NewHandler(store *Store) *Handler {
	return &Handler{store: store}
}

// GetProduct handles GET /products/:productId
// Returns 200 with the product, or 404 if not found.
func (h *Handler) GetProduct(c *gin.Context) {
	id, err := parseProductID(c.Param("productId"))
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "NOT_FOUND",
			Message: "Product not found",
		})
		return
	}

	product, ok := h.store.Get(id)
	if !ok {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "NOT_FOUND",
			Message: "Product not found",
			Details: fmt.Sprintf("No product found with ID %d", id),
		})
		return
	}

	c.JSON(http.StatusOK, product)
}

// AddProductDetails handles POST /products/:productId/details
// Validates input, stores the product, and returns 204 on success.
func (h *Handler) AddProductDetails(c *gin.Context) {
	id, err := parseProductID(c.Param("productId"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid product ID in path",
			Details: "Product ID must be a positive integer",
		})
		return
	}

	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid request body",
			Details: err.Error(),
		})
		return
	}

	// Validate all fields per OpenAPI spec constraints
	if errs := validateProduct(&product); len(errs) > 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Validation failed",
			Details: strings.Join(errs, "; "),
		})
		return
	}

	// Ensure the product_id in the body matches the path parameter
	if product.ProductID != id {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Product ID mismatch",
			Details: "Product ID in URL does not match product_id in request body",
		})
		return
	}

	h.store.Set(id, &product)
	c.Status(http.StatusNoContent)
}

// parseProductID converts a path parameter string to a valid product ID.
// Returns an error if the string is not a positive integer.
func parseProductID(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid product ID: %s", s)
	}
	if id < 1 {
		return 0, fmt.Errorf("product ID must be >= 1, got %d", id)
	}
	return id, nil
}

// validateProduct checks all field constraints defined in the OpenAPI spec.
func validateProduct(p *Product) []string {
	var errs []string

	if p.ProductID < 1 {
		errs = append(errs, "product_id must be at least 1")
	}
	if len(p.SKU) == 0 {
		errs = append(errs, "sku is required")
	} else if len(p.SKU) > 100 {
		errs = append(errs, "sku must be at most 100 characters")
	}
	if len(p.Manufacturer) == 0 {
		errs = append(errs, "manufacturer is required")
	} else if len(p.Manufacturer) > 200 {
		errs = append(errs, "manufacturer must be at most 200 characters")
	}
	if p.CategoryID < 1 {
		errs = append(errs, "category_id must be at least 1")
	}
	if p.Weight < 0 {
		errs = append(errs, "weight must be at least 0")
	}
	if p.SomeOtherID < 1 {
		errs = append(errs, "some_other_id must be at least 1")
	}

	return errs
}
