package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// paymentConcurrency is the max number of simultaneous payment verifications.
// At 3 seconds each: 15 slots / 3s = 5 orders/sec max throughput.
//
// Why a buffered channel instead of time.Sleep alone?
// time.Sleep yields voluntarily — the Go scheduler happily runs hundreds of
// goroutines "concurrently sleeping", so the server handles all requests at once
// (just with 3s latency). That doesn't simulate a real bottleneck.
//
// A send to a FULL buffered channel blocks the goroutine until a slot opens.
// This genuinely caps throughput: once all 15 slots are occupied, new requests
// queue up and response times climb — exactly what happens with a saturated
// payment processor.
const paymentConcurrency = 15

// Handler holds shared state across requests.
type Handler struct {
	paymentSlots chan struct{}
	sns          *SNSClient
}

// NewHandler creates a Handler with the payment semaphore pre-allocated.
// sns may be nil — async endpoint returns 503 if so.
func NewHandler(sns *SNSClient) *Handler {
	return &Handler{
		paymentSlots: make(chan struct{}, paymentConcurrency),
		sns:          sns,
	}
}

// SyncOrder handles POST /orders/sync.
// It blocks until a payment slot is free, simulates 3s verification, then returns.
func (h *Handler) SyncOrder(c *gin.Context) {
	var order Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order.OrderID = fmt.Sprintf("ord-%d", time.Now().UnixNano())
	order.Status = "pending"
	order.CreatedAt = time.Now()

	// Acquire a payment slot — BLOCKS if all 15 are in use.
	h.paymentSlots <- struct{}{}
	defer func() { <-h.paymentSlots }()

	// Simulate payment verification (3 seconds — cannot go faster).
	order.Status = "processing"
	time.Sleep(3 * time.Second)
	order.Status = "completed"

	c.JSON(http.StatusOK, order)
}

// AsyncOrder handles POST /orders/async.
// Publishes the order to SNS and returns 202 immediately — no waiting.
func (h *Handler) AsyncOrder(c *gin.Context) {
	if h.sns == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "async processing not configured"})
		return
	}

	var order Order
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order.OrderID = fmt.Sprintf("ord-%d", time.Now().UnixNano())
	order.Status = "queued"
	order.CreatedAt = time.Now()

	if err := h.sns.Publish(&order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to queue order"})
		return
	}

	// Return immediately — payment happens in the background.
	c.JSON(http.StatusAccepted, order)
}
