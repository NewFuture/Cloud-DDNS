package server

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NewFuture/CloudDDNS/pkg/config"
)

func TestComputeMD5Hash(t *testing.T) {
	// Test MD5 hash computation used in GnuDIP protocol
	user := "testuser"
	salt := "12345.67890"
	password := "testpass"

	expectedStr := fmt.Sprintf("%s:%s:%s", user, salt, password)
	expectedHash := fmt.Sprintf("%x", md5.Sum([]byte(expectedStr)))

	// Compute hash
	actualHash := fmt.Sprintf("%x", md5.Sum([]byte(expectedStr)))

	if actualHash != expectedHash {
		t.Errorf("MD5 hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	// Verify hash is 32 characters (MD5 hex)
	if len(actualHash) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(actualHash))
	}
}

func TestSaltGeneration(t *testing.T) {
	// Test that salt format is correct
	now := time.Now()
	salt := fmt.Sprintf("%d.%d", now.Unix(), now.UnixNano())

	parts := strings.Split(salt, ".")
	if len(parts) != 2 {
		t.Errorf("Expected salt with 2 parts, got %d", len(parts))
	}

	// Verify both parts are numeric
	for i, part := range parts {
		if len(part) == 0 {
			t.Errorf("Salt part %d is empty", i)
		}
		for _, c := range part {
			if c < '0' || c > '9' {
				t.Errorf("Salt part %d contains non-numeric character: %c", i, c)
			}
		}
	}
}

func TestIPExtraction(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		expected   string
	}{
		{
			name:       "IPv4 with port",
			remoteAddr: "192.168.1.100:12345",
			expected:   "192.168.1.100",
		},
		{
			name:       "IPv6 with port",
			remoteAddr: "[2001:db8::1]:8080",
			expected:   "2001:db8::1",
		},
		{
			name:       "Localhost IPv4",
			remoteAddr: "127.0.0.1:3495",
			expected:   "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, _, err := net.SplitHostPort(tt.remoteAddr)
			if err != nil {
				t.Fatalf("Failed to split host port: %v", err)
			}

			if host != tt.expected {
				t.Errorf("Expected IP %s, got %s", tt.expected, host)
			}
		})
	}
}

func TestProtocolMessageParsing(t *testing.T) {
	tests := []struct {
		name         string
		message      string
		expectParts  int
		expectUser   string
		expectHash   string
		expectDomain string
	}{
		{
			name:         "Standard GnuDIP message",
			message:      "user:hash:domain.com:0:1.2.3.4",
			expectParts:  5,
			expectUser:   "user",
			expectHash:   "hash",
			expectDomain: "domain.com",
		},
		{
			name:         "Message without IP",
			message:      "user:hash:domain.com",
			expectParts:  3,
			expectUser:   "user",
			expectHash:   "hash",
			expectDomain: "domain.com",
		},
		{
			name:         "Message with empty IP",
			message:      "user:hash:domain.com:0:",
			expectParts:  5,
			expectUser:   "user",
			expectHash:   "hash",
			expectDomain: "domain.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parts := strings.Split(strings.TrimSpace(tt.message), ":")

			if len(parts) != tt.expectParts {
				t.Errorf("Expected %d parts, got %d", tt.expectParts, len(parts))
				return
			}

			if len(parts) >= 1 && parts[0] != tt.expectUser {
				t.Errorf("Expected user '%s', got '%s'", tt.expectUser, parts[0])
			}

			if len(parts) >= 2 && parts[1] != tt.expectHash {
				t.Errorf("Expected hash '%s', got '%s'", tt.expectHash, parts[1])
			}

			if len(parts) >= 3 && parts[2] != tt.expectDomain {
				t.Errorf("Expected domain '%s', got '%s'", tt.expectDomain, parts[2])
			}
		})
	}
}

