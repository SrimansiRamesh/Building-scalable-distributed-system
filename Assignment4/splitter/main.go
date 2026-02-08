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

// SplitterResponse is the JSON response returned by the splitter
type SplitterResponse struct {
	ChunkURLs []string `json:"chunk_urls"`
	Message   string   `json:"message"`
}

// splitHandler handles GET /split?url=s3://bucket/key&chunks=3
func splitHandler(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()

	// Get the S3 URL of the input text file
	inputURL := r.URL.Query().Get("url")
	if inputURL == "" {
		http.Error(w, `{"error": "missing 'url' query parameter"}`, http.StatusBadRequest)
		return
	}

	// Number of chunks (default 3)
	numChunks := 3
	chunksParam := r.URL.Query().Get("chunks")
	if chunksParam != "" {
		fmt.Sscanf(chunksParam, "%d", &numChunks)
		if numChunks < 1 {
			numChunks = 3
		}
	}

	log.Printf("Splitting %s into %d chunks", inputURL, numChunks)

	// 1. Read the text file from S3
	startTime := time.Now()
	content, err := s3Client.ReadFromS3(ctx, inputURL)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error": "failed to read from S3: %v"}`, err), http.StatusInternalServerError)
		return
	}
	log.Printf("Read %d bytes from S3 in %v", len(content), time.Since(startTime))

	// 2. Split content into roughly equal chunks at word boundaries
	chunks := splitIntoChunks(content, numChunks)

	// 3. Upload each chunk to S3
	timestamp := time.Now().Unix()
	chunkURLs := make([]string, 0, len(chunks))

	for i, chunk := range chunks {
		key := fmt.Sprintf("chunks/chunk_%d_%d.txt", timestamp, i)
		chunkURL, err := s3Client.WriteToS3(ctx, bucketName, key, chunk, "text/plain")
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error": "failed to write chunk %d to S3: %v"}`, i, err), http.StatusInternalServerError)
			return
		}
		chunkURLs = append(chunkURLs, chunkURL)
		log.Printf("Uploaded chunk %d (%d bytes) to %s", i, len(chunk), chunkURL)
	}

	// 4. Return the chunk URLs
	resp := SplitterResponse{
		ChunkURLs: chunkURLs,
		Message:   fmt.Sprintf("Split into %d chunks successfully", len(chunks)),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// splitIntoChunks splits text into n roughly equal parts at word boundaries
func splitIntoChunks(text string, n int) []string {
	if n <= 1 {
		return []string{text}
	}

	// Split by whitespace to get words
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	totalWords := len(words)
	chunkSize := totalWords / n
	remainder := totalWords % n

	chunks := make([]string, 0, n)
	start := 0

	for i := 0; i < n; i++ {
		end := start + chunkSize
		// Distribute remainder words across first chunks
		if i < remainder {
			end++
		}
		if end > totalWords {
			end = totalWords
		}
		if start >= totalWords {
			break
		}

		chunk := strings.Join(words[start:end], " ")
		chunks = append(chunks, chunk)
		start = end
	}

	return chunks
}

// healthHandler returns service health status
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "healthy", "service": "splitter"}`))
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
		port = "8080"
	}

	// Initialize S3 client
	var err error
	s3Client, err = shared.NewS3Client(region)
	if err != nil {
		log.Fatalf("Failed to create S3 client: %v", err)
	}

	// Set up routes
	http.HandleFunc("/split", splitHandler)
	http.HandleFunc("/health", healthHandler)

	log.Printf("Splitter service starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}