package server

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/provider"
)

var debugEnabled atomic.Bool

// SetDebug enables or disables debug logging.
func SetDebug(enabled bool) {
	debugEnabled.Store(enabled)
}

func debugLogf(format string, args ...interface{}) {
	if !debugEnabled.Load() {
		return
	}
	log.Printf("[DEBUG] "+format, args...)
}

// StartTCP starts the GnuDIP TCP listener.
func StartTCP(port int) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("TCP Listen Error: %v", err)
	}
	log.Printf("GnuDIP TCP Server listening on :%d", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("TCP Accept Error: %v", err)
			continue
		}
		debugLogf("Accepted TCP connection from %s", conn.RemoteAddr().String())
		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// 1. Send Salt (Protocol Step: Challenge)
	salt := fmt.Sprintf("%d.%d", time.Now().Unix(), time.Now().UnixNano())
	debugLogf("Generated salt %s for %s", salt, conn.RemoteAddr().String())
	if _, err := conn.Write([]byte(salt + "\n")); err != nil {
		log.Printf("TCP Write Error (salt): %v", err)
		return
	}

	// 2. Read client response (Protocol Step: Response)
	// Format: User:Hash:Domain:ReqC:IP
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Printf("TCP Read Error: %v", err)
		return
	}
	debugLogf("Received raw TCP request: %q", line)
	parts := strings.Split(strings.TrimSpace(line), ":")

	if len(parts) < 3 {
		debugLogf("Invalid TCP request parts length: %d", len(parts))
		return
	}
	user := parts[0]
	clientHash := parts[1]
	domain := parts[2]
	reqcRaw := ""
	if len(parts) > 3 {
		reqcRaw = parts[3]
	}
	reqc, err := parseReqc(reqcRaw)
	if err != nil {
		log.Printf("Invalid reqc value %q: %v", reqcRaw, err)
		if _, writeErr := conn.Write([]byte("1\n")); writeErr != nil {
			log.Printf("TCP Write Error (invalid reqc): %v", writeErr)
		}
		return
	}

	// Validate domain
	if domain == "" || len(domain) < 3 || len(domain) > 253 {
		log.Printf("Invalid domain: %q", domain)
		if _, err := conn.Write([]byte("1\n")); err != nil {
			log.Printf("TCP Write Error (invalid domain): %v", err)
		}
		return
	}

	providedIP := ""
	if len(parts) > 4 {
		providedIP = parts[4]
	}

	targetIP, err := resolveRequestIP(reqc, providedIP, conn.RemoteAddr().String())
	if err != nil {
		log.Printf("Failed to resolve target IP: %v", err)
		if _, writeErr := conn.Write([]byte("1\n")); writeErr != nil {
			log.Printf("TCP Write Error (resolve ip): %v", writeErr)
		}
		return
	}
	debugLogf("TCP request parsed user=%s domain=%s targetIP=%s reqc=%d", user, domain, targetIP, reqc)

	// Validate IP address
	if net.ParseIP(targetIP) == nil {
		log.Printf("Invalid IP address: %q", targetIP)
		if _, err := conn.Write([]byte("1\n")); err != nil {
			log.Printf("TCP Write Error (invalid IP): %v", err)
		}
		return
	}

	// 3. Authentication (Protocol Step: Verify)
	u := config.GetUser(user)
	if u == nil {
		debugLogf("User %q not found", user)
		if _, err := conn.Write([]byte("1\n")); err != nil {
			log.Printf("TCP Write Error (user not found): %v", err)
		}
		return
	}

	// Calculate expected hash: MD5(User + ":" + Salt + ":" + SecretKey)
	/*
	 * SECURITY WARNING:
	 * The following code uses MD5 for password hashing as required by the legacy GnuDIP protocol specification.
	 * MD5 is cryptographically broken and should not be used for security-sensitive operations.
	 * DO NOT use this code without deploying it behind TLS/HTTPS to protect credentials in transit.
	 * See the project README for more information about this security limitation and recommended mitigations.
	 */
	expectedStr := fmt.Sprintf("%s:%s:%s", user, salt, u.Password)
	expectedHash := fmt.Sprintf("%x", md5.Sum([]byte(expectedStr)))

	if clientHash != expectedHash {
		debugLogf("Authentication failed for user=%s expectedHash=%s clientHash=%s", user, expectedHash, clientHash)
		if _, err := conn.Write([]byte("1\n")); err != nil {
			log.Printf("TCP Write Error (auth failed): %v", err)
		}
		return
	}
	debugLogf("Authentication succeeded for user=%s", user)

	// Reset deadline before potentially slow DNS API call
	// The initial 30s deadline is for authentication, extend it for DNS update
	conn.SetDeadline(time.Now().Add(60 * time.Second))

	// 4. Call provider
	p, err := provider.GetProvider(u)
	if err != nil {
		log.Printf("Provider Error: %v", err)
		if _, writeErr := conn.Write([]byte("1\n")); writeErr != nil {
			log.Printf("TCP Write Error (provider error): %v", writeErr)
		}
		return
	}
	debugLogf("Provider initialized for user=%s provider=%s", user, u.Provider)

	err = p.UpdateRecord(domain, targetIP)
	if err != nil {
		log.Printf("Update Error: %v", err)
		debugLogf("DNS update failed for domain=%s ip=%s error=%v", domain, targetIP, err)
		if _, writeErr := conn.Write([]byte("1\n")); writeErr != nil {
			log.Printf("TCP Write Error (update failed): %v", writeErr)
		}
	} else {
		log.Printf("Success: %s -> %s", domain, targetIP)
		debugLogf("DNS update succeeded for domain=%s ip=%s", domain, targetIP)
		if _, writeErr := conn.Write([]byte("0\n")); writeErr != nil {
			log.Printf("TCP Write Error (success response): %v", writeErr)
		}
	}
}

