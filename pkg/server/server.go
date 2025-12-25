package server

import (
	"fmt"
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
		go mode.NewGnuTCPMode(debugLogf).Handle(conn)
	}
}

// handleDDNSUpdate handles DDNS update requests (compatible with optical modem/router clients).
func handleDDNSUpdate(w http.ResponseWriter, r *http.Request) {
	handleDDNSUpdateWithMode(w, r, false)
}

func handleDDNSUpdateWithMode(w http.ResponseWriter, r *http.Request, numericResponse bool) {
	debugLogf("HTTP request %s %s rawQuery=%q", r.Method, r.URL.Path, r.URL.RawQuery)
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
	req, outcome := m.Prepare(r)
	if outcome == mode.OutcomeSuccess {
		outcome = m.Process(req)
	}
	m.Respond(w, req, outcome)
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
