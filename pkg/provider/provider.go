package provider

import (
	"errors"

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
