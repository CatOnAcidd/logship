package config

import (
	"log"
	"os"
)

type FileTail struct {
	Path string `yaml:"path"`
	Glob string `yaml:"glob"` // string pattern (e.g., "*.log"), matches main.go usage
}

type ServerConfig struct {
	HTTPListen      string     `yaml:"http_listen"`
	SyslogTCPListen string     `yaml:"syslog_tcp_listen"`
	SyslogUDPListen string     `yaml:"syslog_udp_listen"`
	FileTails       []FileTail `yaml:"file_tails"`
}

type StorageConfig struct {
	Path    string `yaml:"path"`
	DataDir string `yaml:"data_dir"`
}

type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Storage StorageConfig `yaml:"storage"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPListen:      ":8080",
			SyslogTCPListen: "",
			SyslogUDPListen: "",
			FileTails:       nil,
		},
		Storage: StorageConfig{
			Path:    "/data",
			DataDir: "/data",
		},
	}
}

func PathFromArgsOrDefault() string {
	if len(os.Args) > 1 && os.Args[1] != "" {
		return os.Args[1]
	}
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		return v
	}
	return "/app/config.yaml"
}

func Load(path string) (*Config, error) {
	cfg := Default()
	if v := os.Getenv("HTTP_LISTEN"); v != "" { cfg.Server.HTTPListen = v }
	if v := os.Getenv("SYSLOG_TCP_LISTEN"); v != "" { cfg.Server.SyslogTCPListen = v }
	if v := os.Getenv("SYSLOG_UDP_LISTEN"); v != "" { cfg.Server.SyslogUDPListen = v }
	if v := os.Getenv("DATA_DIR"); v != "" { cfg.Storage.DataDir = v; cfg.Storage.Path = v }
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			log.Printf("config: %s not found, using defaults", path)
		}
	}
	return cfg, nil
}
