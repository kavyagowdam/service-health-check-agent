package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// APIServer provides the HTTP endpoints for the health checker
type APIServer struct {
	checker *Checker
	port    int
	server  *http.Server
}

// NewAPIServer creates a new API server instance
func NewAPIServer(checker *Checker, port int) *APIServer {
	return &APIServer{
		checker: checker,
		port:    port,
	}
}

// Start begins listening for HTTP requests
func (a *APIServer) Start() error {
	mux := http.NewServeMux()
	
	// Register endpoints
	mux.HandleFunc("/health", a.handleHealth)
	mux.HandleFunc("/results", a.handleResults)
	mux.HandleFunc("/result/", a.handleResultByName)
	
	addr := fmt.Sprintf(":%d", a.port)
	a.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	
	fmt.Printf("Starting API server on %s\n", addr)
	return a.server.ListenAndServe()
}

// Stop shuts down the API server
func (a *APIServer) Stop() error {
	if a.server != nil {
		return a.server.Close()
	}
	return nil
}

// handleHealth responds with the service's own health status
func (a *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	health := map[string]interface{}{
		"status":    "UP",
		"timestamp": time.Now().Format(time.RFC3339),
	}
	
	json.NewEncoder(w).Encode(health)
}

// handleResults returns all check results
func (a *APIServer) handleResults(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	results := a.checker.GetResults()
	json.NewEncoder(w).Encode(results)
}

// handleResultByName returns a specific check result by name
func (a *APIServer) handleResultByName(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/result/"):]
	if name == "" {
		http.Error(w, "Service name required", http.StatusBadRequest)
		return
	}
	
	result, found := a.checker.GetResultByName(name)
	if !found {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}