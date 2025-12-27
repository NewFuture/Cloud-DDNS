package provider

import (
	"errors"
	"strings"

	"github.com/NewFuture/CloudDDNS/pkg/config"
)

// Provider 统一接口
type Provider interface {
	UpdateRecord(domain string, ip string) error
}

// GetProvider 工厂方法
func GetProvider(u *config.UserConfig) (Provider, error) {
	switch u.Provider {
	case "aliyun":
		return NewAliyunProvider(u.Username, u.Password), nil
	case "tencent":
		return NewTencentProvider(u.Username, u.Password), nil
	// 扩展其他厂商...
	default:
		return nil, errors.New("unknown provider: " + u.Provider)
	}
}

// ParseDomain 拆分完整域名为基础域名和子域名
// 处理常见的 TLD 情况，包括二级 TLD (如 .co.uk, .com.cn)
func ParseDomain(fullDomain string) (baseDomain, subDomain string, err error) {
	parts := strings.Split(fullDomain, ".")
	if len(parts) < 2 {
		return "", "", errors.New("invalid domain format")
	}

	// 检查常见的二级 TLD
	twoLevelTLDs := map[string]bool{
		"co.uk": true, "com.cn": true, "net.cn": true, "org.cn": true, "gov.cn": true,
		"co.jp": true, "co.nz": true, "co.za": true, "com.au": true, "com.br": true,
		"com.hk": true, "com.tw": true, "org.uk": true, "ac.uk": true, "gov.uk": true,
	}

	// 如果有 3 个或更多部分，检查最后两部分是否是二级 TLD
	if len(parts) >= 3 {
		lastTwo := strings.Join(parts[len(parts)-2:], ".")
		if twoLevelTLDs[lastTwo] {
			// 对于二级 TLD，基础域名是最后 3 部分
			if len(parts) == 3 {
				// example.co.uk
				baseDomain = fullDomain
				subDomain = "@"
			} else {
				// subdomain.example.co.uk
				baseDomain = strings.Join(parts[len(parts)-3:], ".")
				subDomain = strings.Join(parts[:len(parts)-3], ".")
			}
			return baseDomain, subDomain, nil
		}
	}

	// 默认情况：最后两部分是基础域名 (example.com)
	if len(parts) == 2 {
		baseDomain = fullDomain
		subDomain = "@"
	} else {
		baseDomain = strings.Join(parts[len(parts)-2:], ".")
		subDomain = strings.Join(parts[:len(parts)-2], ".")
	}

	if subDomain == "" {
		subDomain = "@"
	}

	return baseDomain, subDomain, nil
}

// IsSupportedProvider returns true if a provider name is recognized.
func IsSupportedProvider(name string) bool {
	switch strings.ToLower(name) {
	case "aliyun", "tencent":
		return true
	default:
		return false
	}
}
