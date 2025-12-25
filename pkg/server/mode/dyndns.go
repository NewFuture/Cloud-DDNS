package mode

import (
	"log"
	"net"
	"net/http"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/provider"
)

// DynMode handles DynDNS2/GnuDIP HTTP style requests.
type DynMode struct {
	numericResponse bool
	debugLogf       func(format string, args ...interface{})
}

func NewDynDNSMode(debug func(format string, args ...interface{})) Mode {
	return NewDynMode(false, debug)
}

func NewGnuHTTPMode(debug func(format string, args ...interface{})) Mode {
	return NewDynMode(true, debug)
}

func NewDtDNSMode(debug func(format string, args ...interface{})) Mode {
	return NewDynMode(false, debug)
}

func NewEasyDNSMode(debug func(format string, args ...interface{})) Mode {
	return NewDynMode(false, debug)
}

func NewOrayMode(debug func(format string, args ...interface{})) Mode {
	return NewDynMode(false, debug)
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
