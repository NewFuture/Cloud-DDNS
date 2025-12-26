package mode

import (
	"net/http/httptest"
	"net/url"
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
	const observedFailureTimeParam = "1766753435" // timestamp captured from reported failure case
	pass := ComputeTCPHash("debug", salt)

	params := url.Values{}
	params.Set("salt", salt)
	params.Set("time", observedFailureTimeParam)
	params.Set("sign", "")
	params.Set("user", "debug")
	params.Set("pass", pass)
	params.Set("domn", "newfuture.lt")
	params.Set("reqc", "0")
	params.Set("addr", "1.2.3.4")

	req := httptest.NewRequest("GET", "/cgi-bin/gdipupdt.cgi?"+params.Encode(), nil)
	mode := NewGnuHTTPMode(t.Logf)

	preparedReq, outcome := mode.Prepare(req)
	if outcome != OutcomeSuccess {
		t.Fatalf("Prepare outcome = %v, want %v", outcome, OutcomeSuccess)
	}

	processOutcome := mode.Process(preparedReq)
	if processOutcome == OutcomeAuthFailure {
		t.Fatalf("Process outcome should not be auth failure when sign is omitted, got %v", processOutcome)
	}
}
