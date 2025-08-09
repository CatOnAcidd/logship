package config

import (
	"log"
	"os"
)

type ServerConfig struct {
	HTTPListen      string `yaml:"http_listen"`
	SyslogTCPListen string `yaml:"syslog_tcp_listen"`
	SyslogUDPListen string `yaml:"syslog_udp_listen"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	DataDir string      `yaml:"data_dir"`
}

func Default() *Config {
	return &Config{
		Server: ServerConfig{
			HTTPListen:      ":8080",
			SyslogTCPListen: "",
			SyslogUDPListen: "",
		},
		DataDir: "/data",
	}
}

// Load is intentionally minimal so Base compiles even without YAML parser.
func Load(path string) *Config {
	cfg := Default()
	if v := os.Getenv("HTTP_LISTEN"); v != "" { cfg.Server.HTTPListen = v }
	if v := os.Getenv("SYSLOG_TCP_LISTEN"); v != "" { cfg.Server.SyslogTCPListen = v }
	if v := os.Getenv("SYSLOG_UDP_LISTEN"); v != "" { cfg.Server.SyslogUDPListen = v }
	if v := os.Getenv("DATA_DIR"); v != "" { cfg.DataDir = v }
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			log.Printf("config: %s not found, using defaults", path)
		}
	}
	return cfg
}
