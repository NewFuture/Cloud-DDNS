package server

import (
	"bufio"
	"crypto/md5"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/NewFuture/CloudDDNS/pkg/config"
	"github.com/NewFuture/CloudDDNS/pkg/provider"
)

// StartTCP 启动 GnuDIP TCP 监听
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
		go handleTCPConnection(conn)
	}
}

func handleTCPConnection(conn net.Conn) {
	defer conn.Close()
	conn.SetDeadline(time.Now().Add(30 * time.Second))

	// 1. 发送 Salt (Protocol Step: Challenge)
	salt := fmt.Sprintf("%d.%d", time.Now().Unix(), time.Now().UnixNano())
	if _, err := conn.Write([]byte(salt + "\n")); err != nil {
		log.Printf("TCP Write Error (salt): %v", err)
		return
	}

	// 2. 读取客户端响应 (Protocol Step: Response)
	// 格式通常为: User:Hash:Domain:ReqC:IP
	line, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		log.Printf("TCP Read Error: %v", err)
		return
	}
	parts := strings.Split(strings.TrimSpace(line), ":")

	if len(parts) < 3 {
		return
	}
	user := parts[0]
	clientHash := parts[1]
	domain := parts[2]

	// Validate domain
	if domain == "" || len(domain) < 3 || len(domain) > 253 {
		log.Printf("Invalid domain: %q", domain)
		if _, err := conn.Write([]byte("1\n")); err != nil {
			log.Printf("TCP Write Error (invalid domain): %v", err)
		}
		return
	}

	// 提取 IP，如果为空则使用 RemoteAddr
	targetIP := ""
	if len(parts) > 4 {
		targetIP = parts[4]
	}
	if targetIP == "" || targetIP == "0.0.0.0" {
		host, _, err := net.SplitHostPort(conn.RemoteAddr().String())
		if err != nil {
			log.Printf("Failed to parse remote address %q: %v", conn.RemoteAddr().String(), err)
			if _, writeErr := conn.Write([]byte("1\n")); writeErr != nil {
				log.Printf("TCP Write Error (parse remote addr): %v", writeErr)
			}
			return
		}
		targetIP = host
	}

	// Validate IP address
	if net.ParseIP(targetIP) == nil {
		log.Printf("Invalid IP address: %q", targetIP)
		if _, err := conn.Write([]byte("1\n")); err != nil {
			log.Printf("TCP Write Error (invalid IP): %v", err)
		}
		return
	}

	// 3. 鉴权 (Protocol Step: Verify)
	u := config.GetUser(user)
	if u == nil {
		if _, err := conn.Write([]byte("1\n")); err != nil {
			log.Printf("TCP Write Error (user not found): %v", err)
		}
		return
	}

	// 计算预期 Hash: MD5(User + ":" + Salt + ":" + SecretKey)
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
		if _, err := conn.Write([]byte("1\n")); err != nil {
			log.Printf("TCP Write Error (auth failed): %v", err)
		}
		return
	}

	// 4. 调用 Provider
	p, err := provider.GetProvider(u)
	if err != nil {
		log.Printf("Provider Error: %v", err)
		if _, writeErr := conn.Write([]byte("1\n")); writeErr != nil {
			log.Printf("TCP Write Error (provider error): %v", writeErr)
		}
		return
	}

	err = p.UpdateRecord(domain, targetIP)
	if err != nil {
		log.Printf("Update Error: %v", err)
		if _, writeErr := conn.Write([]byte("1\n")); writeErr != nil {
			log.Printf("TCP Write Error (update failed): %v", writeErr)
		}
	} else {
		log.Printf("Success: %s -> %s", domain, targetIP)
		if _, writeErr := conn.Write([]byte("0\n")); writeErr != nil {
			log.Printf("TCP Write Error (success response): %v", writeErr)
		}
	}
}

// getQueryParam 从查询参数中获取值，支持多个参数名别名（不区分大小写）
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

// handleDDNSUpdate 处理 DDNS 更新请求（兼容光猫/路由器）
func handleDDNSUpdate(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// 支持多种参数别名（兼容不同光猫固件）
	user := getQueryParam(q, "user", "username", "usr", "name")
	pass := getQueryParam(q, "pass", "password", "pwd")
	domain := getQueryParam(q, "domn", "domain", "hostname", "host")
	ip := getQueryParam(q, "addr", "myip", "ip")

	// 如果未指定 IP，使用客户端 IP
	if ip == "" {
		var err error
		ip, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Printf("Invalid RemoteAddr format: %q, error: %v", r.RemoteAddr, err)
			w.Write([]byte("911"))
			return
		}
	}

	// Validate IP address
	if net.ParseIP(ip) == nil {
		log.Printf("Invalid IP address: %q", ip)
		w.Write([]byte("911"))
		return
	}

	// Validate domain
	if domain == "" || len(domain) < 3 || len(domain) > 253 {
		log.Printf("Invalid domain: %q", domain)
		w.Write([]byte("notfqdn"))
		return
	}

	// 认证
	u := config.GetUser(user)
	if u == nil || u.Password != pass {
		log.Printf("Authentication failed for user: %q", user)
		w.Write([]byte("badauth"))
		return
	}

	// 获取 Provider
	p, err := provider.GetProvider(u)
	if err != nil {
		log.Printf("Provider error for user %q: %v", user, err)
		w.Write([]byte("911"))
		return
	}

	// 更新 DNS 记录
	if err := p.UpdateRecord(domain, ip); err != nil {
		log.Printf("UpdateRecord error for domain %q and ip %q: %v", domain, ip, err)
		w.Write([]byte("911"))
	} else {
		log.Printf("Successfully updated %s to %s", domain, ip)
		w.Write([]byte("good " + ip))
	}
}

// StartHTTP 启动 HTTP 监听
func StartHTTP(port int) {
	// 支持多种路径（兼容不同光猫固件）
	http.HandleFunc("/nic/update", handleDDNSUpdate)
	http.HandleFunc("/update", handleDDNSUpdate)
	http.HandleFunc("/", handleDDNSUpdate)

	log.Printf("HTTP Server listening on :%d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("HTTP Server Error: %v", err)
	}
}
