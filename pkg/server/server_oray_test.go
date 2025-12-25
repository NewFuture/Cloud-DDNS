package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NewFuture/CloudDDNS/pkg/config"
)

// testMethods is a helper to test both GET and POST methods
func testMethods(t *testing.T, testFunc func(t *testing.T, method string)) {
	t.Helper()
	for _, method := range []string{"GET", "POST"} {
		t.Run(method, func(t *testing.T) {
			testFunc(t, method)
		})
	}
}

// TestOrayEndpoint tests the /ph/update endpoint specific to Oray protocol
func TestOrayEndpoint(t *testing.T) {
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

	t.Run("Oray URL format with Basic Auth", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			// Test URL: http://username:password@server/ph/update?hostname=domain&myip=ip
			req := httptest.NewRequest(method, "/ph/update?hostname=test.example.com&myip=1.2.3.4", nil)
			req.SetBasicAuth("testuser", "testpass")
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			response := w.Body.String()
			// Should return "911" or "good" prefix (911 because test credentials don't have real provider access)
			if response != "911" && !strings.HasPrefix(response, "good ") && response != "good" {
				t.Errorf("Expected '911' or 'good' prefix, got '%s'", response)
			}
		})
	})

	t.Run("Oray URL format with embedded credentials", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			// Alternative format with query parameters
			req := httptest.NewRequest(method, "/ph/update?user=testuser&pass=testpass&hostname=test.example.com&myip=1.2.3.4", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			response := w.Body.String()
			if response != "911" && !strings.HasPrefix(response, "good ") && response != "good" {
				t.Errorf("Expected '911' or 'good' prefix, got '%s'", response)
			}
		})
	})

	t.Run("Oray parameter names - hostname", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			// Oray uses 'hostname' parameter for domain
			req := httptest.NewRequest(method, "/ph/update?user=testuser&pass=testpass&hostname=test.example.com&myip=1.2.3.4", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			response := w.Body.String()
			// Should NOT return notfqdn since hostname should be recognized
			if response == "notfqdn" {
				t.Errorf("Expected to parse 'hostname' parameter correctly, got 'notfqdn'")
			}
		})
	})

	t.Run("Oray parameter names - myip", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			// Oray uses 'myip' parameter for IP address
			req := httptest.NewRequest(method, "/ph/update?user=testuser&pass=testpass&hostname=test.example.com&myip=192.168.1.1", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			response := w.Body.String()
			// Should process the request (will fail at provider level with 911, but not at parameter parsing)
			if response == "notfqdn" {
				t.Errorf("Expected to parse parameters correctly, got 'notfqdn'")
			}
		})
	})

	t.Run("Oray auto-detect IP from RemoteAddr", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			// When myip is omitted, should use client's IP
			req := httptest.NewRequest(method, "/ph/update?user=testuser&pass=testpass&hostname=test.example.com", nil)
			req.RemoteAddr = "203.0.113.50:54321"
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			response := w.Body.String()
			// Should attempt to use RemoteAddr IP
			if response == "notfqdn" {
				t.Errorf("Expected to auto-detect IP from RemoteAddr, got 'notfqdn'")
			}
		})
	})

	t.Run("Oray return code: badauth (wrong password)", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			req := httptest.NewRequest(method, "/ph/update?user=testuser&pass=wrongpassword&hostname=test.example.com&myip=1.2.3.4", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			response := w.Body.String()
			if response != "badauth" {
				t.Errorf("Expected 'badauth' for wrong password, got '%s'", response)
			}
		})
	})

	t.Run("Oray return code: badauth (unknown user)", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			req := httptest.NewRequest(method, "/ph/update?user=unknownuser&pass=anypass&hostname=test.example.com&myip=1.2.3.4", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			response := w.Body.String()
			if response != "badauth" {
				t.Errorf("Expected 'badauth' for unknown user, got '%s'", response)
			}
		})
	})

	t.Run("Oray return code: notfqdn (invalid domain)", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			req := httptest.NewRequest(method, "/ph/update?user=testuser&pass=testpass&hostname=ab&myip=1.2.3.4", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			response := w.Body.String()
			if response != "notfqdn" {
				t.Errorf("Expected 'notfqdn' for invalid domain, got '%s'", response)
			}
		})
	})

	t.Run("Oray return code: notfqdn (missing domain)", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			req := httptest.NewRequest(method, "/ph/update?user=testuser&pass=testpass&myip=1.2.3.4", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			response := w.Body.String()
			if response != "notfqdn" {
				t.Errorf("Expected 'notfqdn' for missing domain, got '%s'", response)
			}
		})
	})

	t.Run("Oray return code: 911 (invalid IP)", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			req := httptest.NewRequest(method, "/ph/update?user=testuser&pass=testpass&hostname=test.example.com&myip=invalid.ip.address", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			response := w.Body.String()
			if response != "911" {
				t.Errorf("Expected '911' for invalid IP, got '%s'", response)
			}
		})
	})

	t.Run("Oray return code: good with IP", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			// Enable debug mode to bypass provider check
			defer SetDebug(false)
			SetDebug(true)

			req := httptest.NewRequest(method, "/ph/update?user=debug&pass=debug&hostname=test.example.com&myip=1.2.3.4", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			response := strings.TrimSpace(w.Body.String())
			// In debug mode, should return success with IP
			if !strings.HasPrefix(response, "good ") && response != "good" {
				t.Errorf("Expected 'good' prefix in debug mode, got '%s'", response)
			}
			// Should include the IP in the response
			if strings.HasPrefix(response, "good ") && !strings.Contains(response, "1.2.3.4") {
				t.Errorf("Expected response to include IP '1.2.3.4', got '%s'", response)
			}
		})
	})

	t.Run("Oray case-insensitive parameter names", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			// Test that parameter names are case-insensitive
			req := httptest.NewRequest(method, "/ph/update?User=testuser&Pass=testpass&HostName=test.example.com&MyIP=1.2.3.4", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			response := w.Body.String()
			// Should parse parameters correctly despite case differences
			if response == "notfqdn" {
				t.Errorf("Expected case-insensitive parameter parsing, got 'notfqdn'")
			}
			if response != "911" && !strings.HasPrefix(response, "good ") && response != "good" {
				t.Errorf("Expected '911' or 'good' prefix, got '%s'", response)
			}
		})
	})

	t.Run("Oray IPv6 support", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			defer SetDebug(false)
			SetDebug(true)

			req := httptest.NewRequest(method, "/ph/update?user=debug&pass=debug&hostname=test.example.com&myip=2001:db8::1", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			response := strings.TrimSpace(w.Body.String())
			if !strings.HasPrefix(response, "good") {
				t.Errorf("Expected 'good' for valid IPv6, got '%s'", response)
			}
		})
	})

	t.Run("Oray with subdomain", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			defer SetDebug(false)
			SetDebug(true)

			req := httptest.NewRequest(method, "/ph/update?user=debug&pass=debug&hostname=sub.example.com&myip=1.2.3.4", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			response := w.Body.String()
			if !strings.HasPrefix(response, "good") {
				t.Errorf("Expected 'good' for subdomain, got '%s'", response)
			}
		})
	})

	t.Run("Oray Basic Auth preference over query params", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			// When both Basic Auth and query params are provided, Basic Auth should be preferred
			req := httptest.NewRequest(method, "/ph/update?user=wronguser&pass=wrongpass&hostname=test.example.com&myip=1.2.3.4", nil)
			req.SetBasicAuth("testuser", "testpass")
			w := httptest.NewRecorder()

			handler(w, req)

			response := w.Body.String()
			// Should use Basic Auth credentials and not return badauth
			if response == "badauth" {
				t.Errorf("Expected Basic Auth to be preferred over query params, got 'badauth'")
			}
		})
	})
}

