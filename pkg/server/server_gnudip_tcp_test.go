package server

import (
	"testing"

	"github.com/NewFuture/CloudDDNS/pkg/server/mode"
)

func TestGnuDIPHashMatchesSampleLog(t *testing.T) {
	password := "debug"
	salt := "1766672054.192016939"
	expectedClientHash := "9045b061cc3e4531c14d7b0e8200675a"

	clientHash := mode.ComputeTCPHash(password, salt)

	if clientHash != expectedClientHash {
		t.Fatalf("expected hash %s, got %s", expectedClientHash, clientHash)
	}
}
