package mode

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/provider"
)

// Outcome represents the result of a mode processing step.
type Outcome int

const (
	OutcomeSuccess Outcome = iota
	OutcomeAuthFailure
	OutcomeInvalidDomain
	OutcomeSystemError
)

// Request holds normalized DDNS parameters.
type Request struct {
	Username   string
	Password   string
	Domain     string
	IP         string
	Reqc       int
	RemoteAddr string
}

// Mode defines a protocol handler that can prepare, process, and respond to a DDNS HTTP request.
type Mode interface {
	Prepare(*http.Request) (*Request, Outcome)
	Process(*Request) Outcome
	Respond(http.ResponseWriter, *Request, Outcome)
}

// DynMode handles DynDNS2/GnuDIP HTTP style requests.
type DynMode struct {
	numericResponse bool
	debugLogf       func(format string, args ...interface{})
}

// NewDynMode creates a DynMode instance.
func NewDynMode(numeric bool, debug func(format string, args ...interface{})) *DynMode {
	return &DynMode{
		numericResponse: numeric,
		debugLogf:       debug,
	}
}

// Prepare extracts credentials, domain and IP info from the HTTP request.
func (m *DynMode) Prepare(r *http.Request) (*Request, Outcome) {
	q := r.URL.Query()

	headerUser, headerPass, basicAuthProvided := r.BasicAuth()
	queryUser := getQueryParam(q, "user", "username", "usr", "name")
	queryPass := getQueryParam(q, "pass", "password", "pwd")

	username := preferValue(headerUser, queryUser)
	password := preferValue(headerPass, queryPass)

	domain := getQueryParam(q, "domn", "domain", "hostname", "host")
	ip := getQueryParam(q, "addr", "myip", "ip")
	reqcStr := getQueryParam(q, "reqc")
	reqc, err := parseReqc(reqcStr)
	if err != nil {
		log.Printf("Invalid reqc value %q: %v", reqcStr, err)
		return &Request{Reqc: 0}, OutcomeSystemError
	}

	resolvedIP, err := resolveRequestIP(reqc, ip, r.RemoteAddr)
	if err != nil {
		log.Printf("Invalid RemoteAddr format: %q, error: %v", r.RemoteAddr, err)
		return &Request{Reqc: reqc}, OutcomeSystemError
	}

	if net.ParseIP(resolvedIP) == nil {
		log.Printf("Invalid IP address: %q", resolvedIP)
		return &Request{Reqc: reqc}, OutcomeSystemError
	}

	if domain == "" || len(domain) < 3 || len(domain) > 253 {
		log.Printf("Invalid domain: %q", domain)
		return &Request{Reqc: reqc}, OutcomeInvalidDomain
	}

	req := &Request{
		Username:   username,
		Password:   password,
		Domain:     domain,
		IP:         resolvedIP,
		Reqc:       reqc,
		RemoteAddr: r.RemoteAddr,
	}

	m.debugLogf("Credential source basicAuth=%t headerProvided=%t queryProvided=%t", basicAuthProvided, headerUser != "", queryUser != "")
	m.debugLogf("Prepared DDNS Request domain=%s ip=%s reqc=%d numeric=%t remote=%s", domain, resolvedIP, reqc, m.numericResponse, r.RemoteAddr)
	return req, OutcomeSuccess
}

// Process authenticates the user and executes the provider update.
func (m *DynMode) Process(req *Request) Outcome {
	u := config.GetUser(req.Username)
	if u == nil || !verifyPassword(u.Password, req.Password) {
		log.Printf("Authentication failed for user: %q", req.Username)
		m.debugLogf("DDNS mode authentication failed for user=%s", req.Username)
		return OutcomeAuthFailure
	}
	m.debugLogf("DDNS mode authentication succeeded for user=%s", req.Username)

	p, err := provider.GetProvider(u)
	if err != nil {
		log.Printf("Provider error for user %q: %v", req.Username, err)
		m.debugLogf("DDNS mode provider init failed for user=%s provider=%s error=%v", req.Username, u.Provider, err)
		return OutcomeSystemError
	}
	m.debugLogf("DDNS mode provider initialized for user=%s provider=%s", req.Username, u.Provider)

	if err := p.UpdateRecord(req.Domain, req.IP); err != nil {
		log.Printf("UpdateRecord error for domain %q and ip %q: %v", req.Domain, req.IP, err)
		m.debugLogf("DDNS mode DNS update failed for domain=%s ip=%s error=%v", req.Domain, req.IP, err)
		return OutcomeSystemError
	}

	log.Printf("Successfully updated %s to %s", req.Domain, req.IP)
	m.debugLogf("DDNS mode DNS update succeeded for domain=%s ip=%s", req.Domain, req.IP)
	return OutcomeSuccess
}

// Respond writes protocol-specific responses.
func (m *DynMode) Respond(w http.ResponseWriter, req *Request, outcome Outcome) {
	reqc := 0
	if req != nil {
		reqc = req.Reqc
	}

	var body string
	switch outcome {
	case OutcomeSuccess:
		if m.numericResponse {
			if reqc == 1 {
				body = "2"
			} else {
				body = "0"
			}
		} else {
			body = "good " + req.IP
		}
	case OutcomeAuthFailure:
		if m.numericResponse {
			body = "1"
		} else {
			body = "badauth"
		}
	case OutcomeInvalidDomain:
		if m.numericResponse {
			body = "1"
		} else {
			body = "notfqdn"
		}
	default:
		if m.numericResponse {
			body = "1"
		} else {
			body = "911"
		}
	}

	if _, err := w.Write([]byte(body)); err != nil {
		log.Printf("HTTP Write Error: %v", err)
	}
}

// helpers

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
