package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NewFuture/CloudDDNS/pkg/config"
)

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

func TestNoIPMoreScenarios(t *testing.T) {
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()

	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{Username: "user", Password: "pass", Provider: "aliyun"},
		},
	}
	handler := http.HandlerFunc(handleDDNSUpdate)

	t.Run("success with debug bypass", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true)
		req := httptest.NewRequest("GET", "/nic/update?hostname=test.example.com&myip=1.2.3.4", nil)
		req.SetBasicAuth("debug", "debug")
		w := httptest.NewRecorder()
		handler(w, req)
		resp := strings.TrimSpace(w.Body.String())
		if resp != "good 1.2.3.4" && resp != "good" {
			t.Fatalf("expected good response, got %q", resp)
		}
	})

	t.Run("invalid domain", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/nic/update?hostname=ab&myip=1.2.3.4", nil)
		req.SetBasicAuth("user", "pass")
		w := httptest.NewRecorder()
		handler(w, req)
		if strings.TrimSpace(w.Body.String()) != "notfqdn" {
			t.Fatalf("expected notfqdn, got %q", w.Body.String())
		}
	})

	t.Run("auto-detect IP when myip missing", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true)
		req := httptest.NewRequest("GET", "/nic/update?hostname=test.example.com", nil)
		req.SetBasicAuth("debug", "debug")
		req.RemoteAddr = "198.51.100.77:1234"
		w := httptest.NewRecorder()
		handler(w, req)
		resp := strings.TrimSpace(w.Body.String())
		if resp != "good 198.51.100.77" && resp != "good" {
			t.Fatalf("expected good with remote IP, got %q", resp)
		}
	})

	t.Run("provider error returns 911", func(t *testing.T) {
		config.GlobalConfig.Users[0].Provider = "unknown"
		req := httptest.NewRequest("GET", "/nic/update?hostname=test.example.com&myip=1.2.3.4", nil)
		req.SetBasicAuth("user", "pass")
		w := httptest.NewRecorder()
		handler(w, req)
		if strings.TrimSpace(w.Body.String()) != "911" {
			t.Fatalf("expected 911, got %q", w.Body.String())
		}
	})
}
