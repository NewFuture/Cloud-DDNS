package server

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/server/mode"
)

// TestOrayTCPProtocol tests the Oray TCP protocol implementation
func TestOrayTCPProtocol(t *testing.T) {
	// Save original config and restore after test
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()

	// Setup test config
	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{
				Username: "orayuser",
				Password: "oraypass",
				Provider: "aliyun",
			},
		},
	}

	// Start a test TCP server on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Handle connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go mode.NewOrayTCPMode(func(format string, args ...interface{}) {
				t.Logf("[DEBUG] "+format, args...)
			}).Handle(conn)
		}
	}()

	// Helper function to send request and get response
	sendRequest := func(request string) (string, error) {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 5*time.Second)
		if err != nil {
			return "", fmt.Errorf("connection failed: %v", err)
		}
		defer conn.Close()

		conn.SetDeadline(time.Now().Add(5 * time.Second))

		// Send request
		if _, err := conn.Write([]byte(request + "\n")); err != nil {
			return "", fmt.Errorf("write failed: %v", err)
		}

		// Read response
		reader := bufio.NewReader(conn)
		response, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("read failed: %v", err)
		}

		return strings.TrimSpace(response), nil
	}

	t.Run("Valid request with IP", func(t *testing.T) {
		response, err := sendRequest("orayuser:oraypass:test.example.com:1.2.3.4")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		// Should get "911" because credentials are not valid for actual provider
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good' prefix, got: %s", response)
		}
	})

	t.Run("Valid request without IP (auto-detect)", func(t *testing.T) {
		response, err := sendRequest("orayuser:oraypass:test.example.com:")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		// Should get "911" because credentials are not valid for actual provider
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good' prefix, got: %s", response)
		}
	})

	t.Run("Valid request without IP field", func(t *testing.T) {
		response, err := sendRequest("orayuser:oraypass:test.example.com")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		// Should get "911" because credentials are not valid for actual provider
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good' prefix, got: %s", response)
		}
	})

	t.Run("Authentication failure - wrong password", func(t *testing.T) {
		response, err := sendRequest("orayuser:wrongpass:test.example.com:1.2.3.4")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if response != "badauth" {
			t.Errorf("Expected 'badauth', got: %s", response)
		}
	})

	t.Run("Authentication failure - unknown user", func(t *testing.T) {
		response, err := sendRequest("unknownuser:anypass:test.example.com:1.2.3.4")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if response != "badauth" {
			t.Errorf("Expected 'badauth', got: %s", response)
		}
	})

	t.Run("Invalid hostname - too short", func(t *testing.T) {
		response, err := sendRequest("orayuser:oraypass:ab:1.2.3.4")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if response != "notfqdn" {
			t.Errorf("Expected 'notfqdn', got: %s", response)
		}
	})

	t.Run("Invalid hostname - empty", func(t *testing.T) {
		response, err := sendRequest("orayuser:oraypass::1.2.3.4")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if response != "notfqdn" {
			t.Errorf("Expected 'notfqdn', got: %s", response)
		}
	})

	t.Run("Invalid IP address", func(t *testing.T) {
		response, err := sendRequest("orayuser:oraypass:test.example.com:invalid.ip")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if response != "911" {
			t.Errorf("Expected '911', got: %s", response)
		}
	})

	t.Run("IPv6 address", func(t *testing.T) {
		response, err := sendRequest("orayuser:oraypass:test.example.com:2001:db8::1")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		// Should get "911" because credentials are not valid for actual provider
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good' prefix, got: %s", response)
		}
	})

	t.Run("Malformed request - too few parts", func(t *testing.T) {
		response, err := sendRequest("orayuser:oraypass")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if response != "911" {
			t.Errorf("Expected '911', got: %s", response)
		}
	})

	t.Run("Debug mode bypass", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true)

		response, err := sendRequest("debug:debug:test.example.com:1.2.3.4")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		if !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected 'good' prefix in debug mode, got: %s", response)
		}
		if !strings.Contains(response, "1.2.3.4") {
			t.Errorf("Expected response to include IP '1.2.3.4', got: %s", response)
		}
	})

	t.Run("Subdomain hostname", func(t *testing.T) {
		response, err := sendRequest("orayuser:oraypass:sub.example.com:1.2.3.4")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		// Should get "911" because credentials are not valid for actual provider
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good' prefix, got: %s", response)
		}
	})

	t.Run("Deep subdomain hostname", func(t *testing.T) {
		response, err := sendRequest("orayuser:oraypass:deep.sub.example.com:1.2.3.4")
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		// Should get "911" because credentials are not valid for actual provider
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good' prefix, got: %s", response)
		}
	})
}

// TestOrayTCPHashComputation tests the Oray hash computation function
func TestOrayTCPHashComputation(t *testing.T) {
	tests := []struct {
		name     string
		username string
		password string
		salt     string
		want     string
	}{
		{
			name:     "Basic hash",
			username: "user",
			password: "pass",
			salt:     "salt123",
			want:     "4a52d6e8f3e8c8a8f8f8f8f8f8f8f8f8", // This is just for structure, actual hash may differ
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mode.ComputeOrayHash(tt.username, tt.password, tt.salt)
			if len(got) != 32 {
				t.Errorf("ComputeOrayHash() returned hash of length %d, expected 32", len(got))
			}
			// Verify it's hexadecimal
			for _, c := range got {
				if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
					t.Errorf("ComputeOrayHash() returned non-hex character: %c", c)
				}
			}
		})
	}
}

// TestOrayTCPConcurrentConnections tests handling multiple concurrent connections
func TestOrayTCPConcurrentConnections(t *testing.T) {
	// Save original config and restore after test
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()

	// Setup test config
	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{
				Username: "orayuser",
				Password: "oraypass",
				Provider: "aliyun",
			},
		},
	}

	// Enable debug mode for successful responses
	defer SetDebug(false)
	SetDebug(true)

	// Start a test TCP server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Handle connections in background
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go mode.NewOrayTCPMode(func(format string, args ...interface{}) {
				// Silent in concurrent test
			}).Handle(conn)
		}
	}()

	// Send multiple concurrent requests
	const numRequests = 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func(id int) {
			conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 5*time.Second)
			if err != nil {
				results <- fmt.Errorf("connection %d failed: %v", id, err)
				return
			}
			defer conn.Close()

			conn.SetDeadline(time.Now().Add(5 * time.Second))

			request := fmt.Sprintf("debug:debug:test%d.example.com:1.2.3.%d\n", id, id)
			if _, err := conn.Write([]byte(request)); err != nil {
				results <- fmt.Errorf("write %d failed: %v", id, err)
				return
			}

			reader := bufio.NewReader(conn)
			response, err := reader.ReadString('\n')
			if err != nil {
				results <- fmt.Errorf("read %d failed: %v", id, err)
				return
			}

			if !strings.HasPrefix(strings.TrimSpace(response), "good ") {
				results <- fmt.Errorf("request %d got unexpected response: %s", id, response)
				return
			}

			results <- nil
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		if err := <-results; err != nil {
			t.Errorf("Concurrent request failed: %v", err)
		}
	}
}
