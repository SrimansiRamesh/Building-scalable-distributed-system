package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// SplitterResponse represents the splitter's JSON response
type SplitterResponse struct {
	ChunkURLs []string `json:"chunk_urls"`
	Message   string   `json:"message"`
}

// MapperResponse represents a mapper's JSON response
type MapperResponse struct {
	OutputURL  string `json:"output_url"`
	WordCount  int    `json:"unique_words"`
	TotalWords int    `json:"total_words"`
	Message    string `json:"message"`
}

// ReducerResponse represents the reducer's JSON response
type ReducerResponse struct {
	OutputURL   string `json:"output_url"`
	UniqueWords int    `json:"unique_words"`
	TotalWords  int    `json:"total_words"`
	Message     string `json:"message"`
}

// TimingResult stores timing for performance analysis
type TimingResult struct {
	Phase    string        `json:"phase"`
	Duration time.Duration `json:"duration_ns"`
	DurationMs float64    `json:"duration_ms"`
}

func main() {
	// Service addresses (use ECS public IPs or localhost for testing)
	splitterAddr := getEnvOrDefault("SPLITTER_ADDR", "http://localhost:8080")
	reducerAddr := getEnvOrDefault("REDUCER_ADDR", "http://localhost:8082")

	// Mapper addresses (comma-separated for multiple mappers)
	mapperAddrsStr := getEnvOrDefault("MAPPER_ADDRS", "http://localhost:8081,http://localhost:8081,http://localhost:8081")
	mapperAddrs := strings.Split(mapperAddrsStr, ",")

	// S3 URL of the input text file
	inputURL := getEnvOrDefault("INPUT_URL", "")
	if inputURL == "" {
		if len(os.Args) > 1 {
			inputURL = os.Args[1]
		} else {
			log.Fatal("INPUT_URL env var or command line argument required")
		}
	}

	numChunks := getEnvOrDefault("NUM_CHUNKS", "3")

	var timings []TimingResult

	fmt.Println("========================================")
	fmt.Println("  MapReduce Word Count Orchestrator")
	fmt.Println("========================================")
	fmt.Printf("Input: %s\n", inputURL)
	fmt.Printf("Chunks: %s\n", numChunks)
	fmt.Printf("Mappers: %d\n", len(mapperAddrs))
	fmt.Println()

	// ---- STEP 1: Call Splitter ----
	fmt.Println("[1/3] Splitting input file...")
	splitStart := time.Now()

	splitResp, err := callSplitter(splitterAddr, inputURL, numChunks)
	if err != nil {
		log.Fatalf("Splitter failed: %v", err)
	}

	splitDuration := time.Since(splitStart)
	timings = append(timings, TimingResult{Phase: "split", Duration: splitDuration, DurationMs: float64(splitDuration.Milliseconds())})

	fmt.Printf("  ✓ Split into %d chunks in %v\n", len(splitResp.ChunkURLs), splitDuration)
	for i, u := range splitResp.ChunkURLs {
		fmt.Printf("    Chunk %d: %s\n", i, u)
	}
	fmt.Println()

	// ---- STEP 2: Call Mappers (in parallel) ----
	fmt.Println("[2/3] Mapping chunks in parallel...")
	mapStart := time.Now()

	mapperOutputs, err := callMappersParallel(mapperAddrs, splitResp.ChunkURLs)
	if err != nil {
		log.Fatalf("Mapping failed: %v", err)
	}

	mapDuration := time.Since(mapStart)
	timings = append(timings, TimingResult{Phase: "map", Duration: mapDuration, DurationMs: float64(mapDuration.Milliseconds())})

	fmt.Printf("  ✓ All %d mappers completed in %v\n", len(mapperOutputs), mapDuration)
	mapperURLs := make([]string, len(mapperOutputs))
	for i, mr := range mapperOutputs {
		mapperURLs[i] = mr.OutputURL
		fmt.Printf("    Mapper %d: %d unique / %d total words → %s\n", i, mr.WordCount, mr.TotalWords, mr.OutputURL)
	}
	fmt.Println()

	// ---- STEP 3: Call Reducer ----
	fmt.Println("[3/3] Reducing mapper outputs...")
	reduceStart := time.Now()

	reduceResp, err := callReducer(reducerAddr, mapperURLs)
	if err != nil {
		log.Fatalf("Reducer failed: %v", err)
	}

	reduceDuration := time.Since(reduceStart)
	timings = append(timings, TimingResult{Phase: "reduce", Duration: reduceDuration, DurationMs: float64(reduceDuration.Milliseconds())})

	fmt.Printf("  ✓ Reduce completed in %v\n", reduceDuration)
	fmt.Printf("    Final: %d unique words, %d total words\n", reduceResp.UniqueWords, reduceResp.TotalWords)
	fmt.Printf("    Output: %s\n", reduceResp.OutputURL)
	fmt.Println()

	// ---- Summary ----
	totalDuration := splitDuration + mapDuration + reduceDuration
	timings = append(timings, TimingResult{Phase: "total", Duration: totalDuration, DurationMs: float64(totalDuration.Milliseconds())})

	fmt.Println("========================================")
	fmt.Println("  Performance Summary")
	fmt.Println("========================================")
	fmt.Printf("  Split:  %v (%.1f%%)\n", splitDuration, float64(splitDuration)/float64(totalDuration)*100)
	fmt.Printf("  Map:    %v (%.1f%%)\n", mapDuration, float64(mapDuration)/float64(totalDuration)*100)
	fmt.Printf("  Reduce: %v (%.1f%%)\n", reduceDuration, float64(reduceDuration)/float64(totalDuration)*100)
	fmt.Printf("  Total:  %v\n", totalDuration)
	fmt.Println("========================================")

	// Save timings to JSON for plotting
	timingsJSON, _ := json.MarshalIndent(timings, "", "  ")
	os.WriteFile("timings.json", timingsJSON, 0644)
	fmt.Println("\nTimings saved to timings.json")
}

