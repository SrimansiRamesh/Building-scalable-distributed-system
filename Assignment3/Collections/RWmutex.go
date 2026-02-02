package main

import (
	"fmt"
	"sync"
	"time"
)

// Container with RWMutex instead of Mutex
type Container struct {
	mu sync.RWMutex
	m  map[int]int
}

func (c *Container) Write(key, value int) {
	c.mu.Lock()         // Exclusive lock for writing
	defer c.mu.Unlock()
	c.m[key] = value
}

func (c *Container) Read(key int) (int, bool) {
	c.mu.RLock()        // Shared lock for reading
	defer c.mu.RUnlock()
	value, ok := c.m[key]
	return value, ok
}

func main() {
	c := Container{
		m: make(map[int]int),
	}

	var wg sync.WaitGroup

	start := time.Now()

	// Spawn 50 goroutines - ALL WRITERS
	for g := 0; g < 50; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 1000; i++ {
				c.Write(g*1000+i, i)
			}
		}(g)
	}

	wg.Wait()
	duration := time.Since(start)

	fmt.Println("=== RWMUTEX PROTECTED MAP ===")
	fmt.Println("Expected length: 50000")
	fmt.Println("Actual length:  ", len(c.m))
	fmt.Println("Time taken:     ", duration)
}