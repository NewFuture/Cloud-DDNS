package mode

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"log"
	"net"
	"strings"
	"time"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/provider"
)

// GnuTCPMode handles GnuDIP TCP protocol.
type GnuTCPMode struct {
	debugLogf func(format string, args ...interface{})
}

func NewGnuTCPMode(debug func(format string, args ...interface{})) *GnuTCPMode {
	return &GnuTCPMode{debugLogf: debug}
}

func (m *GnuTCPMode) Handle(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	salt := fmt.Sprintf("%d.%d", time.Now().Unix(), time.Now().UnixNano())
	m.debugLogf("Generated salt %s for %s", salt, conn.RemoteAddr().String())
	if _, err := conn.Write([]byte(salt + "\n")); err != nil {
		log.Printf("TCP Write Error (salt): %v", err)
		return
	}

	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Printf("TCP Read Error: %v", err)
		return
	}
	m.debugLogf("Received raw TCP request: %q", line)
	parts := strings.Split(strings.TrimSpace(line), ":")

	if len(parts) < 3 {
		m.debugLogf("Invalid TCP request parts length: %d", len(parts))
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
	m.debugLogf("TCP request parsed user=%s domain=%s targetIP=%s reqc=%d", user, domain, targetIP, reqc)

	if net.ParseIP(targetIP) == nil {
		log.Printf("Invalid IP address: %q", targetIP)
		if _, err := conn.Write([]byte("1\n")); err != nil {
			log.Printf("TCP Write Error (invalid IP): %v", err)
		}
		return
	}

	u := config.GetUser(user)
	if u == nil {
		m.debugLogf("User %q not found", user)
		if _, err := conn.Write([]byte("1\n")); err != nil {
			log.Printf("TCP Write Error (user not found): %v", err)
		}
		return
	}

	expectedStr := fmt.Sprintf("%s:%s:%s", user, salt, u.Password)
	expectedHash := fmt.Sprintf("%x", md5.Sum([]byte(expectedStr)))

	if clientHash != expectedHash {
		m.debugLogf("Authentication failed for user=%s expectedHash=%s clientHash=%s", user, expectedHash, clientHash)
		if _, err := conn.Write([]byte("1\n")); err != nil {
			log.Printf("TCP Write Error (auth failed): %v", err)
		}
		return
	}
	m.debugLogf("Authentication succeeded for user=%s", user)

	conn.SetDeadline(time.Now().Add(60 * time.Second))

	p, err := provider.GetProvider(u)
	if err != nil {
		log.Printf("Provider Error: %v", err)
		if _, writeErr := conn.Write([]byte("1\n")); writeErr != nil {
			log.Printf("TCP Write Error (provider error): %v", writeErr)
		}
		return
	}
	m.debugLogf("Provider initialized for user=%s provider=%s", user, u.Provider)

	err = p.UpdateRecord(domain, targetIP)
	if err != nil {
		log.Printf("Update Error: %v", err)
		m.debugLogf("DNS update failed for domain=%s ip=%s error=%v", domain, targetIP, err)
		if _, writeErr := conn.Write([]byte("1\n")); writeErr != nil {
			log.Printf("TCP Write Error (update failed): %v", writeErr)
		}
	} else {
		log.Printf("Success: %s -> %s", domain, targetIP)
		m.debugLogf("DNS update succeeded for domain=%s ip=%s", domain, targetIP)
		if _, writeErr := conn.Write([]byte("0\n")); writeErr != nil {
			log.Printf("TCP Write Error (success response): %v", writeErr)
		}
	}
}
