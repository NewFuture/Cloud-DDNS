package main

import (
	"log"
	"sync"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/server"
)

func main() {
	if err := config.LoadConfig("config.yaml"); err != nil {
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