func callSplitter(addr, inputURL, chunks string) (*SplitterResponse, error) {
	reqURL := fmt.Sprintf("%s/split?url=%s&chunks=%s", addr, url.QueryEscape(inputURL), chunks)

	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("splitter returned %d: %s", resp.StatusCode, string(body))
	}

	var result SplitterResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func callMappersParallel(mapperAddrs []string, chunkURLs []string) ([]MapperResponse, error) {
	results := make([]MapperResponse, len(chunkURLs))
	var mu sync.Mutex
	var wg sync.WaitGroup
	var firstErr error

	for i, chunkURL := range chunkURLs {
		wg.Add(1)
		go func(idx int, chunk string) {
			defer wg.Done()

			// Use mapper address based on index (round-robin if fewer mappers than chunks)
			mapperAddr := mapperAddrs[idx%len(mapperAddrs)]
			reqURL := fmt.Sprintf("%s/map?url=%s", mapperAddr, url.QueryEscape(chunk))

			resp, err := http.Get(reqURL)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("mapper %d failed: %w", idx, err)
				}
				mu.Unlock()
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("mapper %d read failed: %w", idx, err)
				}
				mu.Unlock()
				return
			}

			if resp.StatusCode != http.StatusOK {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("mapper %d returned %d: %s", idx, resp.StatusCode, string(body))
				}
				mu.Unlock()
				return
			}

			var result MapperResponse
			if err := json.Unmarshal(body, &result); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("mapper %d parse failed: %w", idx, err)
				}
				mu.Unlock()
				return
			}

			mu.Lock()
			results[idx] = result
			mu.Unlock()
		}(i, chunkURL)
	}

	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	return results, nil
}

func callReducer(addr string, mapperURLs []string) (*ReducerResponse, error) {
	urlsParam := strings.Join(mapperURLs, ",")
	reqURL := fmt.Sprintf("%s/reduce?urls=%s", addr, url.QueryEscape(urlsParam))

	resp, err := http.Get(reqURL)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("reducer returned %d: %s", resp.StatusCode, string(body))
	}

	var result ReducerResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
