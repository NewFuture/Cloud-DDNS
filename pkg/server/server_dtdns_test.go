package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/server/mode"
)

// TestNowDNSUpdateEndpoint tests the /update endpoint with Basic Auth
// as used by Now-DNS (DtDNS) protocol:
// curl -u <email>:<password_or_api_token> "host/update?hostname=<hostname>&myip=<ip>"
// Implemented response codes: good, badauth, notfqdn, 911
// Note: nochg and nohost are documented by Now-DNS but map to good/notfqdn in this implementation
func TestNowDNSUpdateEndpoint(t *testing.T) {
	// Save original config and restore after test
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()

	// Setup test config
	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{
				Username: "user@example.com",
				Password: "api_token",
				Provider: "aliyun",
			},
		},
	}

	handler := http.HandlerFunc(handleDDNSUpdate)

	t.Run("Now-DNS format with Basic Auth and hostname parameter - success returns good", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true) // Enable debug mode to bypass provider

		req := httptest.NewRequest("GET", "/update?hostname=test.example.com&myip=1.2.3.4", nil)
		req.SetBasicAuth("debug", "debug")
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		if !strings.HasPrefix(response, "good") {
			t.Errorf("Expected 'good' response prefix for success, got '%s'", response)
		}
	})

	t.Run("Now-DNS format with Basic Auth - badauth for invalid credentials", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/update?hostname=test.example.com", nil)
		req.SetBasicAuth("invalid@example.com", "wrongtoken")
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		if response != "badauth" {
			t.Errorf("Expected 'badauth' for invalid credentials, got '%s'", response)
		}
	})

	t.Run("Now-DNS format with Basic Auth - notfqdn for invalid hostname", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/update?hostname=ab", nil) // hostname too short
		req.SetBasicAuth("user@example.com", "api_token")
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		if response != "notfqdn" {
			t.Errorf("Expected 'notfqdn' for invalid hostname, got '%s'", response)
		}
	})

	t.Run("Now-DNS format with Basic Auth - notfqdn for missing hostname", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/update", nil) // no hostname
		req.SetBasicAuth("user@example.com", "api_token")
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		if response != "notfqdn" {
			t.Errorf("Expected 'notfqdn' for missing hostname, got '%s'", response)
		}
	})

	t.Run("Now-DNS format with Basic Auth - auto-detect IP from RemoteAddr", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true) // Enable debug mode to bypass provider

		req := httptest.NewRequest("GET", "/update?hostname=test.example.com", nil)
		req.SetBasicAuth("debug", "debug")
		req.RemoteAddr = "10.20.30.40:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		// Should use RemoteAddr IP when myip is not provided
		if !strings.HasPrefix(response, "good") {
			t.Errorf("Expected 'good' response prefix, got '%s'", response)
		}
		if response != "good 10.20.30.40" && response != "good" {
			t.Logf("Response was: %s (IP may be included)", response)
		}
	})

	t.Run("Now-DNS format with Basic Auth - explicit myip parameter", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true) // Enable debug mode to bypass provider

		req := httptest.NewRequest("GET", "/update?hostname=test.example.com&myip=8.8.8.8", nil)
		req.SetBasicAuth("debug", "debug")
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		if response != "good 8.8.8.8" && response != "good" {
			t.Errorf("Expected 'good 8.8.8.8' or 'good', got '%s'", response)
		}
	})

	t.Run("Now-DNS format - 911 for system error (invalid provider)", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/update?hostname=test.example.com&myip=1.2.3.4", nil)
		req.SetBasicAuth("user@example.com", "api_token")
		req.RemoteAddr = "192.168.1.100:12345"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		// With test credentials, provider will fail, returning 911
		if response != "911" {
			t.Errorf("Expected '911' for system error, got '%s'", response)
		}
	})
}