// getQueryParam gets a value from the query parameters, supporting multiple aliases (case-insensitive).
func getQueryParam(q map[string][]string, names ...string) string {
	for _, name := range names {
		// Try exact match first
		if val := q[name]; len(val) > 0 && val[0] != "" {
			return val[0]
		}
		// Try case-insensitive match
		for key, val := range q {
			if strings.EqualFold(key, name) && len(val) > 0 && val[0] != "" {
				return val[0]
			}
		}
	}
	return ""
}

// verifyPassword validates a password with multiple formats: plaintext, MD5, SHA256, Base64.
// Verification order:
// 1. Plaintext match
// 2. MD5(password) match
// 3. SHA256(password) match
// 4. Base64(password) decoded match
func verifyPassword(storedPassword, inputPassword string) bool {
	// 1. Plaintext match
	if storedPassword == inputPassword {
		return true
	}

	// 2. Try MD5 match - inputPassword is MD5(storedPassword)
	md5Hash := md5.Sum([]byte(storedPassword))
	md5Str := hex.EncodeToString(md5Hash[:])
	if strings.EqualFold(md5Str, inputPassword) {
		return true
	}

	// 3. Try SHA256 match - inputPassword is SHA256(storedPassword)
	sha256Hash := sha256.Sum256([]byte(storedPassword))
	sha256Str := hex.EncodeToString(sha256Hash[:])
	if strings.EqualFold(sha256Str, inputPassword) {
		return true
	}

	// 4. Try Base64 decode match - inputPassword is Base64(storedPassword)
	if decoded, err := base64.StdEncoding.DecodeString(inputPassword); err == nil {
		if string(decoded) == storedPassword {
			return true
		}
	}

	// 5. Reverse checks: storedPassword may already be encoded
	// Try storedPassword as MD5 with plaintext input
	inputMd5 := md5.Sum([]byte(inputPassword))
	inputMd5Str := hex.EncodeToString(inputMd5[:])
	if strings.EqualFold(storedPassword, inputMd5Str) {
		return true
	}

	// Try storedPassword as SHA256 with plaintext input
	inputSha256 := sha256.Sum256([]byte(inputPassword))
	inputSha256Str := hex.EncodeToString(inputSha256[:])
	if strings.EqualFold(storedPassword, inputSha256Str) {
		return true
	}

	// Try storedPassword as Base64
	if decoded, err := base64.StdEncoding.DecodeString(storedPassword); err == nil {
		if string(decoded) == inputPassword {
			return true
		}
	}

	return false
}

type responseOutcome int

const (
	responseSuccess responseOutcome = iota
	responseAuthFailure
	responseInvalidDomain
	responseSystemError
)

func parseReqc(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return 0, err
	}
	if value < 0 || value > 2 {
		return 0, fmt.Errorf("unsupported reqc value %d", value)
	}
	return value, nil
}

func resolveRequestIP(reqc int, providedIP string, remoteAddr string) (string, error) {
	switch reqc {
	case 1: // offline
		return "0.0.0.0", nil
	case 2: // auto-detect
		return extractRemoteIP(remoteAddr)
	case 0: // update
		if providedIP != "" && providedIP != "0.0.0.0" {
			return providedIP, nil
		}
		return extractRemoteIP(remoteAddr)
	default:
		return "", fmt.Errorf("unsupported reqc value %d (unexpected after validation)", reqc)
	}
}

func extractRemoteIP(remoteAddr string) (string, error) {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return "", fmt.Errorf("invalid remote address %q: %w", remoteAddr, err)
	}
	return host, nil
}

func writeHTTPResponse(w http.ResponseWriter, body string) {
	if _, err := w.Write([]byte(body)); err != nil {
		log.Printf("HTTP Write Error: %v", err)
	}
}

