package test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/marcelhfm/h3-server/pkg/types"
)

type GeoJSON struct {
	Type     string           `json:"type"`
	Features []GeoJSONFeature `json:"features"`
}

type GeoJSONFeature struct {
	Type       string                  `json:"type"`
	Properties map[string]interface{}  `json:"properties"`
	Geometry   typings.GeoJSONGeometry `json:"geometry"`
}

type ResponseData struct {
	Result []struct {
		Geometry  typings.GeoJSONGeometry `json:"geometry"`
		H3Indices []string                `json:"h3_indices"`
	} `json:"result"`
}

const (
	numWorkers = 8
	resolution = 10
	compact    = true
)

func TestPlzH3Index(t *testing.T) {
	// Load the GeoJSON file
	file, err := os.Open("plz-5stellig.geojson")
	if err != nil {
		t.Fatalf("Failed to open geoJSON file: %v", err)
	}
	defer file.Close()

	rawBytes, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read geoJSON file: %v", err)
	}

	var geojson GeoJSON
	if err := json.Unmarshal(rawBytes, &geojson); err != nil {
		t.Fatalf("Failed to parse GeoJSON: %v", err)
	}

	client := &http.Client{}
	results := make(chan result, len(geojson.Features))

	// Split features into chunks
	features := geojson.Features
	chunkSize := (len(features) + numWorkers - 1) / numWorkers // Ceiling division
	var wg sync.WaitGroup

	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(features) {
			end = len(features)
		}

		wg.Add(1)
		go func(featuresChunk []GeoJSONFeature) {
			defer wg.Done()

			// Process each feature in the chunk
			for _, feature := range featuresChunk {
				// Prepare request body
				requestBody := map[string]interface{}{
					"compact":    compact,
					"resolution": resolution,
					"geometries": []typings.GeoJSONGeometry{feature.Geometry},
				}

				requestBodyJSON, err := json.Marshal(requestBody)
				if err != nil {
					results <- result{err: err}
					continue
				}

				req, err := http.NewRequest("POST", "http://localhost:5005/create-index", bytes.NewBuffer(requestBodyJSON))
				if err != nil {
					results <- result{err: err}
					continue
				}
				req.Header.Set("Content-Type", "application/json")

				// Send request and measure duration
				startTime := time.Now()
				resp, err := client.Do(req)
				duration := time.Since(startTime)

				if err != nil {
					results <- result{err: err, duration: duration}
					continue
				}
				defer resp.Body.Close()

				// Read and parse response
				respBody, err := io.ReadAll(resp.Body)
				if err != nil {
					results <- result{err: err, duration: duration}
					continue
				}

				var responseData ResponseData
				if err := json.Unmarshal(respBody, &responseData); err != nil {
					results <- result{err: err, duration: duration}
					continue
				}

				// Check if the response is successful
				if resp.StatusCode == http.StatusOK {
					// Check if there are h3 indices in the result
					h3Count := 0
					for _, res := range responseData.Result {
						h3Count += len(res.H3Indices)
					}

					if h3Count > 0 {
						results <- result{success: true, duration: duration}
					} else {
						results <- result{err: fmt.Errorf("no H3 indices returned for geometry: %v", feature.Geometry), duration: duration}
					}
				} else {
					results <- result{err: fmt.Errorf("unexpected status code: %v, response body: %s", resp.StatusCode, respBody), duration: duration}
				}
			}
		}(features[start:end])
	}

	// Wait for all workers to finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results and calculate metrics
	passed, failed := 0, 0
	var allDurations []time.Duration

	for res := range results {
		if res.err != nil {
			t.Error(res.err)
			failed++
		} else if res.success {
			passed++
			allDurations = append(allDurations, res.duration)
		} else {
			failed++
		}
	}

	if len(allDurations) > 0 {
		sort.Slice(allDurations, func(i, j int) bool { return allDurations[i] < allDurations[j] })
		minDuration := allDurations[0]
		maxDuration := allDurations[len(allDurations)-1]
		medianDuration := allDurations[len(allDurations)/2]
		totalDuration := time.Duration(0)
		for _, d := range allDurations {
			totalDuration += d
		}
		avgDuration := totalDuration / time.Duration(len(allDurations))

		t.Logf("Passed: %d, Failed: %d", passed, failed)
		t.Logf("Min Duration: %v", minDuration)
		t.Logf("Max Duration: %v", maxDuration)
		t.Logf("Median Duration: %v", medianDuration)
		t.Logf("Average Duration: %v", avgDuration)
	} else {
		t.Logf("No successful requests to report durations.")
	}
}

type result struct {
	success  bool
	err      error
	duration time.Duration
}

