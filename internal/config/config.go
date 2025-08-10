package config

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	Server  ServerConfig
	Storage StorageConfig
	Lists   ListConfig
}

type ServerConfig struct {
	ListenAddr string // :8080
	SyslogUDP  string // :5514
	SyslogTCP  string // :5514
}

type StorageConfig struct {
	DataDir  string
	MaxMB    int
	DroppedN int
	RecentN  int
}

type ListConfig struct {
	Whitelist    []string
	Blacklist    []string
	DefaultAllow bool
}

func env(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
func envInt(key string, def int) int {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func Load(_ string) *Config {
	return &Config{
		Server: ServerConfig{
			ListenAddr: env("LISTEN_ADDR", ":8080"),
			SyslogUDP:  env("SYSLOG_UDP_LISTEN", ":5514"),
			SyslogTCP:  env("SYSLOG_TCP_LISTEN", ":5514"),
		},
		Storage: StorageConfig{
			DataDir:  env("DATA_DIR", "/var/lib/logship"),
			MaxMB:    envInt("MAX_DB_MB", 512),
			DroppedN: envInt("DROPPED_RECENT_N", 100),
			RecentN:  envInt("RECENT_N", 100),
		},
		Lists: ListConfig{
			Whitelist:    split(env("SRC_WHITELIST", "")),
			Blacklist:    split(env("SRC_BLACKLIST", "")),
			DefaultAllow: strings.ToLower(env("SRC_DEFAULT_ALLOW", "true")) != "false",
		},
	}
}

func split(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func PathFromArgsOrDefault(args []string) string {
	fs := flag.NewFlagSet("logship", flag.ContinueOnError)
	var p string
	fs.StringVar(&p, "config", "", "config file (optional)")
	_ = fs.Parse(args)
	if p == "" {
		return ""
	}
	if !filepath.IsAbs(p) {
		wd, _ := os.Getwd()
		p = filepath.Join(wd, p)
	}
	return p
}
