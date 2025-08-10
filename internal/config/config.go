package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DataDir    string
	HTTPAddr   string
	SyslogUDP  string
	SyslogTCP  string
	MaxRows    int
	Whitelist  map[string]struct{}
	Blacklist  map[string]struct{}
	Theme      string // "light" or "dark"
}

func FromEnv() *Config {
	cfg := &Config{
		DataDir:   env("DATA_DIR", "/var/lib/logship"),
		HTTPAddr:  env("HTTP_ADDR", ":8080"),
		SyslogUDP: env("SYSLOG_UDP_LISTEN", ":5514"), // enable by default
		SyslogTCP: env("SYSLOG_TCP_LISTEN", ""),      // disabled by default
		MaxRows:   envInt("MAX_ROWS", 200000),
		Theme:     env("THEME", "dark"),
	}
	cfg.Whitelist = toSet(env("IP_WHITELIST", ""))
	cfg.Blacklist = toSet(env("IP_BLACKLIST", ""))
	return cfg
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func toSet(csv string) map[string]struct{} {
	s := map[string]struct{}{}
	if csv == "" {
		return s
	}
	for _, p := range strings.Split(csv, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			s[p] = struct{}{}
		}
	}
	return s
}
