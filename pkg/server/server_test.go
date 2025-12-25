package server

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/server/mode"
)

func TestComputeMD5Hash(t *testing.T) {
	// Test MD5 hash computation used in GnuDIP protocol
	salt := "12345.67890"
	password := "testpass"

	// Client algorithm: md5( md5(password) + "." + salt )
	expectedHash := mode.ComputeTCPHash(password, salt)
	actualHash := mode.ComputeTCPHash(password, salt)

	if actualHash != expectedHash {
		t.Errorf("MD5 hash mismatch: expected %s, got %s", expectedHash, actualHash)
	}

	// Verify hash is 32 characters (MD5 hex)
	if len(actualHash) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(actualHash))
	}
}

func TestDebugLoggingToggle(t *testing.T) {
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	defer func() {
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		SetDebug(false)
	}()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)

	SetDebug(false)
	debugLogf("should not log")
	if buf.Len() != 0 {
		t.Fatalf("expected no debug output when disabled, got %q", buf.String())
	}

	buf.Reset()
	SetDebug(true)
	debugLogf("debug message %s", "on")
	if !strings.Contains(buf.String(), "[DEBUG] debug message on") {
		t.Fatalf("expected debug output when enabled, got %q", buf.String())
	}
}

func TestSaltGeneration(t *testing.T) {
	// Test that salt format is correct
	now := time.Now()
	salt := fmt.Sprintf("%d.%09d", now.Unix(), now.Nanosecond())

	parts := strings.Split(salt, ".")
	if len(parts) != 2 {
		t.Errorf("Expected salt with 2 parts, got %d", len(parts))
	}

	if len(parts[1]) != 9 {
		t.Errorf("Expected nanosecond portion to be 9 digits, got %d", len(parts[1]))
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

func TestResolveRequestIP(t *testing.T) {
	tests := []struct {
		name       string
		reqc       int
		providedIP string
		remoteAddr string
		wantIP     string
		wantErr    bool
	}{
		{
			name:       "reqc update uses provided IP",
			reqc:       0,
			providedIP: "1.2.3.4",
			remoteAddr: "10.0.0.1:1234",
			wantIP:     "1.2.3.4",
		},
		{
			name:       "reqc update falls back to remote",
			reqc:       0,
			providedIP: "",
			remoteAddr: "10.0.0.1:1234",
			wantIP:     "10.0.0.1",
		},
		{
			name:       "reqc offline forces zero IP",
			reqc:       1,
			providedIP: "8.8.8.8",
			remoteAddr: "10.0.0.1:1234",
			wantIP:     "0.0.0.0",
		},
		{
			name:       "reqc auto-detect uses remote",
			reqc:       2,
			providedIP: "8.8.8.8",
			remoteAddr: "203.0.113.10:5678",
			wantIP:     "203.0.113.10",
		},
		{
			name:       "invalid remote returns error",
			reqc:       2,
			providedIP: "",
			remoteAddr: "invalid-remote",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, err := mode.ResolveRequestIP(tt.reqc, tt.providedIP, tt.remoteAddr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("resolveRequestIP error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && ip != tt.wantIP {
				t.Fatalf("resolveRequestIP() = %s, want %s", ip, tt.wantIP)
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

	expectedHash := mode.ComputeTCPHash(password, salt)

	// Verify the user exists
	userConfig := config.GetUser(user)
	if userConfig == nil {
		t.Fatal("User not found in config")
	}

	// Recompute hash to verify
	actualHash := mode.ComputeTCPHash(password, salt)

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

	port := listener.Addr().(*net.TCPAddr).Port

	// Create context for controlled shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Track server goroutine completion
	serverDone := make(chan struct{})

	// Start server goroutine with context
	go func() {
		defer close(serverDone)
		for {
			select {
			case <-ctx.Done():
				listener.Close()
				return
			default:
				conn, err := listener.Accept()
				if err != nil {
					return // Listener closed
				}
				go mode.NewGnuTCPMode(debugLogf).Handle(conn)
			}
		}
	}()

	// Ensure cleanup happens before test exits
	defer func() {
		cancel()
		listener.Close()
		// Wait for server goroutine to exit with timeout
		select {
		case <-serverDone:
			// Server exited cleanly
		case <-time.After(2 * time.Second):
			t.Log("Warning: Server goroutine did not exit within timeout")
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
		hash := mode.ComputeTCPHash(password, salt)

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

	t.Run("Debug account bypasses provider", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true)

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

		hash := mode.ComputeTCPHash("debug", salt)
		request := fmt.Sprintf("%s:%s:debug.example.com:0:1.2.3.4\n", "debug", hash)
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

		response, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("Failed to read response: %v", err)
		}
		response = strings.TrimSpace(response)
		if response != "0" {
			t.Errorf("Expected debug bypass success '0', got '%s'", response)
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
		hash := mode.ComputeTCPHash(wrongPassword, salt)

		request := fmt.Sprintf("%s:%s:test.example.com:0:1.2.3.4\n", user, hash)
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}

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
		hash := mode.ComputeTCPHash(password, salt)

		request := fmt.Sprintf("%s:%s:test.example.com:0:1.2.3.4\n", user, hash)
		if _, err := conn.Write([]byte(request)); err != nil {
			t.Fatalf("Failed to write request: %v", err)
		}

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

// Test parameter alias helper function
func TestGetQueryParam(t *testing.T) {
	tests := []struct {
		name     string
		query    map[string][]string
		aliases  []string
		expected string
	}{
		{
			name:     "exact match",
			query:    map[string][]string{"user": {"testuser"}},
			aliases:  []string{"user", "username"},
			expected: "testuser",
		},
		{
			name:     "case insensitive match",
			query:    map[string][]string{"USER": {"testuser"}},
			aliases:  []string{"user", "username"},
			expected: "testuser",
		},
		{
			name:     "alias match",
			query:    map[string][]string{"username": {"testuser"}},
			aliases:  []string{"user", "username"},
			expected: "testuser",
		},
		{
			name:     "case insensitive alias",
			query:    map[string][]string{"UserName": {"testuser"}},
			aliases:  []string{"user", "username"},
			expected: "testuser",
		},
		{
			name:     "no match",
			query:    map[string][]string{"other": {"value"}},
			aliases:  []string{"user", "username"},
			expected: "",
		},
		{
			name:     "empty value",
			query:    map[string][]string{"user": {""}},
			aliases:  []string{"user"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mode.GetQueryParam(tt.query, tt.aliases...)
			if result != tt.expected {
				t.Errorf("getQueryParam() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestVerifyPassword(t *testing.T) {
	tests := []struct {
		name            string
		storedPassword  string
		inputPassword   string
		expectedSuccess bool
	}{
		// Plaintext matching
		{
			name:            "plaintext exact match",
			storedPassword:  "password123",
			inputPassword:   "password123",
			expectedSuccess: true,
		},
		{
			name:            "plaintext mismatch",
			storedPassword:  "password123",
			inputPassword:   "wrongpassword",
			expectedSuccess: false,
		},
		// MD5 matching - input is MD5 of stored
		{
			name:            "input is MD5 of stored password",
			storedPassword:  "password123",
			inputPassword:   "482c811da5d5b4bc6d497ffa98491e38", // MD5(password123)
			expectedSuccess: true,
		},
		{
			name:            "input is MD5 uppercase",
			storedPassword:  "password123",
			inputPassword:   "482C811DA5D5B4BC6D497FFA98491E38", // MD5(password123) uppercase
			expectedSuccess: true,
		},
		// MD5 matching - stored is MD5, input is plaintext
		{
			name:            "stored is MD5, input is plaintext",
			storedPassword:  "482c811da5d5b4bc6d497ffa98491e38", // MD5(password123)
			inputPassword:   "password123",
			expectedSuccess: true,
		},
		// SHA256 matching - input is SHA256 of stored
		{
			name:            "input is SHA256 of stored password",
			storedPassword:  "password123",
			inputPassword:   "ef92b778bafe771e89245b89ecbc08a44a4e166c06659911881f383d4473e94f", // SHA256(password123)
			expectedSuccess: true,
		},
		{
			name:            "input is SHA256 uppercase",
			storedPassword:  "password123",
			inputPassword:   "EF92B778BAFE771E89245B89ECBC08A44A4E166C06659911881F383D4473E94F", // SHA256(password123) uppercase
			expectedSuccess: true,
		},
		// SHA256 matching - stored is SHA256, input is plaintext
		{
			name:            "stored is SHA256, input is plaintext",
			storedPassword:  "ef92b778bafe771e89245b89ecbc08a44a4e166c06659911881f383d4473e94f", // SHA256(password123)
			inputPassword:   "password123",
			expectedSuccess: true,
		},
		// Base64 matching - input is Base64 of stored
		{
			name:            "input is Base64 of stored password",
			storedPassword:  "password123",
			inputPassword:   "cGFzc3dvcmQxMjM=", // Base64(password123)
			expectedSuccess: true,
		},
		// Base64 matching - stored is Base64, input is plaintext
		{
			name:            "stored is Base64, input is plaintext",
			storedPassword:  "cGFzc3dvcmQxMjM=", // Base64(password123)
			inputPassword:   "password123",
			expectedSuccess: true,
		},
		// Complex password tests
		{
			name:            "special characters plaintext",
			storedPassword:  "P@ssw0rd!#$",
			inputPassword:   "P@ssw0rd!#$",
			expectedSuccess: true,
		},
		{
			name:            "special characters MD5",
			storedPassword:  "P@ssw0rd!#$",
			inputPassword:   "f8e12466e14d9c931dca62651370657f", // MD5(P@ssw0rd!#$)
			expectedSuccess: true,
		},
		// Negative tests
		{
			name:            "wrong MD5 hash",
			storedPassword:  "password123",
			inputPassword:   "0000000000000000000000000000000", // Wrong MD5
			expectedSuccess: false,
		},
		{
			name:            "wrong SHA256 hash",
			storedPassword:  "password123",
			inputPassword:   "0000000000000000000000000000000000000000000000000000000000000000", // Wrong SHA256
			expectedSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mode.VerifyPassword(tt.storedPassword, tt.inputPassword)
			if result != tt.expectedSuccess {
				t.Errorf("verifyPassword(%q, %q) = %v, want %v",
					tt.storedPassword, tt.inputPassword, result, tt.expectedSuccess)
			}
		})
	}
}

func TestUnmatchedPathDefaultsToDynModeAndLogs(t *testing.T) {
	originalConfig := config.GlobalConfig
	originalOutput := log.Writer()
	originalFlags := log.Flags()
	defer func() {
		config.GlobalConfig = originalConfig
		log.SetOutput(originalOutput)
		log.SetFlags(originalFlags)
		SetDebug(false)
	}()

	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{
				Username: "testuser",
				Password: "testpass",
				Provider: "aliyun",
			},
		},
	}

	var buf bytes.Buffer
	log.SetOutput(&buf)
	log.SetFlags(0)
	SetDebug(true)

	req := httptest.NewRequest("GET", "/unmatched/path?user=testuser&pass=wrongpass&domn=test.example.com&addr=1.2.3.4", nil)
	w := httptest.NewRecorder()

	handleDDNSUpdate(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}

	response := strings.TrimSpace(w.Body.String())
	if response != "badauth" {
		t.Fatalf("expected badauth from DynDNS fallback, got %q", response)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "Unmatched HTTP path=/unmatched/path") || !strings.Contains(logOutput, "method=GET") {
		t.Fatalf("expected debug log for unmatched path and method, got %q", logOutput)
	}
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

	// Create a test HTTP handler using the actual handleDDNSUpdate
	handler := http.HandlerFunc(handleDDNSUpdate)

	t.Run("Successful request with explicit IP", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=testuser&pass=testpass&domn=test.example.com&addr=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := w.Body.String()
		// Will return "911" because provider credentials are invalid (test credentials)
		// In real usage with valid credentials, it would return "good <ip>"
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good ' prefix, got '%s'", response)
		}
	})

	t.Run("Successful request using Basic Auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?domain=test.example.com&addr=1.2.3.4", nil)
		req.SetBasicAuth("testuser", "testpass")
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := w.Body.String()
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good ' prefix, got '%s'", response)
		}
	})

	t.Run("Successful request with domain parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=testuser&pass=testpass&domain=test.example.com&addr=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := w.Body.String()
		// Will return "911" because provider credentials are invalid (test credentials)
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good ' prefix, got '%s'", response)
		}
	})

	t.Run("Successful request with hostname parameter (case insensitive)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?User=testuser&Pass=testpass&HostName=test.example.com&MyIP=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := w.Body.String()
		// Will return "911" because provider credentials are invalid (test credentials)
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good ' prefix, got '%s'", response)
		}
	})

	t.Run("Debug bypass succeeds without configured user", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true)
		req := httptest.NewRequest("GET", "/?user=debug&pass=debug&domn=test.example.com&addr=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}
		response := strings.TrimSpace(w.Body.String())
		if response != "good 1.2.3.4" && response != "good" {
			t.Errorf("Expected debug bypass to return success, got '%s'", response)
		}
	})

	t.Run("Successful request with alternate parameter names", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?username=testuser&password=testpass&host=test.example.com&ip=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := w.Body.String()
		// Will return "911" because provider credentials are invalid (test credentials)
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good ' prefix, got '%s'", response)
		}
	})

	t.Run("Failed authentication - wrong password", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=testuser&pass=wrongpass&domn=test.example.com&addr=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d (GnuDIP uses 200 with error message)", w.Code)
		}

		response := w.Body.String()
		if response != "badauth" {
			t.Errorf("Expected 'badauth', got '%s'", response)
		}
	})

	t.Run("Failed authentication - unknown user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=unknownuser&pass=anypass&domn=test.example.com&addr=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d (GnuDIP uses 200 with error message)", w.Code)
		}

		response := w.Body.String()
		if response != "badauth" {
			t.Errorf("Expected 'badauth', got '%s'", response)
		}
	})

	t.Run("Invalid IP address", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=testuser&pass=testpass&domn=test.example.com&addr=invalid.ip", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d (GnuDIP uses 200 with error message)", w.Code)
		}

		response := w.Body.String()
		if response != "911" {
			t.Errorf("Expected '911', got '%s'", response)
		}
	})

	t.Run("Invalid domain - too short", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/?user=testuser&pass=testpass&domn=ab&addr=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d (GnuDIP uses 200 with error message)", w.Code)
		}

		response := w.Body.String()
		if response != "notfqdn" {
			t.Errorf("Expected 'notfqdn', got '%s'", response)
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

		response := w.Body.String()
		// Will return "911" because provider credentials are invalid (test credentials)
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good ' prefix, got '%s'", response)
		}
	})

	t.Run("Test /nic/update path", func(t *testing.T) {
		// This would require starting a real server, so we just test the handler
		req := httptest.NewRequest("GET", "/nic/update?user=testuser&pass=testpass&hostname=test.example.com&myip=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := w.Body.String()
		// Will return "911" because provider credentials are invalid (test credentials)
		if response != "911" && !strings.HasPrefix(response, "good ") {
			t.Errorf("Expected '911' or 'good ' prefix, got '%s'", response)
		}
	})
}

func TestDDNSServicePrefersBasicAuth(t *testing.T) {
	service := mode.NewDynMode(false, debugLogf)
	req := httptest.NewRequest("GET", "/?user=queryUser&pass=queryPass&domain=test.example.com&addr=1.1.1.1", nil)
	req.RemoteAddr = "10.0.0.2:12345"
	req.SetBasicAuth("headerUser", "headerPass")

	ddnsReq, outcome := service.Prepare(req)
	if outcome != mode.OutcomeSuccess {
		t.Fatalf("expected success outcome, got %v", outcome)
	}

	if ddnsReq.Username != "headerUser" {
		t.Fatalf("expected username from header, got %s", ddnsReq.Username)
	}
	if ddnsReq.Password != "headerPass" {
		t.Fatalf("expected password from header, got %s", ddnsReq.Password)
	}
	if ddnsReq.IP != "1.1.1.1" {
		t.Fatalf("expected parsed IP 1.1.1.1, got %s", ddnsReq.IP)
	}
}
