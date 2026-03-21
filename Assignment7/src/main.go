package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	// SNS client is optional — only wired up when SNS_TOPIC_ARN is set.
	var snsClient *SNSClient
	if topicARN := os.Getenv("SNS_TOPIC_ARN"); topicARN != "" {
		var err error
		snsClient, err = NewSNSClient(topicARN)
		if err != nil {
			log.Fatalf("Failed to create SNS client: %v", err)
		}
		log.Printf("SNS client ready, topic: %s", topicARN)
	} else {
		log.Println("SNS_TOPIC_ARN not set — /orders/async will return 503")
	}

	h := NewHandler(snsClient)

	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.POST("/orders/sync", h.SyncOrder)
	r.POST("/orders/async", h.AsyncOrder)

	log.Printf("Order API starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
