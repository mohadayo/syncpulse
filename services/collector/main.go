package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

type HealthResponse struct {
	Status    string  `json:"status"`
	Service   string  `json:"service"`
	Timestamp float64 `json:"timestamp"`
}

type Target struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type CheckResult struct {
	Target    string  `json:"target"`
	Status    string  `json:"status"`
	Latency   float64 `json:"latency_ms"`
	CheckedAt float64 `json:"checked_at"`
}

var (
	targets   []Target
	results   []CheckResult
	resultsMu sync.RWMutex
	logger    *log.Logger
)

func init() {
	logger = log.New(os.Stdout, "[collector] ", log.LstdFlags)
}

func getPort() string {
	port := os.Getenv("COLLECTOR_PORT")
	if port == "" {
		port = "8002"
	}
	return port
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:    "healthy",
		Service:   "collector",
		Timestamp: float64(time.Now().UnixMilli()) / 1000.0,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func targetsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(targets)
	case http.MethodPost:
		var t Target
		if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
			http.Error(w, `{"error":"invalid JSON body"}`, http.StatusBadRequest)
			return
		}
		if t.Name == "" || t.URL == "" {
			http.Error(w, `{"error":"name and url are required"}`, http.StatusBadRequest)
			return
		}
		targets = append(targets, t)
		logger.Printf("Added target: %s (%s)", t.Name, t.URL)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(t)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func CheckTarget(client *http.Client, t Target) CheckResult {
	start := time.Now()
	result := CheckResult{
		Target:    t.Name,
		CheckedAt: float64(time.Now().UnixMilli()) / 1000.0,
	}

	resp, err := client.Get(t.URL)
	latency := float64(time.Since(start).Microseconds()) / 1000.0
	result.Latency = latency

	if err != nil {
		result.Status = "unreachable"
		logger.Printf("Target %s unreachable: %v", t.Name, err)
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = "healthy"
	} else {
		result.Status = fmt.Sprintf("unhealthy (HTTP %d)", resp.StatusCode)
	}
	logger.Printf("Target %s checked: %s (%.2fms)", t.Name, result.Status, result.Latency)
	return result
}

func checkHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	client := &http.Client{Timeout: 5 * time.Second}
	var newResults []CheckResult
	for _, t := range targets {
		result := CheckTarget(client, t)
		newResults = append(newResults, result)
	}
	resultsMu.Lock()
	results = append(results, newResults...)
	resultsMu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(newResults)
}

func resultsHandler(w http.ResponseWriter, r *http.Request) {
	resultsMu.RLock()
	defer resultsMu.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func main() {
	port := getPort()
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/targets", targetsHandler)
	mux.HandleFunc("/check", checkHandler)
	mux.HandleFunc("/results", resultsHandler)

	logger.Printf("Starting collector on port %s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		logger.Fatalf("Failed to start server: %v", err)
	}
}
