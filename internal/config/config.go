package config

import (
	"os"
)

type ServerConfig struct {
	HTTPListen      string `yaml:"http_listen" json:"http_listen"`
	SyslogTCPListen string `yaml:"syslog_tcp_listen" json:"syslog_tcp_listen"`
	SyslogUDPListen string `yaml:"syslog_udp_listen" json:"syslog_udp_listen"`
}

type StorageConfig struct {
	// May be a directory or a full file path. If directory, we'll use "logship.db" inside it.
	Path string `yaml:"path" json:"path"`
}

type Config struct {
	Server  ServerConfig  `yaml:"server" json:"server"`
	Storage StorageConfig `yaml:"storage" json:"storage"`
}

func Default() *Config {
	data := os.Getenv("DATA_DIR")
	if data == "" {
		data = "/var/lib/logship" // image will own this for nonroot
	}
	return &Config{
		Server: ServerConfig{
			HTTPListen:      ":8080",
			SyslogTCPListen: "",
			SyslogUDPListen: "",
		},
		Storage: StorageConfig{Path: data},
	}
}

// If path=="" use defaults; if file missing, still return defaults (no error).
func Load(path string) (*Config, error) {
	// If you already had YAML loading here, keep it; otherwise:
	return Default(), nil
}

// Helper for main to pick a config path from args (if you already had one, keep it)
func PathFromArgsOrDefault(args []string) string { return "" }
