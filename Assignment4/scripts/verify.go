package main

// verify.go - Standalone word counter to verify MapReduce results
// Usage: go run scripts/verify.go <text-file-path-or-s3-url>
// This reads the text file directly and counts words in a single pass,
// then compares against the MapReduce JSON output.

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var wordRegex = regexp.MustCompile(`[^a-zA-Z]+`)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/verify.go <text-file> [mapreduce-result.json]")
		os.Exit(1)
	}

	textFile := os.Args[1]

	// Read the text file
	data, err := os.ReadFile(textFile)
	if err != nil {
		fmt.Printf("Error reading text file: %v\n", err)
		os.Exit(1)
	}

	// Count words (same logic as mapper)
	expected := countWords(string(data))

	fmt.Printf("Direct word count from '%s':\n", textFile)
	fmt.Printf("  Unique words: %d\n", len(expected))
	totalWords := 0
	for _, c := range expected {
		totalWords += c
	}
	fmt.Printf("  Total words:  %d\n", totalWords)
	fmt.Println()

	// Print top 20 words
	fmt.Println("Top 20 words:")
	type wordCount struct {
		Word  string
		Count int
	}
	var sorted []wordCount
	for w, c := range expected {
		sorted = append(sorted, wordCount{w, c})
	}
	// Simple bubble sort for top 20
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Count > sorted[i].Count {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	for i := 0; i < 20 && i < len(sorted); i++ {
		fmt.Printf("  %-20s %d\n", sorted[i].Word, sorted[i].Count)
	}

	// If a MapReduce result JSON is provided, compare
	if len(os.Args) >= 3 {
		resultFile := os.Args[2]
		resultData, err := os.ReadFile(resultFile)
		if err != nil {
			fmt.Printf("\nError reading result file: %v\n", err)
			os.Exit(1)
		}

		var actual map[string]int
		if err := json.Unmarshal(resultData, &actual); err != nil {
			fmt.Printf("\nError parsing result JSON: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nMapReduce result from '%s':\n", resultFile)
		fmt.Printf("  Unique words: %d\n", len(actual))
		totalActual := 0
		for _, c := range actual {
			totalActual += c
		}
		fmt.Printf("  Total words:  %d\n", totalActual)

		// Compare
		fmt.Println("\nVerification:")
		mismatches := 0
		for word, expectedCount := range expected {
			actualCount, exists := actual[word]
			if !exists {
				fmt.Printf("  MISSING: '%s' (expected %d)\n", word, expectedCount)
				mismatches++
			} else if actualCount != expectedCount {
				fmt.Printf("  MISMATCH: '%s' expected=%d actual=%d\n", word, expectedCount, actualCount)
				mismatches++
			}
		}
		for word := range actual {
			if _, exists := expected[word]; !exists {
				fmt.Printf("  EXTRA: '%s' (count=%d)\n", word, actual[word])
				mismatches++
			}
		}

		if mismatches == 0 {
			fmt.Println("  ✓ PERFECT MATCH! MapReduce result is correct.")
		} else {
			fmt.Printf("  ✗ Found %d mismatches\n", mismatches)
		}
	}
}

func countWords(text string) map[string]int {
	counts := make(map[string]int)
	words := strings.Fields(text)
	for _, word := range words {
		cleaned := strings.ToLower(wordRegex.ReplaceAllString(word, ""))
		if cleaned == "" {
			continue
		}
		counts[cleaned]++
	}
	return counts
}