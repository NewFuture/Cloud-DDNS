package mode

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/provider"
)

// OrayTCPMode handles Oray (PeanutHull/花生壳) TCP protocol.
// Protocol format: username:password:hostname[:ip]
// IP parameter is optional - if not provided, client's remote address is used automatically
// Response: good <ip> / badauth / notfqdn / 911
type OrayTCPMode struct {
	debugLogf func(format string, args ...interface{})
}

func NewOrayTCPMode(debug func(format string, args ...interface{})) *OrayTCPMode {
	return &OrayTCPMode{debugLogf: debug}
}

func (m *OrayTCPMode) Handle(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// Read the request line
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Printf("Oray TCP Read Error: %v", err)
		return
	}
	m.debugLogf("Oray TCP received raw request: %q", line)

	// Parse request: username:password:hostname[:ip]
	// Need to handle IPv6 addresses which contain colons
	// Format: username:password:hostname or username:password:hostname:ipv4 or username:password:hostname:ipv6
	line = strings.TrimSpace(line)

	// Find first three colons (username, password, hostname)
	parts := make([]string, 0, 4)
	lastIdx := 0
	colonCount := 0
	for i, c := range line {
		if c == ':' {
			colonCount++
			parts = append(parts, line[lastIdx:i])
			lastIdx = i + 1
			if colonCount == 3 {
				// Everything after the third colon is the IP (may contain colons for IPv6)
				if lastIdx < len(line) {
					parts = append(parts, line[lastIdx:])
				}
				break
			}
		}
	}

	// If we haven't found 3 colons, add the remaining part
	if colonCount < 3 && lastIdx < len(line) {
		parts = append(parts, line[lastIdx:])
	}

	if len(parts) < 3 {
		m.debugLogf("Oray TCP invalid request parts length: %d", len(parts))
		m.writeResponse(conn, "911")
		return
	}

	username := parts[0]
	password := parts[1]
	hostname := parts[2]

	var providedIP string
	if len(parts) > 3 {
		providedIP = parts[3]
	}

	// Validate hostname
	if hostname == "" || len(hostname) < 3 || len(hostname) > 253 {
		log.Printf("Oray TCP invalid hostname: %q", hostname)
		m.writeResponse(conn, "notfqdn")
		return
	}

	// Resolve IP address
	targetIP := providedIP
	if targetIP == "" || targetIP == "0.0.0.0" {
		// Use client's remote address when IP not provided
		remoteIP, err := extractRemoteIP(conn.RemoteAddr().String())
		if err != nil {
			log.Printf("Oray TCP failed to extract remote IP: %v", err)
			m.writeResponse(conn, "911")
			return
		}
		targetIP = remoteIP
	}
	m.debugLogf("Oray TCP request parsed user=%s hostname=%s targetIP=%s", username, hostname, targetIP)

	// Validate IP address
	if net.ParseIP(targetIP) == nil {
		log.Printf("Oray TCP invalid IP address: %q", targetIP)
		m.writeResponse(conn, "911")
		return
	}

	// Debug mode bypass
	if isDebugMode() && username == "debug" && password == "debug" {
		m.debugLogf("Oray TCP debug mode bypass success for hostname=%s ip=%s", hostname, targetIP)
		m.writeResponse(conn, fmt.Sprintf("good %s", targetIP))
		return
	}

	// Authenticate user
	u := config.GetUser(username)
	if u == nil {
		m.debugLogf("Oray TCP user %q not found", username)
		m.writeResponse(conn, "badauth")
		return
	}

	// Verify password (supports plain text, MD5, SHA256, Base64)
	if !verifyPassword(u.Password, password) {
		m.debugLogf("Oray TCP authentication failed for user=%s", username)
		m.writeResponse(conn, "badauth")
		return
	}
	m.debugLogf("Oray TCP authentication succeeded for user=%s", username)

	// Extend deadline for provider update
	conn.SetDeadline(time.Now().Add(60 * time.Second))

	// Get provider and update DNS record
	p, err := provider.GetProvider(u)
	if err != nil {
		log.Printf("Oray TCP provider error: %v", err)
		m.debugLogf("Oray TCP provider init failed for user=%s provider=%s error=%v", username, u.Provider, err)
		m.writeResponse(conn, "911")
		return
	}
	m.debugLogf("Oray TCP provider initialized for user=%s provider=%s", username, u.Provider)

	err = p.UpdateRecord(hostname, targetIP)
	if err != nil {
		log.Printf("Oray TCP update error: %v", err)
		m.debugLogf("Oray TCP DNS update failed for hostname=%s ip=%s error=%v", hostname, targetIP, err)
		m.writeResponse(conn, "911")
	} else {
		log.Printf("Oray TCP success: %s -> %s", hostname, targetIP)
		m.debugLogf("Oray TCP DNS update succeeded for hostname=%s ip=%s", hostname, targetIP)
		m.writeResponse(conn, fmt.Sprintf("good %s", targetIP))
	}
}

func (m *OrayTCPMode) writeResponse(conn net.Conn, response string) {
	if _, err := conn.Write([]byte(response + "\n")); err != nil {
		log.Printf("Oray TCP write error: %v", err)
	}
}
