package mode

import (
	"testing"

	"github.com/NewFuture/CloudDDNS/pkg/config"
)

func TestBuildPassThroughUser(t *testing.T) {
	tests := []struct {
		name         string
		req          Request
		wantProvider string
		wantAccount  string
	}{
		{
			name: "provider from username prefix",
			req: Request{
				Username: "aliyun/account123",
				Password: "secret",
			},
			wantProvider: "aliyun",
			wantAccount:  "account123",
		},
		{
			name: "provider from host prefix with dash",
			req: Request{
				Username: "account456",
				Password: "secret",
				Host:     "tencent-ddns.example.com:8080",
			},
			wantProvider: "tencent",
			wantAccount:  "account456",
		},
		{
			name: "missing password not allowed",
			req: Request{
				Username: "aliyun/account123",
			},
		},
		{
			name: "unknown provider ignored",
			req: Request{
				Username: "unknown/account",
				Password: "secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := buildPassThroughUser(&tt.req)
			if tt.wantProvider == "" {
				if u != nil {
					t.Fatalf("expected nil passthrough user, got %+v", u)
				}
				return
			}
			if u == nil {
				t.Fatalf("expected passthrough user, got nil")
			}
			if u.Provider != tt.wantProvider || u.Username != tt.wantAccount {
				t.Fatalf("unexpected passthrough user %+v, want provider=%s account=%s", u, tt.wantProvider, tt.wantAccount)
			}
			if u.Password != tt.req.Password {
				t.Fatalf("expected password %q, got %q", tt.req.Password, u.Password)
			}
		})
	}
}

func TestDynModeProcessPassThroughToggle(t *testing.T) {
	original := config.GlobalConfig
	defer func() { config.GlobalConfig = original }()

	config.GlobalConfig = config.Config{}
	mode := NewDynMode(false, func(string, ...interface{}) {})

	req := &Request{
		Username: "tencent/account",
		Password: "secret",
		Domain:   "invalid",
		IP:       "1.2.3.4",
	}

	if outcome := mode.Process(req); outcome != OutcomeAuthFailure {
		t.Fatalf("expected auth failure when passthrough disabled, got %v", outcome)
	}

	config.GlobalConfig.Server.PassThrough = true
	if outcome := mode.Process(req); outcome != OutcomeSystemError {
		t.Fatalf("expected system error when passthrough enabled and provider update fails, got %v", outcome)
	}

	req.Host = "aliyun.example.com"
	req.Username = "account"
	if outcome := mode.Process(req); outcome != OutcomeSystemError {
		t.Fatalf("expected host-based passthrough to reach provider call, got %v", outcome)
	}
}
