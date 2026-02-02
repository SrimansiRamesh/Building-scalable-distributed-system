package main

import (
	"bufio"
	"fmt"
	"os"
	"time"
)

func unbufferedWrite(filename string, iterations int) time.Duration {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	start := time.Now()

	for i := 0; i < iterations; i++ {
		f.Write([]byte(fmt.Sprintf("Line %d: This is some test data\n", i)))
	}

	f.Close()
	return time.Since(start)
}

func bufferedWrite(filename string, iterations int) time.Duration {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}

	w := bufio.NewWriter(f)

	start := time.Now()

	for i := 0; i < iterations; i++ {
		w.WriteString(fmt.Sprintf("Line %d: This is some test data\n", i))
	}

	w.Flush()
	f.Close()
	return time.Since(start)
}

func main() {
	iterations := 100000

	fmt.Println("=== FILE ACCESS EXPERIMENT ===")
	fmt.Printf("Writing %d lines to file\n\n", iterations)

	unbufferedTime := unbufferedWrite("unbuffered_output.txt", iterations)
	bufferedTime := bufferedWrite("buffered_output.txt", iterations)

	fmt.Printf("Unbuffered: %v\n", unbufferedTime)
	fmt.Printf("Buffered:   %v\n", bufferedTime)
}