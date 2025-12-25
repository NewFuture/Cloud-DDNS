package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NewFuture/CloudDDNS/pkg/config"
)

func TestEasyDNSHostIDParameterWithProviderError(t *testing.T) {
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()

	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{
				Username: "testuser",
				Password: "testpass",
				Provider: "aliyun",
			},
		},
	}

	handler := http.HandlerFunc(handleDDNSUpdate)

	req := httptest.NewRequest("GET", "/dyn/ez-ipupdate.php?action=edit&myip=1.2.3.4&wildcard=OFF&mx=&backmx=NO&host_id=easydns.new.com&user=testuser&pass=testpass", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	response := strings.TrimSpace(w.Body.String())
	if response != "NOSERVICE" {
		t.Fatalf("Expected 'NOSERVICE', got '%s'", response)
	}
}

func TestEasyDNSResponseCodes(t *testing.T) {
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()

	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{
				Username: "user",
				Password: "pass",
				Provider: "aliyun",
			},
		},
	}

	handler := http.HandlerFunc(handleDDNSUpdate)

	t.Run("Authentication failure", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dyn/generic.php?user=user&pass=wrong&hostname=test.example.com&myip=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if got := strings.TrimSpace(w.Body.String()); got != "NOACCESS" {
			t.Fatalf("expected NOACCESS, got %q", got)
		}
	})

	t.Run("Invalid domain", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/dyn/generic.php?user=user&pass=pass&hostname=ab&myip=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if got := strings.TrimSpace(w.Body.String()); got != "ILLEGAL INPUT" {
			t.Fatalf("expected ILLEGAL INPUT, got %q", got)
		}
	})

	t.Run("Provider error", func(t *testing.T) {
		originalConfigForProvider := config.GlobalConfig
		config.GlobalConfig = config.Config{
			Users: []config.UserConfig{
				{
					Username: "user",
					Password: "pass",
					Provider: "unknown",
				},
			},
		}
		defer func() { config.GlobalConfig = originalConfigForProvider }()

		req := httptest.NewRequest("GET", "/dyn/generic.php?user=user&pass=pass&hostname=test.example.com&myip=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if got := strings.TrimSpace(w.Body.String()); got != "NOSERVICE" {
			t.Fatalf("expected NOSERVICE, got %q", got)
		}
	})

	t.Run("Debug bypass success", func(t *testing.T) {
		defer SetDebug(false)
		SetDebug(true)
		req := httptest.NewRequest("GET", "/dyn/generic.php?user=debug&pass=debug&hostname=test.example.com&myip=1.2.3.4", nil)
		w := httptest.NewRecorder()

		handler(w, req)

		if got := strings.TrimSpace(w.Body.String()); got != "OK" {
			t.Fatalf("expected OK, got %q", got)
		}
	})
}
