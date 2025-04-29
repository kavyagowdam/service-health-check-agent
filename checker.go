package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
    // Add these imports
    "k8s.io/client-go/kubernetes"
    "k8s.io/client-go/rest"
    "k8s.io/client-go/tools/clientcmd"	
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Name      string
	Type      string
	Target    string
	Status    string // "UP", "DOWN", "UNKNOWN"
	Message   string
	Timestamp time.Time
	Duration  time.Duration
}

// Checker is responsible for checking services and storing results
type Checker struct {
	config    *Config
	results   map[string]CheckResult
	client    *http.Client
	mutex     sync.RWMutex
	cancelFns map[string]context.CancelFunc
}

// NewChecker creates a new Checker instance
func NewChecker(config *Config) *Checker {
	return &Checker{
		config:    config,
		results:   make(map[string]CheckResult),
		client:    &http.Client{},
		cancelFns: make(map[string]context.CancelFunc),
	}
}

// Start begins periodic health checks for all configured services
func (c *Checker) Start() {
	for _, check := range c.config.Checks {
		go c.runPeriodicCheck(check)
	}
}

// Stop cancels all running checks
func (c *Checker) Stop() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	
	for _, cancel := range c.cancelFns {
		cancel()
	}
}

// GetResults returns a copy of the current check results
func (c *Checker) GetResults() []CheckResult {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	results := make([]CheckResult, 0, len(c.results))
	for _, result := range c.results {
		results = append(results, result)
	}
	
	return results
}

// GetResultByName returns the result for a specific service by name
func (c *Checker) GetResultByName(name string) (CheckResult, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	
	result, found := c.results[name]
	return result, found
}

// runPeriodicCheck runs a health check at the specified interval
func (c *Checker) runPeriodicCheck(check ServiceCheck) {
	ctx, cancel := context.WithCancel(context.Background())
	
	c.mutex.Lock()
	c.cancelFns[check.Name] = cancel
	c.mutex.Unlock()
	
	ticker := time.NewTicker(check.Interval)
	defer ticker.Stop()
	
	// Run immediately on start
	c.performCheck(check)
	
	for {
		select {
		case <-ticker.C:
			c.performCheck(check)
		case <-ctx.Done():
			return
		}
	}
}

// performCheck executes a single health check
func (c *Checker) performCheck(check ServiceCheck) {
	startTime := time.Now()
	var result CheckResult
	
	result.Name = check.Name
	result.Type = check.Type
	result.Target = check.Target
	result.Timestamp = startTime
	
	switch check.Type {
	case "http":
		result = c.checkHTTP(check, startTime)
	case "tcp":
		result = c.checkTCP(check, startTime)
	default:
		result.Status = "UNKNOWN"
		result.Message = fmt.Sprintf("Unsupported check type: %s", check.Type)
	}
	
	result.Duration = time.Since(startTime)
	
	c.mutex.Lock()
	c.results[check.Name] = result
	c.mutex.Unlock()
	
	fmt.Printf("[%s] %s - %s: %s (%s)\n", 
		time.Now().Format(time.RFC3339),
		check.Name,
		result.Status,
		result.Message,
		result.Duration)
}

// checkHTTP performs an HTTP health check
func (c *Checker) checkHTTP(check ServiceCheck, startTime time.Time) CheckResult {
	result := CheckResult{
		Name:      check.Name,
		Type:      check.Type,
		Target:    check.Target,
		Timestamp: startTime,
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), check.Timeout)
	defer cancel()
	
	req, err := http.NewRequestWithContext(ctx, "GET", check.Target, nil)
	if err != nil {
		result.Status = "DOWN"
		result.Message = fmt.Sprintf("Failed to create request: %v", err)
		return result
	}
	
	resp, err := c.client.Do(req)
	if err != nil {
		result.Status = "DOWN"
		result.Message = fmt.Sprintf("Request failed: %v", err)
		return result
	}
	defer resp.Body.Close()
	
	expectedStatus := check.ExpectedStatus
	if expectedStatus == 0 {
		expectedStatus = 200
	}
	
	if resp.StatusCode != expectedStatus {
		result.Status = "DOWN"
		result.Message = fmt.Sprintf("Unexpected status: got %d, want %d", resp.StatusCode, expectedStatus)
		return result
	}
	
	result.Status = "UP"
	result.Message = fmt.Sprintf("Status code: %d", resp.StatusCode)
	return result
}

// checkTCP performs a TCP connection health check
func (c *Checker) checkTCP(check ServiceCheck, startTime time.Time) CheckResult {
	result := CheckResult{
		Name:      check.Name,
		Type:      check.Type,
		Target:    check.Target,
		Timestamp: startTime,
	}
	
	dialer := net.Dialer{
		Timeout: check.Timeout,
	}
	
	conn, err := dialer.Dial("tcp", check.Target)
	if err != nil {
		result.Status = "DOWN"
		result.Message = fmt.Sprintf("Failed to connect: %v", err)
		return result
	}
	defer conn.Close()
	
	result.Status = "UP"
	result.Message = "Connection successful"
	return result
}