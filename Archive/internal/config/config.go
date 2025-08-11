package config

import (
	"log"
	"os"
)

type Config struct {
	DataDir        string
	HTTPListen     string
	SyslogUDP      string
	SyslogTCP      string
	FileTailPath   string
	FileTailGlob   bool
}

func FromEnv() *Config {
	c := &Config{
		DataDir:    env("DATA_DIR", "/var/lib/logship"),
		HTTPListen: env("HTTP_LISTEN", ":8080"),
		SyslogUDP:  env("SYSLOG_UDP_LISTEN", ":5514"),
		SyslogTCP:  env("SYSLOG_TCP_LISTEN", ":5514"),
	}

	// Optional file tail
	c.FileTailPath = os.Getenv("FILE_TAIL_PATH")
	if os.Getenv("FILE_TAIL_GLOB") == "1" || os.Getenv("FILE_TAIL_GLOB") == "true" {
		c.FileTailGlob = true
	}

	if os.Getenv("LOGSHIP_DEBUG") != "" {
		log.Printf("config: %+v", c)
	}
	return c
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
