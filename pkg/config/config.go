package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Users  []UserConfig `yaml:"users"`
}

type ServerConfig struct {
	TCPPort  int `yaml:"tcp_port"`
	HTTPPort int `yaml:"http_port"`
}

type UserConfig struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"` // 用作 API SecretKey
	Provider string `yaml:"provider"`
}

var GlobalConfig Config

func LoadConfig(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &GlobalConfig)
}

// GetUser 根据用户名查找配置
func GetUser(username string) *UserConfig {
	for _, u := range GlobalConfig.Users {
		if u.Username == username {
			return &u
		}
	}
	return nil
}
