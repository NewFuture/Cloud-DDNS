package mode

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// Outcome represents the result of a mode processing step.
type Outcome int

const (
	OutcomeSuccess Outcome = iota
	OutcomeAuthFailure
	OutcomeInvalidDomain
	OutcomeSystemError
)

var debugMode atomic.Bool

// SetDebugMode toggles debug behaviours across modes.
func SetDebugMode(enabled bool) {
	debugMode.Store(enabled)
}

func isDebugMode() bool {
	return debugMode.Load()
}

// Request holds normalized DDNS parameters.
type Request struct {
	Username   string
	Password   string
	Domain     string
	IP         string
	Reqc       int
	RemoteAddr string
	Time       string
	Salt       string
	Sign       string
}

// Mode defines a protocol handler that can prepare, process, and respond to a DDNS HTTP request.
type Mode interface {
	Prepare(*http.Request) (*Request, Outcome)
	Process(*Request) Outcome
	Respond(http.ResponseWriter, *Request, Outcome)
}

// DynMode holds shared DynDNS-like behaviour (DynDNS, DtDNS, EasyDNS, Oray).
type DynMode struct {
	numericResponse bool
	debugLogf       func(format string, args ...interface{})
}

func NewDynMode(numeric bool, debug func(format string, args ...interface{})) *DynMode {
	return &DynMode{
		numericResponse: numeric,
		debugLogf:       debug,
	}
}

// Helpers shared by multiple modes.
func preferValue(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

func getQueryParam(q map[string][]string, names ...string) string {
	for _, name := range names {
		if val := q[name]; len(val) > 0 && val[0] != "" {
			return val[0]
		}
		for key, val := range q {
			if strings.EqualFold(key, name) && len(val) > 0 && val[0] != "" {
				return val[0]
			}
		}
	}
	return ""
}

func verifyPassword(storedPassword, inputPassword string) bool {
	if storedPassword == inputPassword {
		return true
	}

	md5Hash := md5.Sum([]byte(storedPassword))
	md5Str := hex.EncodeToString(md5Hash[:])
	if strings.EqualFold(md5Str, inputPassword) {
		return true
	}

	sha256Hash := sha256.Sum256([]byte(storedPassword))
	sha256Str := hex.EncodeToString(sha256Hash[:])
	if strings.EqualFold(sha256Str, inputPassword) {
		return true
	}

	if decoded, err := base64.StdEncoding.DecodeString(inputPassword); err == nil {
		if string(decoded) == storedPassword {
			return true
		}
	}

	inputMd5 := md5.Sum([]byte(inputPassword))
	inputMd5Str := hex.EncodeToString(inputMd5[:])
	if strings.EqualFold(storedPassword, inputMd5Str) {
		return true
	}

	inputSha256 := sha256.Sum256([]byte(inputPassword))
	inputSha256Str := hex.EncodeToString(inputSha256[:])
	if strings.EqualFold(storedPassword, inputSha256Str) {
		return true
	}

	if decoded, err := base64.StdEncoding.DecodeString(storedPassword); err == nil {
		if string(decoded) == inputPassword {
			return true
		}
	}

	return false
}

func parseReqc(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if value < 0 || value > 2 {
		return 0, strconv.ErrRange
	}
	return value, nil
}

func resolveRequestIP(reqc int, providedIP string, remoteAddr string) (string, error) {
	switch reqc {
	case 1:
		return "0.0.0.0", nil
	case 2:
		return extractRemoteIP(remoteAddr)
	case 0:
		if providedIP != "" && providedIP != "0.0.0.0" {
			return providedIP, nil
		}
		return extractRemoteIP(remoteAddr)
	default:
		return "", strconv.ErrRange
	}
}

func extractRemoteIP(remoteAddr string) (string, error) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return "", err
	}
	return host, nil
}

// Exported helpers for reuse and testing.
func ParseReqc(raw string) (int, error) { return parseReqc(raw) }
func ResolveRequestIP(reqc int, providedIP, remote string) (string, error) {
	return resolveRequestIP(reqc, providedIP, remote)
}
func GetQueryParam(q map[string][]string, names ...string) string { return getQueryParam(q, names...) }
func VerifyPassword(storedPassword, inputPassword string) bool {
	return verifyPassword(storedPassword, inputPassword)
}

func fallbackSalt(length int) string {
	if length <= 0 {
		return ""
	}
	hash := md5.Sum([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	hexStr := hex.EncodeToString(hash[:])
	if len(hexStr) < length {
		return hexStr
	}
	return hexStr[:length]
}
