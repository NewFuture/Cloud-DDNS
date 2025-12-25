package server

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync/atomic"

	"github.com/NewFuture/CloudDDNS/pkg/server/mode"
)

var debugEnabled atomic.Bool

// SetDebug enables or disables debug logging.
func SetDebug(enabled bool) {
	debugEnabled.Store(enabled)
	mode.SetDebugMode(enabled)
}

func debugLogf(format string, args ...interface{}) {
	if !debugEnabled.Load() {
		return
	}
	log.Printf("[DEBUG] "+format, args...)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (lrw *loggingResponseWriter) WriteHeader(statusCode int) {
	lrw.status = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

func (lrw *loggingResponseWriter) Write(b []byte) (int, error) {
	if lrw.status == 0 {
		lrw.status = http.StatusOK
	}
	_, _ = lrw.body.Write(b)
	return lrw.ResponseWriter.Write(b)
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
		go mode.NewGnuTCPMode(debugLogf).Handle(conn)
	}
}

// handleDDNSUpdate handles DDNS update requests (compatible with optical modem/router clients).
func handleDDNSUpdate(w http.ResponseWriter, r *http.Request) {
	handleDDNSUpdateWithMode(w, r, false)
}

func handleDDNSUpdateWithMode(w http.ResponseWriter, r *http.Request, numericResponse bool) {
	const maxLoggedBody = 4096 // cap logged body to 4KB to prevent excessive memory usage
	limitedBody := io.LimitReader(r.Body, maxLoggedBody)
	bodyBytes, err := io.ReadAll(limitedBody)
	if err != nil {
		debugLogf("HTTP request body read error after %d bytes (logging partial body): %v", len(bodyBytes), err)
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	debugLogf("HTTP request rawURL=%q auth=%q body=%q", r.URL.String(), r.Header.Get("Authorization"), string(bodyBytes))

	lrw := &loggingResponseWriter{ResponseWriter: w, status: http.StatusOK}
	var m mode.Mode
	if numericResponse {
		m = mode.NewGnuHTTPMode(debugLogf)
	} else {
		switch r.URL.Path {
		case "/dyn/generic.php", "/dyn/tomato.php":
			m = mode.NewEasyDNSMode(debugLogf)
		case "/api/autodns.cfm":
			m = mode.NewDtDNSMode(debugLogf)
		default:
			m = mode.NewDynMode(false, debugLogf)
		}
	}
	debugLogf("Selected mode for path %s: %T", r.URL.Path, m)
	req, outcome := m.Prepare(r)
	if outcome == mode.OutcomeSuccess {
		outcome = m.Process(req)
	}
	m.Respond(lrw, req, outcome)
	debugLogf("HTTP response status=%d body=%q", lrw.status, lrw.body.String())
}

func handleCGIUpdate(w http.ResponseWriter, r *http.Request) {
	handleDDNSUpdateWithMode(w, r, true)
}

// StartHTTP starts the HTTP listener.
func StartHTTP(port int) {
	// Support multiple paths (compatible with various firmware clients).
	http.HandleFunc("/nic/update", handleDDNSUpdate)
	http.HandleFunc("/update", handleDDNSUpdate)
	http.HandleFunc("/dyndns/update", handleDDNSUpdate)   // 3322/qDNS
	http.HandleFunc("/ph/update", handleDDNSUpdate)       // Oray
	http.HandleFunc("/dyn/generic.php", handleDDNSUpdate) // easyDNS
	http.HandleFunc("/dyn/tomato.php", handleDDNSUpdate)  // easyDNS
	http.HandleFunc("/api/autodns.cfm", handleDDNSUpdate) // DtDNS
	http.HandleFunc("/cgi-bin/gdipupdt.cgi", handleCGIUpdate)
	http.HandleFunc("/", handleDDNSUpdate)

	log.Printf("HTTP Server listening on :%d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("HTTP Server Error: %v", err)
	}
}
