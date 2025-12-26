package mode

import (
	"net/http/httptest"
	"testing"

	"github.com/NewFuture/CloudDDNS/pkg/config"
)

func TestGnuHTTPAllowsSaltWithoutSign(t *testing.T) {
	originalConfig := config.GlobalConfig
	defer func() { config.GlobalConfig = originalConfig }()

	config.GlobalConfig = config.Config{
		Users: []config.UserConfig{
			{
				Username: "debug",
				Password: "debug",
				Provider: "",
			},
		},
	}

	salt := "Y7Bu3WOpEm"
	pass := ComputeTCPHash("debug", salt)

	req := httptest.NewRequest("GET", "/cgi-bin/gdipupdt.cgi?salt="+salt+"&time=1766753435&sign=&user=debug&pass="+pass+"&domn=newfuture.lt&reqc=0&addr=1.2.3.4", nil)
	mode := NewGnuHTTPMode(func(format string, args ...interface{}) {})

	preparedReq, outcome := mode.Prepare(req)
	if outcome != OutcomeSuccess {
		t.Fatalf("Prepare outcome = %v, want %v", outcome, OutcomeSuccess)
	}

	processOutcome := mode.Process(preparedReq)
	if processOutcome == OutcomeAuthFailure {
		t.Fatalf("Process outcome should not be auth failure when sign is omitted, got %v", processOutcome)
	}
}
