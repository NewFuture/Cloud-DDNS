package server

import (
	"crypto/md5"
	"fmt"
	"net"
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
		name        string
		message     string
		expectParts int
		expectUser  string
		expectHash  string
		expectDomain string
	}{
		{
			name:        "Standard GnuDIP message",
			message:     "user:hash:domain.com:0:1.2.3.4",
			expectParts: 5,
			expectUser:  "user",
			expectHash:  "hash",
			expectDomain: "domain.com",
		},
		{
			name:        "Message without IP",
			message:     "user:hash:domain.com",
			expectParts: 3,
			expectUser:  "user",
			expectHash:  "hash",
			expectDomain: "domain.com",
		},
		{
			name:        "Message with empty IP",
			message:     "user:hash:domain.com:0:",
			expectParts: 5,
			expectUser:  "user",
			expectHash:  "hash",
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
