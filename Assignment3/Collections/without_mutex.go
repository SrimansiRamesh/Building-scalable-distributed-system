package main

import (
	"fmt"
	"sync"
)

func main() {
	// Plain map - NOT safe for concurrent access
	m := make(map[int]int)

	var wg sync.WaitGroup

	// Spawn 50 goroutines
	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			// Each goroutine writes 1000 entries
			for i := 0; i < 1000; i++ {
				m[g*1000+i] = i  // UNSAFE! Concurrent write to map
			}
		}(g)
	}

	wg.Wait()
	fmt.Println("Expected length: 50000")
	fmt.Println("Actual length:  ", len(m))
}