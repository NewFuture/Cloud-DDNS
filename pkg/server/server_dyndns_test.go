package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NewFuture/CloudDDNS/pkg/config"
)

// TestDynDNSEndpoint covers the generic DynDNS style /update handler.
func TestDynDNSEndpoint(t *testing.T) {
	defer SetDebug(false)
	SetDebug(true)

	handler := http.HandlerFunc(handleDDNSUpdate)
	req := httptest.NewRequest("GET", "/update?hostname=test.example.com&myip=1.2.3.4", nil)
	req.SetBasicAuth("debug", "debug")
	req.RemoteAddr = "203.0.113.1:4567"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	resp := strings.TrimSpace(w.Body.String())
	if resp != "good 1.2.3.4" && resp != "good" {
		t.Fatalf("Expected DynDNS success response 'good 1.2.3.4' or 'good', got %q", resp)
	}
}

func TestDynDNSAuthAndValidation(t *testing.T) {
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()
	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{Username: "user", Password: "pass", Provider: "aliyun"},
		},
	}
	handler := http.HandlerFunc(handleDDNSUpdate)

	t.Run("auth failure wrong password", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/update?hostname=test.example.com&myip=1.2.3.4", nil)
		req.SetBasicAuth("user", "wrong")
		w := httptest.NewRecorder()
		handler(w, req)
		if strings.TrimSpace(w.Body.String()) != "badauth" {
			t.Fatalf("expected badauth, got %q", w.Body.String())
		}
	})

	t.Run("auth failure unknown user", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/update?hostname=test.example.com&myip=1.2.3.4", nil)
		req.SetBasicAuth("unknown", "any")
		w := httptest.NewRecorder()
		handler(w, req)
		if strings.TrimSpace(w.Body.String()) != "badauth" {
			t.Fatalf("expected badauth for unknown user, got %q", w.Body.String())
		}
	})

	t.Run("invalid domain too short", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/update?hostname=ab&myip=1.2.3.4", nil)
		req.SetBasicAuth("user", "pass")
		w := httptest.NewRecorder()
		handler(w, req)
		if strings.TrimSpace(w.Body.String()) != "notfqdn" {
			t.Fatalf("expected notfqdn, got %q", w.Body.String())
		}
	})

	t.Run("missing domain", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/update?myip=1.2.3.4", nil)
		req.SetBasicAuth("user", "pass")
		w := httptest.NewRecorder()
		handler(w, req)
		if strings.TrimSpace(w.Body.String()) != "notfqdn" {
			t.Fatalf("expected notfqdn for missing domain, got %q", w.Body.String())
		}
	})

	t.Run("system error provider failure", func(t *testing.T) {
		originalProvider := config.GlobalConfig.Users[0].Provider
		defer func() { config.GlobalConfig.Users[0].Provider = originalProvider }()
		config.GlobalConfig.Users[0].Provider = "unknown"
		req := httptest.NewRequest("GET", "/update?hostname=test.example.com&myip=1.2.3.4", nil)
		req.SetBasicAuth("user", "pass")
		w := httptest.NewRecorder()
		handler(w, req)
		if strings.TrimSpace(w.Body.String()) != "911" {
			t.Fatalf("expected 911 on provider error, got %q", w.Body.String())
		}
	})

	t.Run("auto-detect IP from RemoteAddr", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true)
		req := httptest.NewRequest("GET", "/update?hostname=test.example.com", nil)
		req.SetBasicAuth("debug", "debug")
		req.RemoteAddr = "203.0.113.50:8888"
		w := httptest.NewRecorder()
		handler(w, req)
		resp := strings.TrimSpace(w.Body.String())
		if resp != "good 203.0.113.50" {
			t.Fatalf("expected 'good 203.0.113.50' with remote IP, got %q", resp)
		}
	})
}
