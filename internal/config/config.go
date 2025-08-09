package config

import (
	"flag"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		HTTPListen      string `yaml:"http_listen"`
		SyslogTCPListen string `yaml:"syslog_tcp_listen"`
		SyslogUDPListen string `yaml:"syslog_udp_listen"`
		FileTails       []struct {
			Path string `yaml:"path"`
			Glob string `yaml:"glob"`
		} `yaml:"file_tails"`
	} `yaml:"server"`

	Storage struct {
		Path          string `yaml:"path"`
		RetentionDays int    `yaml:"retention_days"`
		MaxDBMB       int    `yaml:"max_db_mb"`
	} `yaml:"storage"`

	Transforms []TransformRule `yaml:"transforms"`

	Forwarders []Forwarder `yaml:"forwarders"`

	UI struct {
		Enabled bool   `yaml:"enabled"`
		Auth    string `yaml:"auth"`
	} `yaml:"ui"`
}

type TransformRule struct {
	Name string `yaml:"name"`
	When string `yaml:"when"` // reserved (not implemented yet)
	JQ   string `yaml:"jq"`
}

type Forwarder struct {
	Name   string            `yaml:"name"`
	Type   string            `yaml:"type"` // only "http" for MVP
	URL    string            `yaml:"url"`
	Header map[string]string `yaml:"headers"`
	Batch  struct {
		Size    int    `yaml:"size"`
		Timeout string `yaml:"timeout"`
	} `yaml:"batch"`
	Retry struct {
		MaxAttempts int    `yaml:"max_attempts"`
		Backoff     string `yaml:"backoff"`
		MinBackoff  string `yaml:"min_backoff"`
		MaxBackoff  string `yaml:"max_backoff"`
	} `yaml:"retry"`
}

func PathFromArgsOrDefault() string {
	var p string
	flag.StringVar(&p, "config", "/app/config.yaml", "path to config.yaml")
	flag.Parse()
	return p
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, err
	}
	// Defaults
	if c.Server.HTTPListen == "" {
		c.Server.HTTPListen = ":8080"
	}
	if c.Server.SyslogTCPListen == "" {
		c.Server.SyslogTCPListen = ":514"
	}
	if c.Server.SyslogUDPListen == "" {
		c.Server.SyslogUDPListen = ":514"
	}
	if c.Storage.Path == "" {
		c.Storage.Path = "/data/logs.db"
	}
	if c.Storage.RetentionDays == 0 {
		c.Storage.RetentionDays = 7
	}
	if c.Storage.MaxDBMB == 0 {
		c.Storage.MaxDBMB = 2048
	}
	return &c, nil
}
