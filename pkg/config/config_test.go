package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.yaml")

	configContent := `server:
  tcp_port: 3495
  http_port: 8080

users:
  - username: "test_user"
    password: "test_password"
    provider: "aliyun"
  - username: "user2"
    password: "pass2"
    provider: "tencent"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	// Test loading config
	err = LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify server config
	if GlobalConfig.Server.TCPPort != 3495 {
		t.Errorf("Expected TCP port 3495, got %d", GlobalConfig.Server.TCPPort)
	}
	if GlobalConfig.Server.HTTPPort != 8080 {
		t.Errorf("Expected HTTP port 8080, got %d", GlobalConfig.Server.HTTPPort)
	}

	// Verify users
	if len(GlobalConfig.Users) != 2 {
		t.Errorf("Expected 2 users, got %d", len(GlobalConfig.Users))
	}

	if GlobalConfig.Users[0].Username != "test_user" {
		t.Errorf("Expected username 'test_user', got '%s'", GlobalConfig.Users[0].Username)
	}
	if GlobalConfig.Users[0].Provider != "aliyun" {
		t.Errorf("Expected provider 'aliyun', got '%s'", GlobalConfig.Users[0].Provider)
	}
}

func TestLoadConfigNonExistent(t *testing.T) {
	err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error loading non-existent config, got nil")
	}
}

func TestLoadConfigInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidContent := `server:
  tcp_port: not_a_number
  invalid yaml content {[
`

	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error loading invalid YAML, got nil")
	}
}

func TestGetUser(t *testing.T) {
	// Save original config and restore after test
	originalConfig := GlobalConfig
	defer func() { GlobalConfig = originalConfig }()

	// Setup test config
	GlobalConfig = Config{
		Users: []UserConfig{
			{Username: "user1", Password: "pass1", Provider: "aliyun"},
			{Username: "user2", Password: "pass2", Provider: "tencent"},
		},
	}

	// Test finding existing user
	user := GetUser("user1")
	if user == nil {
		t.Fatal("Expected to find user1, got nil")
	}
	if user.Username != "user1" {
		t.Errorf("Expected username 'user1', got '%s'", user.Username)
	}
	if user.Password != "pass1" {
		t.Errorf("Expected password 'pass1', got '%s'", user.Password)
	}

	// Test finding another user
	user = GetUser("user2")
	if user == nil {
		t.Fatal("Expected to find user2, got nil")
	}
	if user.Provider != "tencent" {
		t.Errorf("Expected provider 'tencent', got '%s'", user.Provider)
	}

	// Test non-existent user
	user = GetUser("nonexistent")
	if user != nil {
		t.Errorf("Expected nil for non-existent user, got %v", user)
	}
}

func TestGetUserEmptyConfig(t *testing.T) {
	// Save original config and restore after test
	originalConfig := GlobalConfig
	defer func() { GlobalConfig = originalConfig }()

	GlobalConfig = Config{Users: []UserConfig{}}

	user := GetUser("anyuser")
	if user != nil {
		t.Errorf("Expected nil for empty config, got %v", user)
	}
}
