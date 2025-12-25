package mode

import (
	"crypto/md5"
	"crypto/rand"
	"fmt"
	"html"
	"log"
	"math/big"
	"net/http"
	"time"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/provider"
)

// GnuHTTPMode implements the two-step GnuDIP HTTP challenge flow.
// Step 1: client requests without pass/sign -> server returns meta tags with retc/time/sign/addr.
// Step 2: client requests with pass/sign (md5(user:time:secret)) -> server validates and updates.
type GnuHTTPMode struct {
	debugLogf func(format string, args ...interface{})
}

func NewGnuHTTPMode(debug func(format string, args ...interface{})) Mode {
	return &GnuHTTPMode{debugLogf: debug}
}

func (m *GnuHTTPMode) Prepare(r *http.Request) (*Request, Outcome) {
	q := r.URL.Query()

	user := GetQueryParam(q, "user", "username")
	domain := GetQueryParam(q, "domn", "domain", "hostname", "host")
	pass := GetQueryParam(q, "pass", "password", "pwd")
	sign := GetQueryParam(q, "sign")
	timeParam := GetQueryParam(q, "time")
	salt := GetQueryParam(q, "salt")
	ip := GetQueryParam(q, "addr", "myip", "ip")
	authPresent := q.Has("pass") || q.Has("password") || q.Has("pwd") || q.Has("sign")
	isHandshake := !authPresent

	reqc := 0
	resolvedIP, err := resolveRequestIP(reqc, ip, r.RemoteAddr)
	if err != nil {
		log.Printf("Invalid RemoteAddr format: %q, error: %v", r.RemoteAddr, err)
		return &Request{Reqc: reqc}, OutcomeSystemError
	}

	req := &Request{
		Username:   user,
		Password:   pass,
		Domain:     domain,
		IP:         resolvedIP,
		Reqc:       reqc,
		RemoteAddr: r.RemoteAddr,
		Time:       timeParam,
		Salt:       salt,
		Sign:       sign,
	}

	if (!isHandshake && domain == "") || (domain != "" && (len(domain) < 3 || len(domain) > 253)) {
		log.Printf("Invalid domain: %q", domain)
		return req, OutcomeInvalidDomain
	}

	logMsg := "GnuHTTP prepare user=%s domain=%s ip=%s time=%s remote=%s"
	if isHandshake {
		logMsg = "GnuHTTP handshake prepare user=%s domain=%s ip=%s time=%s remote=%s"
	}
	m.debugLogf(logMsg, user, domain, resolvedIP, timeParam, r.RemoteAddr)
	return req, OutcomeSuccess
}

func (m *GnuHTTPMode) Process(req *Request) Outcome {
	if isDebugMode() && req.Username == "debug" && req.Password == "debug" {
		m.debugLogf("GnuHTTP debug bypass for domain=%s ip=%s", req.Domain, req.IP)
		return OutcomeSuccess
	}

	u := config.GetUser(req.Username)
	if u == nil {
		log.Printf("Authentication failed for user: %q", req.Username)
		return OutcomeAuthFailure
	}

	// Two-step: if Password is empty, defer auth to Respond (will issue challenge).
	if req.Password == "" {
		return OutcomeAuthFailure
	}

	if req.Salt != "" {
		if (req.Sign != "" && req.Time == "") || (req.Sign == "" && req.Time != "") {
			return OutcomeAuthFailure
		}
		if req.Sign != "" {
			expectedSign := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", req.Username, req.Time, u.Password))))
			if req.Sign != expectedSign {
				return OutcomeAuthFailure
			}
		}
		inner := md5.Sum([]byte(u.Password))
		expected := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%x.%s", inner, req.Salt))))
		if req.Password != expected {
			return OutcomeAuthFailure
		}
	} else {
		if req.Time != "" || req.Sign != "" {
			if req.Time == "" {
				return OutcomeAuthFailure
			}
			expected := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%s:%s", req.Username, req.Time, u.Password))))
			if req.Sign != "" && req.Sign != expected {
				return OutcomeAuthFailure
			}
			if req.Password != expected {
				return OutcomeAuthFailure
			}
		} else if !verifyPassword(u.Password, req.Password) {
			return OutcomeAuthFailure
		}
	}

	p, err := provider.GetProvider(u)
	if err != nil {
		log.Printf("Provider error for user %q: %v", req.Username, err)
		return OutcomeSystemError
	}

	if err := p.UpdateRecord(req.Domain, req.IP); err != nil {
		log.Printf("UpdateRecord error for domain %q and ip %q: %v", req.Domain, req.IP, err)
		return OutcomeSystemError
	}

	log.Printf("Successfully updated %s to %s", req.Domain, req.IP)
	return OutcomeSuccess
}

func (m *GnuHTTPMode) Respond(w http.ResponseWriter, req *Request, outcome Outcome) {
	// If no password provided, issue challenge page.
	if req != nil && req.Password == "" {
		now := time.Now().Unix()
		user := req.Username
		u := config.GetUser(user)
		if u == nil {
			if _, err := w.Write([]byte("1")); err != nil {
				log.Printf("HTTP Write Error: %v", err)
			}
			return
		}
		salt := generateSalt(10)
		if salt == "" {
			salt = fallbackSalt(10)
		}
		sign := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s:%d:%s", user, now, u.Password))))
		body := fmt.Sprintf(`<html><head>
<meta name="salt" content="%s">
<meta name="time" content="%d">
<meta name="sign" content="%s">
<meta name="addr" content="%s">
</head><body></body></html>`, html.EscapeString(salt), now, html.EscapeString(sign), html.EscapeString(req.IP))
		if _, err := w.Write([]byte(body)); err != nil {
			log.Printf("HTTP Write Error: %v", err)
		}
		return
	}

	// Standard response mapping similar to DynDNS numeric
	var body string
	switch outcome {
	case OutcomeSuccess:
		body = "0"
	case OutcomeAuthFailure:
		body = "1"
	case OutcomeInvalidDomain:
		body = "1"
	default:
		body = "1"
	}

	if _, err := w.Write([]byte(body)); err != nil {
		log.Printf("HTTP Write Error: %v", err)
	}
}

func generateSalt(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	if length <= 0 {
		return ""
	}
	buf := make([]byte, length)
	max := big.NewInt(int64(len(charset)))
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			log.Printf("crypto/rand failed generating salt: %v", err)
			return ""
		}
		buf[i] = charset[n.Int64()]
	}
	return string(buf)
}