// TestDtDNSModeResponseCodes tests the DtDNS mode specifically
func TestDtDNSModeResponseCodes(t *testing.T) {
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

	dtdnsMode := mode.NewDtDNSMode(debugLogf)

	t.Run("DtDNS mode Prepare extracts hostname from id parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/autodns.cfm?id=test.example.com&ip=1.2.3.4", nil)
		req.SetBasicAuth("testuser", "testpass")
		req.RemoteAddr = "10.0.0.1:1234"

		ddnsReq, outcome := dtdnsMode.Prepare(req)
		if outcome != mode.OutcomeSuccess {
			t.Fatalf("expected success outcome, got %v", outcome)
		}

		if ddnsReq.Domain != "test.example.com" {
			t.Errorf("Expected domain 'test.example.com', got '%s'", ddnsReq.Domain)
		}
		if ddnsReq.IP != "1.2.3.4" {
			t.Errorf("Expected IP '1.2.3.4', got '%s'", ddnsReq.IP)
		}
	})

	t.Run("DtDNS mode Prepare extracts hostname from hostname parameter", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/autodns.cfm?hostname=myhost.example.com&myip=5.6.7.8", nil)
		req.SetBasicAuth("testuser", "testpass")
		req.RemoteAddr = "10.0.0.1:1234"

		ddnsReq, outcome := dtdnsMode.Prepare(req)
		if outcome != mode.OutcomeSuccess {
			t.Fatalf("expected success outcome, got %v", outcome)
		}

		if ddnsReq.Domain != "myhost.example.com" {
			t.Errorf("Expected domain 'myhost.example.com', got '%s'", ddnsReq.Domain)
		}
		if ddnsReq.IP != "5.6.7.8" {
			t.Errorf("Expected IP '5.6.7.8', got '%s'", ddnsReq.IP)
		}
	})

	t.Run("DtDNS mode Respond returns good for success", func(t *testing.T) {
		w := httptest.NewRecorder()
		ddnsReq := &mode.Request{
			Domain: "test.example.com",
			IP:     "1.2.3.4",
		}
		dtdnsMode.Respond(w, ddnsReq, mode.OutcomeSuccess)

		response := strings.TrimSpace(w.Body.String())
		if !strings.HasPrefix(response, "good") {
			t.Errorf("Expected 'good' prefix for success, got '%s'", response)
		}
	})

	t.Run("DtDNS mode Respond returns badauth for auth failure", func(t *testing.T) {
		w := httptest.NewRecorder()
		ddnsReq := &mode.Request{
			Domain: "test.example.com",
			IP:     "1.2.3.4",
		}
		dtdnsMode.Respond(w, ddnsReq, mode.OutcomeAuthFailure)

		response := strings.TrimSpace(w.Body.String())
		if response != "badauth" {
			t.Errorf("Expected 'badauth' for auth failure, got '%s'", response)
		}
	})

	t.Run("DtDNS mode Respond returns notfqdn for invalid domain", func(t *testing.T) {
		w := httptest.NewRecorder()
		ddnsReq := &mode.Request{
			Domain: "ab",
			IP:     "1.2.3.4",
		}
		dtdnsMode.Respond(w, ddnsReq, mode.OutcomeInvalidDomain)

		response := strings.TrimSpace(w.Body.String())
		if response != "notfqdn" {
			t.Errorf("Expected 'notfqdn' for invalid domain, got '%s'", response)
		}
	})

	t.Run("DtDNS mode Respond returns 911 for system error", func(t *testing.T) {
		w := httptest.NewRecorder()
		ddnsReq := &mode.Request{
			Domain: "test.example.com",
			IP:     "1.2.3.4",
		}
		dtdnsMode.Respond(w, ddnsReq, mode.OutcomeSystemError)

		response := strings.TrimSpace(w.Body.String())
		if response != "911" {
			t.Errorf("Expected '911' for system error, got '%s'", response)
		}
	})

	t.Run("DtDNS mode with pw parameter for password", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/autodns.cfm?id=test.example.com&ip=1.2.3.4&pw=testpass", nil)
		req.RemoteAddr = "10.0.0.1:1234"

		// Add user to query params instead of Basic Auth
		req.URL.RawQuery = req.URL.RawQuery + "&user=testuser"

		ddnsReq, outcome := dtdnsMode.Prepare(req)
		if outcome != mode.OutcomeSuccess {
			t.Fatalf("expected success outcome, got %v", outcome)
		}

		if ddnsReq.Password != "testpass" {
			t.Errorf("Expected password 'testpass' from pw param, got '%s'", ddnsReq.Password)
		}
	})
}
