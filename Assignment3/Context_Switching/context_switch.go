package main

import (
	"fmt"
	"runtime"
	"time"
)

func pingPong(iterations int) time.Duration {
	// Two channels for ping-pong communication
	ping := make(chan struct{})
	pong := make(chan struct{})
	done := make(chan struct{})

	// Goroutine 1: receives ping, sends pong
	go func() {
		for i := 0; i < iterations; i++ {
			<-ping        // Wait for ping
			pong <- struct{}{} // Send pong
		}
		done <- struct{}{}
	}()

	// Goroutine 2: sends ping, receives pong
	go func() {
		for i := 0; i < iterations; i++ {
			ping <- struct{}{} // Send ping
			<-pong        // Wait for pong
		}
		done <- struct{}{}
	}()

	start := time.Now()

	// Wait for both goroutines to finish
	<-done
	<-done

	return time.Since(start)
}

func main() {
	iterations := 1000000

	fmt.Println("=== CONTEXT SWITCHING EXPERIMENT ===")
	fmt.Printf("Ping-pong iterations: %d\n", iterations)
	fmt.Printf("Total switches: %d (2 per round-trip)\n\n", iterations*2)

	// Test 1: Single OS thread (GOMAXPROCS = 1)
	runtime.GOMAXPROCS(1)
	fmt.Printf("GOMAXPROCS: %d (single OS thread)\n", runtime.GOMAXPROCS(0))
	
	singleThreadTime := pingPong(iterations)
	singleSwitchAvg := float64(singleThreadTime.Nanoseconds()) / float64(iterations*2)
	
	fmt.Printf("Total time:     %v\n", singleThreadTime)
	fmt.Printf("Avg switch:     %.2f ns\n\n", singleSwitchAvg)

	// Test 2: Multiple OS threads (GOMAXPROCS = NumCPU)
	numCPU := runtime.NumCPU()
	runtime.GOMAXPROCS(numCPU)
	fmt.Printf("GOMAXPROCS: %d (multiple OS threads)\n", runtime.GOMAXPROCS(0))
	
	multiThreadTime := pingPong(iterations)
	multiSwitchAvg := float64(multiThreadTime.Nanoseconds()) / float64(iterations*2)
	
	fmt.Printf("Total time:     %v\n", multiThreadTime)
	fmt.Printf("Avg switch:     %.2f ns\n\n", multiSwitchAvg)

	// Comparison
	fmt.Println("=== COMPARISON ===")
	if singleThreadTime < multiThreadTime {
		fmt.Printf("Single thread is %.2fx faster\n", float64(multiThreadTime)/float64(singleThreadTime))
	} else {
		fmt.Printf("Multi thread is %.2fx faster\n", float64(singleThreadTime)/float64(multiThreadTime))
	}
}