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

// TestNoIPNicUpdate ensures No-IP compatible /nic/update uses DynDNS flow.
func TestNoIPNicUpdate(t *testing.T) {
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()

	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{
				Username: "noipuser",
				Password: "noippass",
				Provider: "aliyun",
			},
		},
	}

	handler := http.HandlerFunc(handleDDNSUpdate)
	req := httptest.NewRequest("GET", "/nic/update?hostname=test.example.com&myip=1.2.3.4", nil)
	req.SetBasicAuth("noipuser", "wrongpass")
	req.RemoteAddr = "203.0.113.20:2222"
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	resp := strings.TrimSpace(w.Body.String())
	if resp != "badauth" {
		t.Fatalf("Expected badauth for invalid No-IP credentials, got %q", resp)
	}
}
