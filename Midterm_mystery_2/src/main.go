package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

const totalProducts = 100_000 // Number of searchable products to generate at startup

func main() {
	log.Printf("Generating %d searchable products...", totalProducts)
	searchStore := NewSearchStore(totalProducts)
	log.Printf("Done. %d products loaded into memory.", totalProducts)

	handler := NewHandler(searchStore)

	r := gin.Default()

	// Health check endpoint (required by ALB target group)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Search endpoint (internally calls recommend service)
	r.GET("/products/search", handler.SearchProducts)

	// Debug endpoint — shows circuit breaker state live
	r.GET("/debug/circuit", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"circuit_breaker_state": handler.cb.State(),
			"protection_enabled":    os.Getenv("PROTECTION_ENABLED") == "true",
		})
	})

	log.Println("Product Search API server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
