package main

import (
	"flag"
	"log"
	"os"
	"sync"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/server"
)

func main() {
	// Allow config path to be specified via flag or environment variable
	configPath := flag.String("config", getEnvOrDefault("CONFIG_PATH", "config.yaml"), "Path to configuration file")
	flag.Parse()

	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatalf("Config Load Error: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(2)

	errCh := make(chan error, 2)

	go func() {
		defer wg.Done()
		if err := server.StartTCP(config.GlobalConfig.Server.TCPPort); err != nil {
			errCh <- err
		}
	}()

	go func() {
		defer wg.Done()
		if err := server.StartHTTP(config.GlobalConfig.Server.HTTPPort); err != nil {
			errCh <- err
		}
	}()

	// Wait for either an error or both servers to finish
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case err := <-errCh:
		log.Fatalf("Server Error: %v", err)
	case <-done:
		// Both servers exited cleanly
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
