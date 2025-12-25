package server

import (
	"log"
	"net"
	"net/http"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/provider"
)

// DDNSRequest carries normalized parameters for a DDNS update request.
type DDNSRequest struct {
	Username        string
	Password        string
	Domain          string
	IP              string
	Reqc            int
	NumericResponse bool
	RemoteAddr      string
}

// DDNSService defines the behaviour needed by protocol frontends to process DDNS updates.
type DDNSService interface {
	PrepareHTTPRequest(r *http.Request, numeric bool) (*DDNSRequest, responseOutcome)
	Process(req *DDNSRequest) responseOutcome
}

type defaultDDNSService struct{}

func newDefaultDDNSService() DDNSService {
	return &defaultDDNSService{}
}

func preferValue(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}

// PrepareHTTPRequest extracts credentials, domain and IP information from an HTTP request.
// It supports both HTTP Basic Auth and URL query parameters, resolves missing IPs using RemoteAddr,
// and validates domain/IP formats before handing off to the provider layer.
func (s *defaultDDNSService) PrepareHTTPRequest(r *http.Request, numeric bool) (*DDNSRequest, responseOutcome) {
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
		return &DDNSRequest{Reqc: 0}, responseSystemError
	}

	resolvedIP, err := resolveRequestIP(reqc, ip, r.RemoteAddr)
	if err != nil {
		log.Printf("Invalid RemoteAddr format: %q, error: %v", r.RemoteAddr, err)
		return &DDNSRequest{Reqc: reqc}, responseSystemError
	}

	if net.ParseIP(resolvedIP) == nil {
		log.Printf("Invalid IP address: %q", resolvedIP)
		return &DDNSRequest{Reqc: reqc}, responseSystemError
	}

	if domain == "" || len(domain) < 3 || len(domain) > 253 {
		log.Printf("Invalid domain: %q", domain)
		return &DDNSRequest{Reqc: reqc}, responseInvalidDomain
	}

	req := &DDNSRequest{
		Username:        username,
		Password:        password,
		Domain:          domain,
		IP:              resolvedIP,
		Reqc:            reqc,
		NumericResponse: numeric,
		RemoteAddr:      r.RemoteAddr,
	}

	debugLogf("Credential source basicAuth=%t user=%s queryUser=%s", basicAuthProvided, headerUser, queryUser)
	debugLogf("Prepared DDNSRequest user=%s domain=%s ip=%s reqc=%d numeric=%t remote=%s", username, domain, resolvedIP, reqc, numeric, r.RemoteAddr)
	return req, responseSuccess
}

// Process authenticates the user and executes the provider update.
func (s *defaultDDNSService) Process(req *DDNSRequest) responseOutcome {
	u := config.GetUser(req.Username)
	if u == nil || !verifyPassword(u.Password, req.Password) {
		log.Printf("Authentication failed for user: %q", req.Username)
		debugLogf("DDNS service authentication failed for user=%s", req.Username)
		return responseAuthFailure
	}
	debugLogf("DDNS service authentication succeeded for user=%s", req.Username)

	p, err := provider.GetProvider(u)
	if err != nil {
		log.Printf("Provider error for user %q: %v", req.Username, err)
		debugLogf("DDNS service provider init failed for user=%s provider=%s error=%v", req.Username, u.Provider, err)
		return responseSystemError
	}
	debugLogf("DDNS service provider initialized for user=%s provider=%s", req.Username, u.Provider)

	if err := p.UpdateRecord(req.Domain, req.IP); err != nil {
		log.Printf("UpdateRecord error for domain %q and ip %q: %v", req.Domain, req.IP, err)
		debugLogf("DDNS service DNS update failed for domain=%s ip=%s error=%v", req.Domain, req.IP, err)
		return responseSystemError
	}

	log.Printf("Successfully updated %s to %s", req.Domain, req.IP)
	debugLogf("DDNS service DNS update succeeded for domain=%s ip=%s", req.Domain, req.IP)
	return responseSuccess
}
