package main

import (
	"fmt"
	"sync"
)

func main() {
	// Regular integer - NOT atomic
	var ops uint64

	var wg sync.WaitGroup

	// Start 50 goroutines, each incrementing 1000 times
	// Expected total: 50 * 1000 = 50,000
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 1000; j++ {
				ops = ops + 1  // RACE CONDITION!
			}
		}()
	}

	wg.Wait()
	fmt.Println("Expected: 50000")
	fmt.Println("Got:     ", ops)
}