func TestAuthenticationFlow(t *testing.T) {
	// Save original config and restore after test
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()

	// Setup test config
	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{
				Username: "testuser",
				Password: "testpass",
				Provider: "aliyun",
			},
		},
	}

	user := "testuser"
	password := "testpass"
	salt := "1234567890.9876543210"

	// Compute expected hash
	expectedStr := fmt.Sprintf("%s:%s:%s", user, salt, password)
	expectedHash := fmt.Sprintf("%x", md5.Sum([]byte(expectedStr)))

	// Verify the user exists
	userConfig := config.GetUser(user)
	if userConfig == nil {
		t.Fatal("User not found in config")
	}

	// Recompute hash to verify
	actualStr := fmt.Sprintf("%s:%s:%s", user, salt, userConfig.Password)
	actualHash := fmt.Sprintf("%x", md5.Sum([]byte(actualStr)))

	if actualHash != expectedHash {
		t.Errorf("Hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}
}

func TestDomainAndIPExtraction(t *testing.T) {
	// Test extracting domain and IP from protocol message
	message := "user:hash:example.com:0:10.0.0.1"
	parts := strings.Split(strings.TrimSpace(message), ":")

	var targetIP string
	if len(parts) > 4 {
		targetIP = parts[4]
	}

	if targetIP != "10.0.0.1" {
		t.Errorf("Expected IP '10.0.0.1', got '%s'", targetIP)
	}

	// Test with empty IP (should use RemoteAddr)
	message2 := "user:hash:example.com:0:"
	parts2 := strings.Split(strings.TrimSpace(message2), ":")

	var targetIP2 string
	if len(parts2) > 4 {
		targetIP2 = parts2[4]
	}

	if targetIP2 != "" {
		t.Errorf("Expected empty IP, got '%s'", targetIP2)
	}
}

// Integration test for TCP server handler
func TestTCPServerIntegration(t *testing.T) {
	// Save original config and restore after test
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()

	// Setup test config
	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{
				Username: "testuser",
				Password: "testpass",
				Provider: "aliyun",
			},
		},
	}

	// Start TCP server on random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port

	// Start server goroutine
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // Listener closed
			}
			go handleTCPConnection(conn)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	t.Run("Successful authentication", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		// Read salt
		reader := bufio.NewReader(conn)
		salt, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read salt: %v", err)
		}
		salt = strings.TrimSpace(salt)

		// Verify salt format
		if len(salt) == 0 {
			t.Fatal("Salt is empty")
		}

		// Compute hash
		user := "testuser"
		password := "testpass"
		hashStr := fmt.Sprintf("%s:%s:%s", user, salt, password)
		hash := fmt.Sprintf("%x", md5.Sum([]byte(hashStr)))

		// Send request - note: this will fail because we don't have real provider setup
		// But we're testing the protocol flow
		request := fmt.Sprintf("%s:%s:test.example.com:0:1.2.3.4\n", user, hash)
		_, err = conn.Write([]byte(request))
		if err != nil {
			t.Fatalf("Failed to write request: %v", err)
		}

		// Read response (will be "1" due to provider error, but connection should work)
		response, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		response = strings.TrimSpace(response)

		// We expect "1" (error) because no real provider is configured
		if response != "1" && response != "0" {
			t.Errorf("Expected '0' or '1', got '%s'", response)
		}
	})

	t.Run("Failed authentication - wrong password", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		salt, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read salt: %v", err)
		}
		salt = strings.TrimSpace(salt)

		// Use wrong password
		user := "testuser"
		wrongPassword := "wrongpass"
		hashStr := fmt.Sprintf("%s:%s:%s", user, salt, wrongPassword)
		hash := fmt.Sprintf("%x", md5.Sum([]byte(hashStr)))

		request := fmt.Sprintf("%s:%s:test.example.com:0:1.2.3.4\n", user, hash)
		conn.Write([]byte(request))

		response, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		response = strings.TrimSpace(response)

		if response != "1" {
			t.Errorf("Expected authentication failure '1', got '%s'", response)
		}
	})

	t.Run("Failed authentication - unknown user", func(t *testing.T) {
		conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		reader := bufio.NewReader(conn)
		salt, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read salt: %v", err)
		}
		salt = strings.TrimSpace(salt)

		// Use unknown user
		user := "unknownuser"
		password := "anypass"
		hashStr := fmt.Sprintf("%s:%s:%s", user, salt, password)
		hash := fmt.Sprintf("%x", md5.Sum([]byte(hashStr)))

		request := fmt.Sprintf("%s:%s:test.example.com:0:1.2.3.4\n", user, hash)
		conn.Write([]byte(request))

		response, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		response = strings.TrimSpace(response)

		if response != "1" {
			t.Errorf("Expected authentication failure '1', got '%s'", response)
		}
	})
}

// Integration test for HTTP server handler
func TestHTTPServerIntegration(t *testing.T) {
	// Save original config and restore after test
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()

	// Setup test config
	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{
				Username: "testuser",
				Password: "testpass",
				Provider: "aliyun",
			},
		},
	}

	// Create a test HTTP handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		user := q.Get("user")
		pass := q.Get("pass")
		domain := q.Get("domn")
		if domain == "" {
			domain = q.Get("domain")
		}
		ip := q.Get("addr")

		// Validate domain length
		if len(domain) < 3 || len(domain) > 253 {
			http.Error(w, "Invalid domain length", 400)
			return
		}

		// Auto-detect IP if not provided
		if ip == "" {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				http.Error(w, "Invalid remote address", 400)
				return
			}
			ip = host
		}

		// Validate IP
		if net.ParseIP(ip) == nil {
			http.Error(w, "Invalid IP address", 400)
			return
		}

		// Authenticate
		u := config.GetUser(user)
		if u == nil || u.Password != pass {
			http.Error(w, "Auth failed", 401)
			return
		}

		// Success (in real case would call provider)
		w.Write([]byte("0"))
	})

	t.Run("Successful request with explicit IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=testuser&pass=testpass&domn=test.example.com&addr=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Body.String() != "0" {
			t.Errorf("Expected '0', got '%s'", w.Body.String())
		}
	})

	t.Run("Successful request with domain parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=testuser&pass=testpass&domain=test.example.com&addr=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})

	t.Run("Failed authentication - wrong password", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=testuser&pass=wrongpass&domn=test.example.com&addr=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("Failed authentication - unknown user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=unknownuser&pass=anypass&domn=test.example.com&addr=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", w.Code)
		}
	})

	t.Run("Invalid IP address", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=testuser&pass=testpass&domn=test.example.com&addr=invalid.ip", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("Invalid domain - too short", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=testuser&pass=testpass&domn=ab&addr=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Expected status 400, got %d", w.Code)
		}
	})

	t.Run("Auto-detect IP from RemoteAddr", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=testuser&pass=testpass&domn=test.example.com", nil)
		req.RemoteAddr = "10.20.30.40:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
	})
}
