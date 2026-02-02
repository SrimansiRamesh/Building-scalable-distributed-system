package main

import (
	"fmt"
	"sync"
	"time"
)

// Container holds a map; since we want to update it
// concurrently from multiple goroutines, we add a
// Mutex to synchronize access.
type Container struct {
	mu sync.Mutex
	m  map[int]int
}

func (c *Container) Write(key, value int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = value
}

func main() {
	c := Container{
		m: make(map[int]int),
	}

	var wg sync.WaitGroup

	start := time.Now()

	// Spawn 50 goroutines
	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			// Each goroutine writes 1000 entries
			for i := 0; i < 1000; i++ {
				c.Write(g*1000+i, i)
			}
		}(g)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Println("=== MUTEX PROTECTED MAP ===")
	fmt.Println("Expected length: 50000")
	fmt.Println("Actual length:  ", len(c.m))
	fmt.Println("Time taken:     ", duration)
}