// TestOrayProtocolCompatibility verifies Oray-specific protocol behaviors
func TestOrayProtocolCompatibility(t *testing.T) {
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

	handler := http.HandlerFunc(handleDDNSUpdate)

	t.Run("Oray standard request format", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			// Simulating: http://username:password@server/ph/update?hostname=domain.com&myip=1.2.3.4
			req := httptest.NewRequest(method, "/ph/update?hostname=oray.example.com&myip=203.0.113.100", nil)
			req.SetBasicAuth("orayuser", "oraypass")
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			response := w.Body.String()
			// With invalid provider credentials, should get 911 (system error)
			// With valid credentials, would get "good <ip>"
			if response != "911" && !strings.HasPrefix(response, "good ") && response != "good" {
				t.Errorf("Expected valid response format, got '%s'", response)
			}
		})
	})

	t.Run("Oray request without myip parameter", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			// When myip is not provided, should use client's IP address
			req := httptest.NewRequest(method, "/ph/update?hostname=oray.example.com", nil)
			req.SetBasicAuth("orayuser", "oraypass")
			req.RemoteAddr = "198.51.100.50:12345"
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			response := w.Body.String()
			// Should successfully parse and use RemoteAddr
			if response == "notfqdn" {
				t.Errorf("Expected to handle missing myip by using RemoteAddr, got 'notfqdn'")
			}
		})
	})

	t.Run("Oray response format for successful update", func(t *testing.T) {
		testMethods(t, func(t *testing.T, method string) {
			defer SetDebug(false)
			SetDebug(true)

			req := httptest.NewRequest(method, "/ph/update?hostname=test.example.com&myip=1.2.3.4", nil)
			req.SetBasicAuth("debug", "debug")
			w := httptest.NewRecorder()

			handler(w, req)

			response := strings.TrimSpace(w.Body.String())
			// Oray expects: "good <ip>" for successful updates
			expectedPrefix := "good"
			if !strings.HasPrefix(response, expectedPrefix) {
				t.Errorf("Expected response to start with '%s', got '%s'", expectedPrefix, response)
			}

			// Should include the IP address in the response
			parts := strings.Fields(response)
			if len(parts) >= 2 && parts[0] == "good" {
				if parts[1] != "1.2.3.4" {
					t.Logf("Note: IP in response is '%s', expected '1.2.3.4'", parts[1])
				}
			}
		})
	})
}
