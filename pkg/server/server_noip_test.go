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
