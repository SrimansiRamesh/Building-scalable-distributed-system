package main

import (
	"log"

	"github.com/gin-gonic/gin"
)

func main() {
	store := NewStore()
	handler := NewHandler(store)

	r := gin.Default()

	// Product API endpoints (per OpenAPI spec)
	r.GET("/products/:productId", handler.GetProduct)
	r.POST("/products/:productId/details", handler.AddProductDetails)

	log.Println("Product API server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
