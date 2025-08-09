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
	Server  ServerConfig `yaml:"server"`
	DataDir string       `yaml:"data_dir"`
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

// PathFromArgsOrDefault returns args[1] if provided, otherwise a sane default path.
func PathFromArgsOrDefault(args []string) string {
	if len(args) > 1 && args[1] != "" {
		return args[1]
	}
	if v := os.Getenv("CONFIG_PATH"); v != "" {
		return v
	}
	return "/app/config.yaml"
}

// Load is minimal for Base; returns defaults and no error if file missing.
func Load(path string) (*Config, error) {
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
	return cfg, nil
}
