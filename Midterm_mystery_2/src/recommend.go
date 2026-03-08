package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ── Flaky Recommendation Service ─────────────────────────────────────────────
// Simulates a slow/unreliable downstream ML recommendation service.
// In a real system this would be an HTTP call to a separate service.

func getRecommendationsFlaky(productIDs []int) ([]string, error) {
	// Randomly sleep between 2-10 seconds to simulate a slow ML model
	delay := time.Duration(2+rand.Intn(8)) * time.Second
	time.Sleep(delay)

	// Also randomly fail 30% of the time
	if rand.Float32() < 0.3 {
		return nil, fmt.Errorf("recommendation service unavailable")
	}

	// Return fake recommendations
	return []string{
		"Product Alpha 1",
		"Product Beta 42",
		"Product Gamma 7",
	}, nil
}

// ── Circuit Breaker ───────────────────────────────────────────────────────────
// Tracks failures and opens the circuit after maxFailures consecutive failures.
// After resetTimeout, moves to HALF_OPEN and tries one request.
// If successful, closes the circuit. If not, reopens it.

type cbState int

const (
	cbClosed   cbState = iota // normal — calls go through
	cbOpen                    // tripped — calls fail immediately
	cbHalfOpen                // testing — one call allowed through
)

type CircuitBreaker struct {
	mu              sync.Mutex
	state           cbState
	failures        int
	maxFailures     int
	resetTimeout    time.Duration
	lastFailureTime time.Time
}

func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:        cbClosed,
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
	}
}

// Allow checks whether a call should be allowed through.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case cbClosed:
		return true
	case cbOpen:
		// Check if cooldown has passed — move to half-open
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = cbHalfOpen
			return true
		}
		return false
	case cbHalfOpen:
		// Only allow one trial request
		return true
	}
	return false
}

// RecordSuccess resets the circuit breaker on a successful call.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.state = cbClosed
	cb.failures = 0
}

// RecordFailure increments failure count and opens circuit if threshold reached.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	cb.lastFailureTime = time.Now()
	if cb.failures >= cb.maxFailures {
		cb.state = cbOpen
	}
}

func (cb *CircuitBreaker) State() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	switch cb.state {
	case cbClosed:
		return "closed"
	case cbOpen:
		return "open"
	case cbHalfOpen:
		return "half_open"
	}
	return "unknown"
}

// ── Protected Recommend Call ──────────────────────────────────────────────────
// Wraps the flaky service with:
//  1. Circuit breaker — skip the call entirely if circuit is open
//  2. Fail fast (timeout) — give up after 500ms instead of waiting forever

func getRecommendationsProtected(ctx context.Context, cb *CircuitBreaker, productIDs []int) ([]string, string) {
	// Circuit breaker check — fail immediately if circuit is open
	if !cb.Allow() {
		return nil, "circuit_open"
	}

	// Run the flaky call in a goroutine with a 500ms timeout
	type result struct {
		recs []string
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		recs, err := getRecommendationsFlaky(productIDs)
		ch <- result{recs, err}
	}()

	select {
	case <-ctx.Done():
		// Context timed out (fail fast)
		cb.RecordFailure()
		return nil, "timeout"
	case res := <-ch:
		if res.err != nil {
			cb.RecordFailure()
			return nil, "error"
		}
		cb.RecordSuccess()
		return res.recs, "ok"
	}
}
