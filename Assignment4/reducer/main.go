package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"mapreduce-wordcount/shared"
)

var (
	s3Client   *shared.S3Client
	bucketName string
)

// ReducerResponse is the JSON response returned by the reducer
type ReducerResponse struct {
	OutputURL   string `json:"output_url"`
	UniqueWords int    `json:"unique_words"`
	TotalWords  int    `json:"total_words"`
	Message     string `json:"message"`
}

// reduceHandler handles GET /reduce?urls=s3://url1,s3://url2,s3://url3
func reduceHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()

	// Get the comma-separated mapper output URLs
	urlsParam := r.URL.Query().Get("urls")
	if urlsParam == "" {
		http.Error(w, `{"error": "missing 'urls' query parameter (comma-separated S3 URLs)"}`, http.StatusBadRequest)
		return
	}

	mapperURLs := strings.Split(urlsParam, ",")
	log.Printf("Reducing %d mapper outputs", len(mapperURLs))

	// 1. Read and aggregate all mapper results
	finalCounts := make(map[string]int)

	for i, mapperURL := range mapperURLs {
		mapperURL = strings.TrimSpace(mapperURL)
		log.Printf("Reading mapper output %d: %s", i, mapperURL)

		content, err := s3Client.ReadFromS3(ctx, mapperURL)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "failed to read mapper output %d: %v"}`, i, err), http.StatusInternalServerError)
			return
		}

		// Parse the JSON word counts
		var wordCounts map[string]int
		if err := json.Unmarshal([]byte(content), &wordCounts); err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "failed to parse mapper output %d: %v"}`, i, err), http.StatusInternalServerError)
			return
		}

		// Aggregate: add counts to final map
		for word, count := range wordCounts {
			finalCounts[word] += count
		}

		log.Printf("Mapper %d: %d unique words merged", i, len(wordCounts))
	}

	// 2. Compute totals
	totalWords := 0
	for _, count := range finalCounts {
		totalWords += count
	}
	log.Printf("Final result: %d unique words, %d total words", len(finalCounts), totalWords)

	// 3. Save final result to S3
	jsonData, err := json.Marshal(finalCounts)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "failed to marshal final counts: %v"}`, err), http.StatusInternalServerError)
		return
	}

	timestamp := time.Now().Unix()
	key := fmt.Sprintf("results/final_wordcount_%d.json", timestamp)
	outputURL, err := s3Client.WriteToS3(ctx, bucketName, key, string(jsonData), "application/json")
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "failed to write final results to S3: %v"}`, err), http.StatusInternalServerError)
		return
	}

	log.Printf("Wrote final results to %s", outputURL)

	// 4. Return the final output URL
	resp := ReducerResponse{
		OutputURL:   outputURL,
		UniqueWords: len(finalCounts),
		TotalWords:  totalWords,
		Message:     "Reduce completed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// healthHandler returns service health status
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "healthy", "service": "reducer"}`))
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
		port = "8082"
	}

	// Initialize S3 client
	var err error
	s3Client, err = shared.NewS3Client(region)
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}

	// Set up routes
	http.HandleFunc("/reduce", reduceHandler)
	http.HandleFunc("/health", healthHandler)

	log.Printf("Reducer service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}