package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubeConfig returns a Kubernetes client configuration
func GetKubeConfig() (*rest.Config, error) {
	// Try in-cluster config first (for when running inside Kubernetes)
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, nil
	}

	// Fall back to kubeconfig file
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("cannot get user home directory: %w", err)
		}
		kubeconfig = filepath.Join(homeDir, ".kube", "config")
	}

	// Use the current context in kubeconfig
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

// GetKubernetesClient creates a Kubernetes clientset
func GetKubernetesClient() (*kubernetes.Clientset, error) {
	config, err := GetKubeConfig()
	if err != nil {
		return nil, err
	}

	// Create the clientset
	return kubernetes.NewForConfig(config)
}

// CheckKubernetes checks the Kubernetes API health
func (c *Checker) CheckKubernetes(check ServiceCheck, startTime time.Time) CheckResult {
	result := CheckResult{
		Name:      check.Name,
		Type:      check.Type,
		Target:    check.Target,
		Timestamp: startTime,
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), check.Timeout)
	defer cancel()

	// Get Kubernetes client
	clientset, err := GetKubernetesClient()
	if err != nil {
		result.Status = "DOWN"
		result.Message = fmt.Sprintf("Failed to create Kubernetes client: %v", err)
		return result
	}

	// Check the healthz endpoint
	healthStatus, err := clientset.RESTClient().Get().AbsPath("/healthz").DoRaw(ctx)
	if err != nil {
		result.Status = "DOWN"
		result.Message = fmt.Sprintf("Kubernetes API health check failed: %v", err)
		return result
	}

	// Check the response
	if string(healthStatus) == "ok" {
		result.Status = "UP"
		result.Message = "Kubernetes API is healthy"
	} else {
		result.Status = "DOWN"
		result.Message = fmt.Sprintf("Kubernetes API returned unexpected status: %s", string(healthStatus))
	}

	return result
}