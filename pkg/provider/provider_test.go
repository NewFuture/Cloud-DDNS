package provider

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NewFuture/CloudDDNS/pkg/config"
)

func TestGetProviderAliyun(t *testing.T) {
	userConfig := &config.UserConfig{
		Username: "test_ak",
		Password: "test_sk",
		Provider: "aliyun",
	}

	provider, err := GetProvider(userConfig)
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	if provider == nil {
		t.Fatal("Expected provider, got nil")
	}

	// Check if it's the right type
	if _, ok := provider.(*AliyunProvider); !ok {
		t.Error("Expected AliyunProvider type")
	}
}

func TestGetProviderTencent(t *testing.T) {
	userConfig := &config.UserConfig{
		Username: "test_id",
		Password: "test_token",
		Provider: "tencent",
	}

	provider, err := GetProvider(userConfig)
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	if provider == nil {
		t.Fatal("Expected provider, got nil")
	}

	// Check if it's the right type
	if _, ok := provider.(*TencentProvider); !ok {
		t.Error("Expected TencentProvider type")
	}
}

func TestGetProviderUnknown(t *testing.T) {
	userConfig := &config.UserConfig{
		Username: "test_user",
		Password: "test_pass",
		Provider: "unknown_provider",
	}

	provider, err := GetProvider(userConfig)
	if err == nil {
		t.Error("Expected error for unknown provider, got nil")
	}

	if provider != nil {
		t.Errorf("Expected nil provider for unknown type, got %v", provider)
	}

	expectedErrMsg := "unknown provider: unknown_provider"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestGetProviderEmptyProvider(t *testing.T) {
	userConfig := &config.UserConfig{
		Username: "test_user",
		Password: "test_pass",
		Provider: "",
	}

	provider, err := GetProvider(userConfig)
	if err == nil {
		t.Error("Expected error for empty provider, got nil")
	}

	if provider != nil {
		t.Errorf("Expected nil provider for empty provider, got %v", provider)
	}
}

func TestNewAliyunProvider(t *testing.T) {
	provider := NewAliyunProvider("test_ak", "test_sk")
	if provider == nil {
		t.Fatal("NewAliyunProvider returned nil")
	}

	if provider.accessKey != "test_ak" {
		t.Errorf("Expected accessKey 'test_ak', got '%s'", provider.accessKey)
	}

	if provider.secretKey != "test_sk" {
		t.Errorf("Expected secretKey 'test_sk', got '%s'", provider.secretKey)
	}
}

func TestNewTencentProvider(t *testing.T) {
	provider := NewTencentProvider("test_id", "test_key")
	if provider == nil {
		t.Fatal("NewTencentProvider returned nil")
	}

	if provider.secretId != "test_id" {
		t.Errorf("Expected secretId 'test_id', got '%s'", provider.secretId)
	}

	if provider.secretKey != "test_key" {
		t.Errorf("Expected secretKey 'test_key', got '%s'", provider.secretKey)
	}
}

func TestParseDomain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantBase string
		wantSub  string
		wantErr  bool
	}{
		// Standard TLD cases
		{
			name:     "Simple domain",
			input:    "example.com",
			wantBase: "example.com",
			wantSub:  "@",
			wantErr:  false,
		},
		{
			name:     "Subdomain",
			input:    "www.example.com",
			wantBase: "example.com",
			wantSub:  "www",
			wantErr:  false,
		},
		{
			name:     "Deep subdomain",
			input:    "api.v2.example.com",
			wantBase: "example.com",
			wantSub:  "api.v2",
			wantErr:  false,
		},
		// Country-code TLD cases
		{
			name:     "UK domain",
			input:    "example.co.uk",
			wantBase: "example.co.uk",
			wantSub:  "@",
			wantErr:  false,
		},
		{
			name:     "UK subdomain",
			input:    "www.example.co.uk",
			wantBase: "example.co.uk",
			wantSub:  "www",
			wantErr:  false,
		},
		{
			name:     "China domain",
			input:    "example.com.cn",
			wantBase: "example.com.cn",
			wantSub:  "@",
			wantErr:  false,
		},
		{
			name:     "China subdomain",
			input:    "api.example.com.cn",
			wantBase: "example.com.cn",
			wantSub:  "api",
			wantErr:  false,
		},
		{
			name:     "China deep subdomain",
			input:    "test.api.example.com.cn",
			wantBase: "example.com.cn",
			wantSub:  "test.api",
			wantErr:  false,
		},
		{
			name:     "Australia domain",
			input:    "example.com.au",
			wantBase: "example.com.au",
			wantSub:  "@",
			wantErr:  false,
		},
		{
			name:     "Japan domain",
			input:    "example.co.jp",
			wantBase: "example.co.jp",
			wantSub:  "@",
			wantErr:  false,
		},
		// Error cases
		{
			name:     "Single part",
			input:    "example",
			wantBase: "",
			wantSub:  "",
			wantErr:  true,
		},
		{
			name:     "Empty string",
			input:    "",
			wantBase: "",
			wantSub:  "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotBase, gotSub, err := ParseDomain(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotBase != tt.wantBase {
				t.Errorf("ParseDomain() gotBase = %v, want %v", gotBase, tt.wantBase)
			}
			if gotSub != tt.wantSub {
				t.Errorf("ParseDomain() gotSub = %v, want %v", gotSub, tt.wantSub)
			}
		})
	}
}

