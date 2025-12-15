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

	go func() {
		defer wg.Done()
		server.StartTCP(config.GlobalConfig.Server.TCPPort)
	}()

	go func() {
		defer wg.Done()
		server.StartHTTP(config.GlobalConfig.Server.HTTPPort)
	}()

	wg.Wait()
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