func sendHTTPResponse(w http.ResponseWriter, numeric bool, reqc int, outcome responseOutcome, ip string) {
	var body string
	switch outcome {
	case responseSuccess:
		if numeric {
			if reqc == 1 {
				body = "2"
			} else {
				body = "0"
			}
		} else {
			body = "good " + ip
		}
	case responseAuthFailure:
		if numeric {
			body = "1"
		} else {
			body = "badauth"
		}
	case responseInvalidDomain:
		if numeric {
			body = "1"
		} else {
			body = "notfqdn"
		}
	default:
		if numeric {
			body = "1"
		} else {
			body = "911"
		}
	}
	writeHTTPResponse(w, body)
}

// handleDDNSUpdate handles DDNS update requests (compatible with optical modem/router clients).
func handleDDNSUpdate(w http.ResponseWriter, r *http.Request) {
	handleDDNSUpdateWithMode(w, r, false)
}

func handleDDNSUpdateWithMode(w http.ResponseWriter, r *http.Request, numericResponse bool) {
	q := r.URL.Query()
	debugLogf("HTTP request %s %s rawQuery=%q params=%v", r.Method, r.URL.Path, r.URL.RawQuery, q)

	// Support multiple parameter aliases (case-insensitive).
	user := getQueryParam(q, "user", "username", "usr", "name")
	pass := getQueryParam(q, "pass", "password", "pwd")
	domain := getQueryParam(q, "domn", "domain", "hostname", "host")
	ip := getQueryParam(q, "addr", "myip", "ip")
	reqcStr := getQueryParam(q, "reqc")
	reqc, err := parseReqc(reqcStr)
	if err != nil {
		log.Printf("Invalid reqc value %q: %v", reqcStr, err)
		sendHTTPResponse(w, numericResponse, 0, responseSystemError, "")
		return
	}

	resolvedIP, err := resolveRequestIP(reqc, ip, r.RemoteAddr)
	if err != nil {
		log.Printf("Invalid RemoteAddr format: %q, error: %v", r.RemoteAddr, err)
		sendHTTPResponse(w, numericResponse, reqc, responseSystemError, "")
		return
	}
	ip = resolvedIP
	debugLogf("HTTP parsed user=%s domain=%s ip=%s remote=%s reqc=%d", user, domain, ip, r.RemoteAddr, reqc)

	// Validate IP address
	if net.ParseIP(ip) == nil {
		log.Printf("Invalid IP address: %q", ip)
		sendHTTPResponse(w, numericResponse, reqc, responseSystemError, "")
		return
	}

	// Validate domain
	if domain == "" || len(domain) < 3 || len(domain) > 253 {
		log.Printf("Invalid domain: %q", domain)
		sendHTTPResponse(w, numericResponse, reqc, responseInvalidDomain, "")
		return
	}
	debugLogf("HTTP domain validation passed for %s", domain)

	// Authentication - supports multiple password formats.
	u := config.GetUser(user)
	if u == nil || !verifyPassword(u.Password, pass) {
		log.Printf("Authentication failed for user: %q", user)
		debugLogf("HTTP authentication failed for user=%s", user)
		sendHTTPResponse(w, numericResponse, reqc, responseAuthFailure, "")
		return
	}
	debugLogf("HTTP authentication succeeded for user=%s", user)

	// Initialize provider
	p, err := provider.GetProvider(u)
	if err != nil {
		log.Printf("Provider error for user %q: %v", user, err)
		debugLogf("HTTP provider init failed for user=%s provider=%s error=%v", user, u.Provider, err)
		sendHTTPResponse(w, numericResponse, reqc, responseSystemError, "")
		return
	}
	debugLogf("HTTP provider initialized for user=%s provider=%s", user, u.Provider)

	// Update DNS record
	if err := p.UpdateRecord(domain, ip); err != nil {
		log.Printf("UpdateRecord error for domain %q and ip %q: %v", domain, ip, err)
		debugLogf("HTTP DNS update failed for domain=%s ip=%s error=%v", domain, ip, err)
		sendHTTPResponse(w, numericResponse, reqc, responseSystemError, "")
	} else {
		log.Printf("Successfully updated %s to %s", domain, ip)
		debugLogf("HTTP DNS update succeeded for domain=%s ip=%s", domain, ip)
		sendHTTPResponse(w, numericResponse, reqc, responseSuccess, ip)
	}
}

func handleCGIUpdate(w http.ResponseWriter, r *http.Request) {
	handleDDNSUpdateWithMode(w, r, true)
}

// StartHTTP starts the HTTP listener.
func StartHTTP(port int) {
	// Support multiple paths (compatible with various firmware clients).
	http.HandleFunc("/nic/update", handleDDNSUpdate)
	http.HandleFunc("/update", handleDDNSUpdate)
	http.HandleFunc("/cgi-bin/gdipupdt.cgi", handleCGIUpdate)
	http.HandleFunc("/", handleDDNSUpdate)

	log.Printf("HTTP Server listening on :%d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("HTTP Server Error: %v", err)
	}
}
