package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"mapreduce-wordcount/shared"
)

var (
	s3Client   *shared.S3Client
	bucketName string
	// Regex to strip non-alphabetic characters (keep only letters)
	wordRegex = regexp.MustCompile(`[^a-zA-Z]+`)
)

// MapperResponse is the JSON response returned by the mapper
type MapperResponse struct {
	OutputURL  string `json:"output_url"`
	WordCount  int    `json:"unique_words"`
	TotalWords int    `json:"total_words"`
	Message    string `json:"message"`
}

// mapHandler handles GET /map?url=s3://bucket/chunk.txt
func mapHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()

	// Get the S3 URL of the chunk to process
	chunkURL := r.URL.Query().Get("url")
	if chunkURL == "" {
		http.Error(w, `{"error": "missing 'url' query parameter"}`, http.StatusBadRequest)
		return
	}

	log.Printf("Mapping chunk: %s", chunkURL)

	// 1. Read the chunk from S3
	startTime := time.Now()
	content, err := s3Client.ReadFromS3(ctx, chunkURL)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "failed to read chunk from S3: %v"}`, err), http.StatusInternalServerError)
		return
	}
	log.Printf("Read chunk (%d bytes) in %v", len(content), time.Since(startTime))

	// 2. Count word occurrences
	startTime = time.Now()
	wordCounts := countWords(content)
	totalWords := 0
	for _, count := range wordCounts {
		totalWords += count
	}
	log.Printf("Counted %d unique words (%d total) in %v", len(wordCounts), totalWords, time.Since(startTime))

	// 3. Save results as JSON to S3
	jsonData, err := json.Marshal(wordCounts)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "failed to marshal word counts: %v"}`, err), http.StatusInternalServerError)
		return
	}

	timestamp := time.Now().UnixNano()
	key := fmt.Sprintf("mapped/map_result_%d.json", timestamp)
	outputURL, err := s3Client.WriteToS3(ctx, bucketName, key, string(jsonData), "application/json")
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "failed to write results to S3: %v"}`, err), http.StatusInternalServerError)
		return
	}

	log.Printf("Wrote map results to %s", outputURL)

	// 4. Return the output URL
	resp := MapperResponse{
		OutputURL:  outputURL,
		WordCount:  len(wordCounts),
		TotalWords: totalWords,
		Message:    "Mapping completed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// countWords takes text and returns a map of word -> count
// Words are lowercased and stripped of non-alphabetic characters
func countWords(text string) map[string]int {
	counts := make(map[string]int)

	words := strings.Fields(text)
	for _, word := range words {
		// Clean: lowercase and remove non-alpha characters
		cleaned := strings.ToLower(wordRegex.ReplaceAllString(word, ""))
		if cleaned == "" {
			continue
		}
		counts[cleaned]++
	}

	return counts
}

// healthHandler returns service health status
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "healthy", "service": "mapper"}`))
}

func main() {
	// Configuration from environment variables
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "us-east-1"
	}

	bucketName = os.Getenv("S3_BUCKET")
	if bucketName == "" {
		log.Fatal("S3_BUCKET environment variable is required")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// Initialize S3 client
	var err error
	s3Client, err = shared.NewS3Client(region)
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}

	// Set up routes
	http.HandleFunc("/map", mapHandler)
	http.HandleFunc("/health", healthHandler)

	log.Printf("Mapper service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}