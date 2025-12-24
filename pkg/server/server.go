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
	reqc := "0"
	if len(parts) > 3 && parts[3] != "" {
		reqc = parts[3]
	}

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
	switch reqc {
	case "1":
		targetIP = "0.0.0.0"
	case "2":
		targetIP = ""
	}

	if reqc == "1" {
		targetIP = "0.0.0.0"
	} else {
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

	// Reset deadline before potentially slow DNS API call
	// The initial 30s deadline is for authentication, extend it for DNS update
	conn.SetDeadline(time.Now().Add(60 * time.Second))

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
		responseCode := "0\n"
		if reqc == "1" {
			responseCode = "2\n"
		}
		if _, writeErr := conn.Write([]byte(responseCode)); writeErr != nil {
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

// verifyPassword 验证密码，支持多种格式：明文、MD5、SHA256、Base64
// 尝试以下验证顺序：
// 1. 明文匹配
// 2. MD5(password) 匹配
// 3. SHA256(password) 匹配
// 4. Base64(password) 解码后匹配
func verifyPassword(storedPassword, inputPassword string) bool {
	// 1. 明文匹配
	if storedPassword == inputPassword {
		return true
	}

	// 2. 尝试 MD5 匹配 - inputPassword 是 MD5(storedPassword)
	md5Hash := md5.Sum([]byte(storedPassword))
	md5Str := hex.EncodeToString(md5Hash[:])
	if strings.EqualFold(md5Str, inputPassword) {
		return true
	}

	// 3. 尝试 SHA256 匹配 - inputPassword 是 SHA256(storedPassword)
	sha256Hash := sha256.Sum256([]byte(storedPassword))
	sha256Str := hex.EncodeToString(sha256Hash[:])
	if strings.EqualFold(sha256Str, inputPassword) {
		return true
	}

	// 4. 尝试 Base64 解码匹配 - inputPassword 是 Base64(storedPassword)
	if decoded, err := base64.StdEncoding.DecodeString(inputPassword); err == nil {
		if string(decoded) == storedPassword {
			return true
		}
	}

	// 5. 反向检查：storedPassword 可能是编码的
	// 尝试 storedPassword 是 MD5 且 inputPassword 是明文
	inputMd5 := md5.Sum([]byte(inputPassword))
	inputMd5Str := hex.EncodeToString(inputMd5[:])
	if strings.EqualFold(storedPassword, inputMd5Str) {
		return true
	}

	// 尝试 storedPassword 是 SHA256 且 inputPassword 是明文
	inputSha256 := sha256.Sum256([]byte(inputPassword))
	inputSha256Str := hex.EncodeToString(inputSha256[:])
	if strings.EqualFold(storedPassword, inputSha256Str) {
		return true
	}

	// 尝试 storedPassword 是 Base64
	if decoded, err := base64.StdEncoding.DecodeString(storedPassword); err == nil {
		if string(decoded) == inputPassword {
			return true
		}
	}

	return false
}

// handleDDNSUpdate handles DDNS update requests (compatible with modem/router firmwares)
func handleDDNSUpdate(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	// Support multiple parameter aliases (case-insensitive across modem firmware)
	user := getQueryParam(q, "user", "username", "usr", "name")
	pass := getQueryParam(q, "pass", "password", "pwd")
	domain := getQueryParam(q, "domn", "domain", "hostname", "host")
	ip := getQueryParam(q, "addr", "myip", "ip")
	reqc := strings.TrimSpace(getQueryParam(q, "reqc"))

	// Use numeric GnuDIP-compatible responses for standard CGI path or explicit reqc
	gnudipCompat := strings.Contains(r.URL.Path, "gdipupdt.cgi") || reqc != ""

	if reqc == "" {
		reqc = "0"
	}

	// Choose IP based on reqc mode; default to client IP when unspecified
	switch reqc {
	case "1":
		// offline/delete request
		ip = "0.0.0.0"
	case "2":
		// force auto-detect
		ip = ""
	}

	if ip == "" {
		var err error
		ip, _, err = net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Printf("Invalid RemoteAddr format: %q, error: %v", r.RemoteAddr, err)
			if gnudipCompat {
				_, _ = w.Write([]byte("1"))
			} else if _, writeErr := w.Write([]byte("911")); writeErr != nil {
				log.Printf("HTTP Write Error: %v", writeErr)
			}
			return
		}
	}

	// Validate IP address
	if net.ParseIP(ip) == nil {
		log.Printf("Invalid IP address: %q", ip)
		if gnudipCompat {
			_, _ = w.Write([]byte("1"))
		} else if _, err := w.Write([]byte("911")); err != nil {
			log.Printf("HTTP Write Error: %v", err)
		}
		return
	}

	// Validate domain
	if domain == "" || len(domain) < 3 || len(domain) > 253 {
		log.Printf("Invalid domain: %q", domain)
		if gnudipCompat {
			_, _ = w.Write([]byte("1"))
		} else if _, err := w.Write([]byte("notfqdn")); err != nil {
			log.Printf("HTTP Write Error: %v", err)
		}
		return
	}

	// 认证 - 支持多种密码格式
	u := config.GetUser(user)
	if u == nil || !verifyPassword(u.Password, pass) {
		log.Printf("Authentication failed for user: %q", user)
		if gnudipCompat {
			_, _ = w.Write([]byte("1"))
		} else if _, err := w.Write([]byte("badauth")); err != nil {
			log.Printf("HTTP Write Error: %v", err)
		}
		return
	}

	// 获取 Provider
	p, err := provider.GetProvider(u)
	if err != nil {
		log.Printf("Provider error for user %q: %v", user, err)
		if gnudipCompat {
			_, _ = w.Write([]byte("1"))
		} else if _, writeErr := w.Write([]byte("911")); writeErr != nil {
			log.Printf("HTTP Write Error: %v", writeErr)
		}
		return
	}

	// Update DNS record
	if err := p.UpdateRecord(domain, ip); err != nil {
		log.Printf("UpdateRecord error for domain %q and ip %q: %v", domain, ip, err)
		if gnudipCompat {
			if _, writeErr := w.Write([]byte("1")); writeErr != nil {
				log.Printf("HTTP Write Error: %v", writeErr)
			}
		} else if _, writeErr := w.Write([]byte("911")); writeErr != nil {
			log.Printf("HTTP Write Error: %v", writeErr)
		}
	} else {
		log.Printf("Successfully updated %s to %s", domain, ip)
		if gnudipCompat {
			if reqc == "1" {
				if _, writeErr := w.Write([]byte("2")); writeErr != nil {
					log.Printf("HTTP Write Error: %v", writeErr)
				}
			} else {
				if _, writeErr := w.Write([]byte("0")); writeErr != nil {
					log.Printf("HTTP Write Error: %v", writeErr)
				}
			}
		} else if _, writeErr := w.Write([]byte("good " + ip)); writeErr != nil {
			log.Printf("HTTP Write Error: %v", writeErr)
		}
	}
}

// StartHTTP starts the HTTP listener
func StartHTTP(port int) {
	// Support multiple paths for compatibility with modem/router firmware
	http.HandleFunc("/nic/update", handleDDNSUpdate)
	http.HandleFunc("/update", handleDDNSUpdate)
	http.HandleFunc("/cgi-bin/gdipupdt.cgi", handleDDNSUpdate)
	http.HandleFunc("/", handleDDNSUpdate)

	log.Printf("HTTP Server listening on :%d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("HTTP Server Error: %v", err)
	}
}
