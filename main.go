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
	debug := flag.Bool("debug", false, "Enable debug logging to print full request parameters and step-by-step status")
	flag.Parse()

	if err := config.LoadConfig(*configPath); err != nil {
		log.Fatalf("Config Load Error: %v", err)
	}

	server.SetDebug(*debug)

	var wg sync.WaitGroup

	// Start GnuDIP TCP server
	wg.Add(1)
	go func() {
		defer wg.Done()
		server.StartTCP(config.GlobalConfig.Server.TCPPort)
	}()

	// Start Oray TCP server if port is configured
	if config.GlobalConfig.Server.OrayTCPPort > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			server.StartOrayTCP(config.GlobalConfig.Server.OrayTCPPort)
		}()
	}

	// Start HTTP server
	wg.Add(1)
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
