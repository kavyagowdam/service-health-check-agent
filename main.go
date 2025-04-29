package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	// Parse command line flags
	configFile := flag.String("config", "config.yaml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	config, err := LoadConfig(*configFile)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded configuration with %d checks\n", len(config.Checks))

	// Create and start the checker
	checker := NewChecker(config)
	checker.Start()

	// Create and start the API server in a goroutine
	apiServer := NewAPIServer(checker, config.APIPort)
	go func() {
		if err := apiServer.Start(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("API server error: %v\n", err)
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("Shutting down...")
	checker.Stop()
	apiServer.Stop()
	fmt.Println("Shutdown complete")
}