package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Order mirrors the receiver's Order struct for JSON unmarshalling.
type Order struct {
	OrderID    string    `json:"order_id"`
	CustomerID int       `json:"customer_id"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
}

// snsEnvelope wraps the actual message payload when SNS delivers to SQS.
type snsEnvelope struct {
	Message string `json:"Message"`
}

func main() {
	queueURL := os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		log.Fatal("SQS_QUEUE_URL environment variable is required")
	}

	workerCount := 1
	if v := os.Getenv("WORKER_COUNT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			log.Fatalf("Invalid WORKER_COUNT: %s", v)
		}
		workerCount = n
	}
	log.Printf("Starting with %d worker goroutine(s) — max throughput: %.2f orders/sec",
		workerCount, float64(workerCount)/3.0)

	cfg, err := config.LoadDefaultConfig(context.Background())
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}
	client := sqs.NewFromConfig(cfg)

	// Health endpoint — ECS needs this to confirm the task is alive.
	go func() {
		http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})
		log.Println("Health server listening on :8080")
		if err := http.ListenAndServe(":8080", nil); err != nil {
			log.Fatalf("Health server failed: %v", err)
		}
	}()

	log.Println("Order processor started, polling SQS...")
	poll(client, queueURL, workerCount)
}

// poll continuously receives messages from SQS.
// A semaphore channel limits concurrent processing goroutines to workerCount.
func poll(client *sqs.Client, queueURL string, workerCount int) {
	// semaphore: only workerCount goroutines may process simultaneously.
	// When full, the poll loop blocks — which also stops fetching new messages.
	sem := make(chan struct{}, workerCount)

	for {
		result, err := client.ReceiveMessage(context.Background(), &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     20, // long polling
		})
		if err != nil {
			log.Printf("ReceiveMessage error: %v", err)
			continue
		}

		for _, msg := range result.Messages {
			sem <- struct{}{} // acquire slot — blocks if workerCount goroutines are busy
			go func(m types.Message) {
				defer func() { <-sem }() // release slot when done
				process(client, queueURL, m)
			}(msg)
		}
	}
}

// process handles a single SQS message: unwrap SNS envelope → verify payment → delete.
func process(client *sqs.Client, queueURL string, msg types.Message) {
	// SQS messages from SNS are wrapped in an envelope: {"Message": "<json>"}
	var envelope snsEnvelope
	if err := json.Unmarshal([]byte(*msg.Body), &envelope); err != nil {
		log.Printf("Failed to parse SNS envelope: %v", err)
		return
	}

	var order Order
	if err := json.Unmarshal([]byte(envelope.Message), &order); err != nil {
		log.Printf("Failed to parse order JSON: %v", err)
		return
	}

	log.Printf("Processing order %s (customer %d)", order.OrderID, order.CustomerID)

	// Simulate payment verification — same 3s as the sync endpoint.
	time.Sleep(3 * time.Second)

	log.Printf("Completed order %s", order.OrderID)

	// Delete from queue so it isn't redelivered.
	_, err := client.DeleteMessage(context.Background(), &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queueURL),
		ReceiptHandle: msg.ReceiptHandle,
	})
	if err != nil {
		log.Printf("Failed to delete message %s: %v", *msg.MessageId, err)
	}
}
