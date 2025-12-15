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
	conn.Write([]byte(salt + "\n"))

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
		conn.Write([]byte("1\n"))
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
			conn.Write([]byte("1\n"))
			return
		}
		targetIP = host
	}

	// Validate IP address
	if net.ParseIP(targetIP) == nil {
		log.Printf("Invalid IP address: %q", targetIP)
		conn.Write([]byte("1\n"))
		return
	}

	// 3. 鉴权 (Protocol Step: Verify)
	u := config.GetUser(user)
	if u == nil {
		conn.Write([]byte("1\n")) // 用户未找到
		return
	}

	// 计算预期 Hash: MD5(User + ":" + Salt + ":" + SecretKey)
	expectedStr := fmt.Sprintf("%s:%s:%s", user, salt, u.Password)
	expectedHash := fmt.Sprintf("%x", md5.Sum([]byte(expectedStr)))

	if clientHash != expectedHash {
		conn.Write([]byte("1\n")) // 鉴权失败
		return
	}

	// 4. 调用 Provider
	p, err := provider.GetProvider(u)
	if err != nil {
		log.Printf("Provider Error: %v", err)
		conn.Write([]byte("1\n"))
		return
	}

	err = p.UpdateRecord(domain, targetIP)
	if err != nil {
		log.Printf("Update Error: %v", err)
		conn.Write([]byte("1\n"))
	} else {
		log.Printf("Success: %s -> %s", domain, targetIP)
		conn.Write([]byte("0\n"))
	}
}

// StartHTTP 启动 HTTP 监听
func StartHTTP(port int) {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		user := q.Get("user")
		pass := q.Get("pass") // HTTP 模式下直接获取 Secret
		domain := q.Get("domn")
		if domain == "" {
			domain = q.Get("domain")
		}

		ip := q.Get("addr")
		if ip == "" {
			var err error
			ip, _, err = net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				log.Printf("Invalid RemoteAddr format: %q, error: %v", r.RemoteAddr, err)
				http.Error(w, "Invalid remote address", 400)
				return
			}
		}

		// Validate IP address
		if net.ParseIP(ip) == nil {
			log.Printf("Invalid IP address: %q", ip)
			http.Error(w, "Invalid IP address", 400)
			return
		}

		// Validate domain
		if domain == "" || len(domain) < 3 || len(domain) > 253 {
			http.Error(w, "Invalid domain", 400)
			return
		}

		u := config.GetUser(user)
		if u == nil || u.Password != pass {
			http.Error(w, "Auth failed", 401)
			return
		}

		p, err := provider.GetProvider(u)
		if err != nil {
			http.Error(w, "Provider error: "+err.Error(), 500)
			return
		}

		if err := p.UpdateRecord(domain, ip); err != nil {
			http.Error(w, err.Error(), 500)
		} else {
			w.Write([]byte("0"))
		}
	})

	log.Printf("HTTP Server listening on :%d", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatalf("HTTP Server Error: %v", err)
	}
}
