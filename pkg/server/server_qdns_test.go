package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NewFuture/CloudDDNS/pkg/config"
)

// TestQDNSEndpoint verifies qDNS (/dyndns/update) handling and reqc=2 IP resolution.
func TestQDNSEndpoint(t *testing.T) {
	defer SetDebug(false)
	SetDebug(true)

	handler := http.HandlerFunc(handleDDNSUpdate)
	req := httptest.NewRequest("GET", "/dyndns/update?domn=qdns.example.com&reqc=2", nil)
	req.SetBasicAuth("debug", "debug")
	req.RemoteAddr = "198.51.100.10:9876"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	resp := strings.TrimSpace(w.Body.String())
	if resp != "good 198.51.100.10" && resp != "good" {
		t.Fatalf("Expected qDNS success response with remote IP or generic good, got %q", resp)
	}
}

func TestQDNSAuthReqcAndValidation(t *testing.T) {
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()
	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{Username: "user", Password: "pass", Provider: "aliyun"},
		},
	}
	handler := http.HandlerFunc(handleDDNSUpdate)

	t.Run("auth failure wrong password", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dyndns/update?domn=qdns.example.com&myip=1.2.3.4", nil)
		req.SetBasicAuth("user", "wrong")
		w := httptest.NewRecorder()
		handler(w, req)
		if strings.TrimSpace(w.Body.String()) != "badauth" {
			t.Fatalf("expected badauth, got %q", w.Body.String())
		}
	})

	t.Run("invalid domain", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dyndns/update?domn=ab&myip=1.2.3.4", nil)
		req.SetBasicAuth("user", "pass")
		w := httptest.NewRecorder()
		handler(w, req)
		if strings.TrimSpace(w.Body.String()) != "notfqdn" {
			t.Fatalf("expected notfqdn, got %q", w.Body.String())
		}
	})

	t.Run("reqc=1 offline returns good with zero ip numeric disabled", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true)
		req := httptest.NewRequest("GET", "/dyndns/update?domn=qdns.example.com&reqc=1", nil)
		req.SetBasicAuth("debug", "debug")
		req.RemoteAddr = "203.0.113.60:9999"
		w := httptest.NewRecorder()
		handler(w, req)
		resp := strings.TrimSpace(w.Body.String())
		if resp != "good 0.0.0.0" && resp != "good" {
			t.Fatalf("expected good with 0.0.0.0 for reqc=1, got %q", resp)
		}
	})

	t.Run("reqc=0 uses provided IP", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true)
		req := httptest.NewRequest("GET", "/dyndns/update?domn=qdns.example.com&myip=9.8.7.6", nil)
		req.SetBasicAuth("debug", "debug")
		w := httptest.NewRecorder()
		handler(w, req)
		resp := strings.TrimSpace(w.Body.String())
		if resp != "good 9.8.7.6" && resp != "good" {
			t.Fatalf("expected good with provided IP, got %q", resp)
		}
	})

	t.Run("provider error returns 911", func(t *testing.T) {
		config.GlobalConfig.Users[0].Provider = "unknown"
		req := httptest.NewRequest("GET", "/dyndns/update?domn=qdns.example.com&myip=1.2.3.4", nil)
		req.SetBasicAuth("user", "pass")
		w := httptest.NewRecorder()
		handler(w, req)
		if strings.TrimSpace(w.Body.String()) != "911" {
			t.Fatalf("expected 911, got %q", w.Body.String())
		}
	})
}
