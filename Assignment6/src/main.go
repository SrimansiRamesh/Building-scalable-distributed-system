package main

import (
	"log"
	"net/http"

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

	// Search endpoint
	r.GET("/products/search", handler.SearchProducts)

	log.Println("Product Search API server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}