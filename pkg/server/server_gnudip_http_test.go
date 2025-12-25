package server

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NewFuture/CloudDDNS/pkg/config"
)

func TestGnuDIPHTTPHandlers(t *testing.T) {
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
	cgiHandler := http.HandlerFunc(handleCGIUpdate)

	t.Run("GnuDIP HTTP handshake on /nic/update without password", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/nic/update?user=testuser&domn=test.example.com", nil)
		req.RemoteAddr = "203.0.113.10:4321"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		response := w.Body.String()
		if !strings.Contains(response, `<meta name="retc" content="0">`) || !strings.Contains(response, `<meta name="sign"`) {
			t.Fatalf("Expected GnuDIP challenge response, got '%s'", response)
		}
	})

	t.Run("GnuDIP HTTP second step with sign", func(t *testing.T) {
		timeParam := "1234567890"
		sign := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", "testuser", timeParam, "testpass"))))
		req := httptest.NewRequest("GET", fmt.Sprintf("/nic/update?user=testuser&domn=test.example.com&time=%s&sign=%s&addr=1.2.3.4", timeParam, sign), nil)
		req.RemoteAddr = "203.0.113.10:4321"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		if response == "badauth" {
			t.Fatalf("Expected GnuDIP processing, got DynDNS auth failure response")
		}
		if response != "0" && response != "1" {
			t.Fatalf("Expected GnuDIP numeric response, got '%s'", response)
		}
	})

	t.Run("CGI path returns numeric codes for update", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/cgi-bin/gdipupdt.cgi?user=testuser&pass=testpass&domn=test.example.com&addr=1.2.3.4&reqc=0", nil)
		w := httptest.NewRecorder()

		cgiHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		if response != "0" && response != "1" {
			t.Fatalf("Expected numeric '0' or '1', got '%s'", response)
		}
	})

	t.Run("CGI path handles offline reqc", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/cgi-bin/gdipupdt.cgi?user=testuser&pass=testpass&domn=test.example.com&reqc=1", nil)
		req.RemoteAddr = "10.20.30.40:2345"
		w := httptest.NewRecorder()

		cgiHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		if response != "0" && response != "1" && response != "2" {
			t.Fatalf("Expected numeric response, got '%s'", response)
		}
	})

	t.Run("CGI path auto-detects IP when missing", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/cgi-bin/gdipupdt.cgi?user=testuser&pass=testpass&domn=test.example.com&reqc=2", nil)
		req.RemoteAddr = "192.0.2.10:4567"
		w := httptest.NewRecorder()

		cgiHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		if response != "0" && response != "1" {
			t.Fatalf("Expected numeric '0' or '1', got '%s'", response)
		}
	})
}
