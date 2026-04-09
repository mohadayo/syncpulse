package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	healthHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp HealthResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "healthy" {
		t.Errorf("expected healthy, got %s", resp.Status)
	}
	if resp.Service != "collector" {
		t.Errorf("expected collector, got %s", resp.Service)
	}
}

func TestTargetsHandlerPost(t *testing.T) {
	targets = nil
	body := `{"name":"test-svc","url":"http://localhost:8001/health"}`
	req := httptest.NewRequest(http.MethodPost, "/targets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	targetsHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	if targets[0].Name != "test-svc" {
		t.Errorf("expected test-svc, got %s", targets[0].Name)
	}
}

func TestTargetsHandlerPostInvalid(t *testing.T) {
	targets = nil
	body := `{"name":"","url":""}`
	req := httptest.NewRequest(http.MethodPost, "/targets", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	targetsHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTargetsHandlerGet(t *testing.T) {
	targets = []Target{{Name: "svc1", URL: "http://localhost:1234"}}
	req := httptest.NewRequest(http.MethodGet, "/targets", nil)
	w := httptest.NewRecorder()
	targetsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []Target
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1, got %d", len(resp))
	}
}

func TestCheckTarget(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := srv.Client()
	target := Target{Name: "mock-svc", URL: srv.URL}
	result := CheckTarget(client, target)

	if result.Status != "healthy" {
		t.Errorf("expected healthy, got %s", result.Status)
	}
	if result.Target != "mock-svc" {
		t.Errorf("expected mock-svc, got %s", result.Target)
	}
	if result.Latency <= 0 {
		t.Errorf("expected positive latency, got %f", result.Latency)
	}
}

func TestCheckTargetUnhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := srv.Client()
	target := Target{Name: "bad-svc", URL: srv.URL}
	result := CheckTarget(client, target)

	if result.Status != "unhealthy (HTTP 500)" {
		t.Errorf("expected unhealthy status, got %s", result.Status)
	}
}

func TestResultsHandler(t *testing.T) {
	results = []CheckResult{{Target: "t1", Status: "healthy", Latency: 1.5, CheckedAt: 123456}}
	req := httptest.NewRequest(http.MethodGet, "/results", nil)
	w := httptest.NewRecorder()
	resultsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp []CheckResult
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if len(resp) != 1 {
		t.Errorf("expected 1, got %d", len(resp))
	}
}
