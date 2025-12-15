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