func TestGetProviderNowDNS(t *testing.T) {
	userConfig := &config.UserConfig{
		Username: "test@example.com",
		Password: "test_token",
		Provider: "nowdns",
	}

	provider, err := GetProvider(userConfig)
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	if provider == nil {
		t.Fatal("Expected provider, got nil")
	}

	// Check if it's the right type
	if _, ok := provider.(*NowDNSProvider); !ok {
		t.Error("Expected NowDNSProvider type")
	}
}

func TestNewNowDNSProvider(t *testing.T) {
	provider := NewNowDNSProvider("test@example.com", "test_token")
	if provider == nil {
		t.Fatal("NewNowDNSProvider returned nil")
	}

	if provider.email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got '%s'", provider.email)
	}

	if provider.password != "test_token" {
		t.Errorf("Expected password 'test_token', got '%s'", provider.password)
	}

	if provider.client == nil {
		t.Error("Expected http.Client, got nil")
	}

	if provider.endpoint != "https://now-dns.com/update" {
		t.Errorf("Expected endpoint 'https://now-dns.com/update', got '%s'", provider.endpoint)
	}
}

func TestNowDNSProviderUpdateRecord(t *testing.T) {
	tests := []struct {
		name           string
		response       string
		statusCode     int
		expectError    bool
		errorContains  string
		checkBasicAuth bool
	}{
		{
			name:           "Success good response",
			response:       "good 1.2.3.4",
			statusCode:     http.StatusOK,
			expectError:    false,
			checkBasicAuth: true,
		},
		{
			name:        "Success good without IP",
			response:    "good",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "Success nochg response",
			response:    "nochg 1.2.3.4",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "Success nochg without IP",
			response:    "nochg",
			statusCode:  http.StatusOK,
			expectError: false,
		},
		{
			name:          "Error nohost response",
			response:      "nohost",
			statusCode:    http.StatusOK,
			expectError:   true,
			errorContains: "host supplied not valid for given user",
		},
		{
			name:          "Error notfqdn response",
			response:      "notfqdn",
			statusCode:    http.StatusOK,
			expectError:   true,
			errorContains: "host supplied is not a valid hostname",
		},
		{
			name:          "Error badauth response",
			response:      "badauth",
			statusCode:    http.StatusOK,
			expectError:   true,
			errorContains: "invalid credentials",
		},
		{
			name:          "Error unexpected response",
			response:      "unknown_error",
			statusCode:    http.StatusOK,
			expectError:   true,
			errorContains: "unexpected response from now-dns.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method
				if r.Method != http.MethodGet {
					t.Errorf("Expected GET method, got %s", r.Method)
				}

				// Verify query parameters
				query := r.URL.Query()
				hostname := query.Get("hostname")
				myip := query.Get("myip")
				if hostname != "test.example.com" {
					t.Errorf("Expected hostname 'test.example.com', got '%s'", hostname)
				}
				if myip != "1.2.3.4" {
					t.Errorf("Expected myip '1.2.3.4', got '%s'", myip)
				}

				// Verify Basic Auth if required
				if tt.checkBasicAuth {
					user, pass, ok := r.BasicAuth()
					if !ok {
						t.Error("Expected Basic Auth, got none")
					}
					if user != "test@example.com" {
						t.Errorf("Expected user 'test@example.com', got '%s'", user)
					}
					if pass != "test_token" {
						t.Errorf("Expected pass 'test_token', got '%s'", pass)
					}
				}

				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.response))
			}))
			defer server.Close()

			// Create provider with test server endpoint
			provider := NewNowDNSProvider("test@example.com", "test_token")
			provider.endpoint = server.URL

			// Call UpdateRecord
			err := provider.UpdateRecord("test.example.com", "1.2.3.4")

			// Verify results
			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func TestNowDNSProviderUpdateRecordHTTPError(t *testing.T) {
	// Create provider with invalid endpoint to trigger HTTP error
	provider := NewNowDNSProvider("test@example.com", "test_token")
	provider.endpoint = "http://invalid.localhost:99999/update"

	err := provider.UpdateRecord("test.example.com", "1.2.3.4")
	if err == nil {
		t.Error("Expected error for HTTP failure, got nil")
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
