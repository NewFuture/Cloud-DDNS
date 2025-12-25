package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
