package provider

import (
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
		name           string
		input          string
		wantBase       string
		wantSub        string
		wantErr        bool
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
