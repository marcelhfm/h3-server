package test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"sort"
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
	resolution = 8
	compact    = true
)

func TestPlzH3Index(t *testing.T) {
	file, err := os.Open("plz-5stellig.geojson")
	if err != nil {
		t.Fatalf("Failed to open geoJSON file: %v", err)
	}
	defer file.Close()

	rawBytes, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read geoJSON file %v", err)
	}

	var geojson GeoJSON
	if err := json.Unmarshal(rawBytes, &geojson); err != nil {
		t.Fatalf("Failed to parse GeoJSON: %v", err)
	}

	client := &http.Client{}
	var durations []time.Duration
	passed, failed := 0, 0

	for _, feature := range geojson.Features {
		requestBody := map[string]interface{}{
			"compact":    compact,
			"resolution": resolution,
			"geometries": []typings.GeoJSONGeometry{feature.Geometry},
		}

		requestBodyJSON, err := json.Marshal(requestBody)
		if err != nil {
			t.Errorf("Failed to marshal geometry: %v", err)
			failed++
			continue
		}

		req, err := http.NewRequest("POST", "http://localhost:5005/create-index", bytes.NewBuffer(requestBodyJSON))
		if err != nil {
			t.Errorf("failed to create HTTP request: %v", err)
			failed++
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		start := time.Now()
		resp, err := client.Do(req)
		duration := time.Since(start)

		if err != nil {
			t.Errorf("HTTP request failed: %v", err)
			failed++
			continue
		}
		defer resp.Body.Close()

		durations = append(durations, duration)

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("Failed to read response body: %v", err)
			failed++
			continue
		}

		var responseData ResponseData
		if err := json.Unmarshal(respBody, &responseData); err != nil {
			t.Errorf("Failed to parse response JSON: %v", err)
			failed++
			continue
		}

		if resp.StatusCode == http.StatusOK {
			h3Count := 0
			for _, res := range responseData.Result {
				h3Count += len(res.H3Indices)
			}

			if h3Count > 0 {
				passed++
			} else {
				failed++
				t.Errorf("No H3 indices returned for geometry")
			}
		} else {
			failed++
			t.Errorf("Unexpected status code: %v, Response body: %s", resp.StatusCode, respBody)
		}
	}

	if len(durations) > 0 {
		sort.Slice(durations, func(i, j int) bool { return durations[i] < durations[j] })
		minDuration := durations[0]
		maxDuration := durations[len(durations)-1]
		medianDuration := durations[len(durations)/2]
		totalDuration := time.Duration(0)
		for _, d := range durations {
			totalDuration += d
		}
		avgDuration := totalDuration / time.Duration(len(durations))
		t.Logf("Passed: %d, Failed: %d", passed, failed)
		t.Logf("Min Duration: %v", minDuration)
		t.Logf("Max Duration: %v", maxDuration)
		t.Logf("Median Duration: %v", medianDuration)
		t.Logf("Average Duration: %v", avgDuration)
		t.Logf("Total Duration: %v", totalDuration)
	} else {
		t.Logf("No successful requests to report durations.")
	}
}
