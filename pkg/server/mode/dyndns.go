package mode

import (
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/provider"
)

// Prepare extracts credentials, domain and IP info from the HTTP request.
func (m *DynMode) Prepare(r *http.Request) (*Request, Outcome) {
	q := r.URL.Query()

	headerUser, headerPass, basicAuthProvided := r.BasicAuth()
	queryUser := getQueryParam(q, "user", "username", "usr", "name")
	queryPass := getQueryParam(q, "pass", "password", "pwd", "pw")

	username := preferValue(headerUser, queryUser)
	password := preferValue(headerPass, queryPass)

	// Domain aliases: domn/domain/hostname/host (standard), id (DtDNS), host_id (EasyDNS).
	domain := getQueryParam(q, "domn", "domain", "hostname", "host", "id", "host_id")
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
		Host:       r.Host,
	}

	m.debugLogf("Credential source basicAuth=%t headerProvided=%t queryProvided=%t", basicAuthProvided, headerUser != "", queryUser != "")
	m.debugLogf("Prepared DDNS Request domain=%s ip=%s reqc=%d numeric=%t remote=%s", domain, resolvedIP, reqc, m.numericResponse, r.RemoteAddr)
	return req, OutcomeSuccess
}

// Process authenticates the user and executes the provider update.
func (m *DynMode) Process(req *Request) Outcome {
	if isDebugMode() && req.Username == "debug" && req.Password == "debug" {
		m.debugLogf("Debug bypass for domain=%s ip=%s", req.Domain, req.IP)
		return OutcomeSuccess
	}

	u := config.GetUser(req.Username)
	passthrough := false
	if u == nil && config.GlobalConfig.Server.PassThrough {
		if ptUser := buildPassThroughUser(req); ptUser != nil {
			u = ptUser
			passthrough = true
			m.debugLogf("DDNS passthrough enabled for provider=%s account=%s host=%s", ptUser.Provider, ptUser.Username, req.Host)
		}
	}

	if u == nil {
		log.Printf("Authentication failed for user: %q", req.Username)
		m.debugLogf("DDNS mode authentication failed for user=%s", req.Username)
		return OutcomeAuthFailure
	}

	// In passthrough mode we trust the provider credentials in the request and let the upstream provider validate them.
	if !passthrough && !verifyPassword(u.Password, req.Password) {
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
			if req != nil && req.IP != "" {
				body = "good " + req.IP
			} else {
				body = "good"
			}
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

func buildPassThroughUser(req *Request) *config.UserConfig {
	providerName, account := detectProviderAndAccount(req.Username, req.Host)
	if providerName == "" || account == "" || req.Password == "" {
		return nil
	}

	if !isSupportedProvider(providerName) {
		return nil
	}

	return &config.UserConfig{
		Username: account,
		Password: req.Password,
		Provider: providerName,
	}
}

func detectProviderAndAccount(username, host string) (string, string) {
	if providerName, account := providerFromUsername(username); providerName != "" && account != "" {
		return providerName, account
	}

	if providerName := providerFromHost(host); providerName != "" && username != "" {
		return providerName, username
	}

	return "", ""
}

func providerFromUsername(username string) (string, string) {
	parts := strings.SplitN(username, "/", 2)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return strings.ToLower(parts[0]), parts[1]
	}
	return "", ""
}

// providerFromHost extracts provider prefix from hosts like "aliyun.example.com" or "tencent-ddns.example.com".
func providerFromHost(host string) string {
	host = normalizeHost(host)
	if host == "" {
		return ""
	}

	lowerHost := strings.ToLower(host)
	dotIdx := strings.Index(lowerHost, ".")
	dashIdx := strings.Index(lowerHost, "-")

	switch {
	case dotIdx > 0 && dashIdx > 0:
		if dotIdx < dashIdx {
			return lowerHost[:dotIdx]
		}
		return lowerHost[:dashIdx]
	case dotIdx > 0:
		return lowerHost[:dotIdx]
	case dashIdx > 0:
		return lowerHost[:dashIdx]
	}

	return ""
}

func normalizeHost(host string) string {
	if host == "" {
		return ""
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		return h
	}
	return host
}

func isSupportedProvider(providerName string) bool {
	_, err := provider.GetProvider(&config.UserConfig{Provider: strings.ToLower(providerName)})
	return err == nil
}
