package main

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

// Order mirrors the receiver's Order struct.
type Order struct {
	OrderID    string    `json:"order_id"`
	CustomerID int       `json:"customer_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// handler is invoked once per SNS event (may contain multiple records).
// Each SNS record holds one order published by the order-api.
func handler(ctx context.Context, snsEvent events.SNSEvent) error {
	for _, record := range snsEvent.Records {
		var order Order
		if err := json.Unmarshal([]byte(record.SNS.Message), &order); err != nil {
			log.Printf("Failed to parse order: %v", err)
			continue
		}

		log.Printf("Processing order %s (customer %d)", order.OrderID, order.CustomerID)

		// Same 3-second payment verification as the ECS worker.
		time.Sleep(3 * time.Second)

		log.Printf("Completed order %s", order.OrderID)
	}
	return nil
}

func main() {
	lambda.Start(handler)
}
