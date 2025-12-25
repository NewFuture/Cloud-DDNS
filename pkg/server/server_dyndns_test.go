package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
