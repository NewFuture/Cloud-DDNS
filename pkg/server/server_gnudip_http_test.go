package server

import (
	"crypto/md5"
	"fmt"
	"html"
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
		if !strings.Contains(response, `<meta name="salt"`) || !strings.Contains(response, `<meta name="time"`) || !strings.Contains(response, `<meta name="sign"`) {
			t.Fatalf("Expected GnuDIP challenge response with salt/time/sign, got '%s'", response)
		}
		salt := getMetaContent(response, "salt")
		if len(salt) != 10 {
			t.Fatalf("Expected salt length 10, got %d (%q)", len(salt), salt)
		}
	})

	t.Run("GnuDIP HTTP handshake on CGI path without domain", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/cgi-bin/gdipupdt.cgi?user=testuser", nil)
		req.RemoteAddr = "198.51.100.20:4567"
		w := httptest.NewRecorder()

		cgiHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		response := w.Body.String()
		if !strings.Contains(response, `<meta name="salt"`) || !strings.Contains(response, `<meta name="time"`) || !strings.Contains(response, `<meta name="sign"`) {
			t.Fatalf("Expected GnuDIP challenge response, got '%s'", response)
		}
		salt := getMetaContent(response, "salt")
		if len(salt) != 10 {
			t.Fatalf("Expected salt length 10, got %d (%q)", len(salt), salt)
		}
		if !strings.Contains(response, `198.51.100.20`) {
			t.Fatalf("Expected response to include client IP, got '%s'", response)
		}
	})

	t.Run("GnuDIP HTTP handshake on CGI path with BasicAuth only", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/cgi-bin/gdipupdt.cgi", nil)
		req.SetBasicAuth("testuser", "")
		req.RemoteAddr = "203.0.113.50:7890"
		w := httptest.NewRecorder()

		cgiHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		response := w.Body.String()
		if !strings.Contains(response, `<meta name="salt"`) || !strings.Contains(response, `<meta name="time"`) || !strings.Contains(response, `<meta name="sign"`) {
			t.Fatalf("Expected GnuDIP challenge response, got '%s'", response)
		}
		if !strings.Contains(response, `203.0.113.50`) {
			t.Fatalf("Expected response to include client IP, got '%s'", response)
		}
	})

	t.Run("GnuDIP HTTP second step with salt-based pass", func(t *testing.T) {
		timeParam := "1234567890"
		salt := "abcdefghij"
		inner := md5.Sum([]byte("testpass"))
		pass := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%x.%s", inner, salt))))
		sign := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", "testuser", timeParam, "testpass"))))
		req := httptest.NewRequest("GET", fmt.Sprintf("/nic/update?user=testuser&domn=test.example.com&time=%s&sign=%s&salt=%s&pass=%s&addr=1.2.3.4", timeParam, sign, salt, pass), nil)
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

	t.Run("GnuDIP HTTP second step with salt but wrong sign fails", func(t *testing.T) {
		timeParam := "1234567890"
		salt := "abcdefghij"
		inner := md5.Sum([]byte("testpass"))
		pass := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%x.%s", inner, salt))))
		req := httptest.NewRequest("GET", fmt.Sprintf("/nic/update?user=testuser&domn=test.example.com&time=%s&sign=deadbeef&salt=%s&pass=%s&addr=1.2.3.4", timeParam, salt, pass), nil)
		req.RemoteAddr = "203.0.113.10:4321"
		w := httptest.NewRecorder()

		handler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		if response != "1" {
			t.Fatalf("Expected auth failure numeric '1', got '%s'", response)
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

	t.Run("CGI path with no parameters returns auth failure", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/cgi-bin/gdipupdt.cgi", nil)
		req.RemoteAddr = "198.51.100.30:5678"
		w := httptest.NewRecorder()

		cgiHandler(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Expected status 200, got %d", w.Code)
		}

		response := strings.TrimSpace(w.Body.String())
		if response != "1" {
			t.Fatalf("Expected auth failure '1' when no user provided, got '%s'", response)
		}
	})

	t.Run("GnuDIP exact protocol flow: Step 1 and Step 2", func(t *testing.T) {
		// Step 1: REQUEST SALT with user parameter
		// Following the GnuDIP protocol: client requests salt/time/sign from server
		req1 := httptest.NewRequest("GET", "/cgi-bin/gdipupdt.cgi?user=testuser", nil)
		req1.RemoteAddr = "192.168.0.4:1234"
		w1 := httptest.NewRecorder()

		cgiHandler(w1, req1)

		if w1.Code != http.StatusOK {
			t.Fatalf("Step 1: Expected status 200, got %d", w1.Code)
		}

		response1 := w1.Body.String()

		// Verify step 1 response contains meta tags
		if !strings.Contains(response1, `<meta name="salt"`) ||
			!strings.Contains(response1, `<meta name="time"`) ||
			!strings.Contains(response1, `<meta name="sign"`) ||
			!strings.Contains(response1, `<meta name="addr"`) {
			t.Fatalf("Step 1: Expected HTML with salt/time/sign/addr meta tags, got: %s", response1)
		}

		// Extract values from step 1 response
		salt := getMetaContent(response1, "salt")
		timeParam := getMetaContent(response1, "time")
		serverSign := getMetaContent(response1, "sign")
		addr := getMetaContent(response1, "addr")

		t.Logf("Step 1 extracted: salt=%s time=%s sign=%s addr=%s", salt, timeParam, serverSign, addr)

		// Step 2: REQUEST UPDATE with all parameters
		// Compute password hash as: MD5(MD5(password) + "." + salt)
		inner := md5.Sum([]byte("testpass"))
		pass := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%x.%s", inner, salt))))

		// Build step 2 request with all params matching GnuDIP protocol
		req2URL := fmt.Sprintf("/cgi-bin/gdipupdt.cgi?salt=%s&time=%s&sign=%s&user=testuser&pass=%s&domn=test.example.com&reqc=0&addr=192.168.0.4",
			salt, timeParam, serverSign, pass)

		req2 := httptest.NewRequest("GET", req2URL, nil)
		req2.RemoteAddr = "192.168.0.4:1234"
		w2 := httptest.NewRecorder()

		cgiHandler(w2, req2)

		if w2.Code != http.StatusOK {
			t.Fatalf("Step 2: Expected status 200, got %d", w2.Code)
		}

		response2 := strings.TrimSpace(w2.Body.String())
		t.Logf("Step 2 response: %s", response2)

		// Should return numeric code (0=success, 1=failure, 2=offline)
		if response2 != "0" && response2 != "1" && response2 != "2" {
			t.Fatalf("Step 2: Expected numeric response (0/1/2), got '%s'", response2)
		}
	})

}

func getMetaContent(body, name string) string {
	prefix := fmt.Sprintf(`<meta name="%s" content="`, name)
	start := strings.Index(body, prefix)
	if start == -1 {
		return ""
	}
	start += len(prefix)
	end := strings.Index(body[start:], `"`)
	if end == -1 {
		return ""
	}
	return html.UnescapeString(body[start : start+end])
}
