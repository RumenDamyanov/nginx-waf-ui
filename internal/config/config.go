package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	API      APIConfig      `yaml:"api"`
	Session  SessionConfig  `yaml:"session"`
	Database DatabaseConfig `yaml:"database"`
	Logging  LogConfig      `yaml:"logging"`
}

type ServerConfig struct {
	Listen string    `yaml:"listen"`
	TLS    TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	Cert string `yaml:"cert"`
	Key  string `yaml:"key"`
}

type APIConfig struct {
	URL     string        `yaml:"url"`
	APIKey  string        `yaml:"api_key"`
	Timeout time.Duration `yaml:"timeout"`
}

type SessionConfig struct {
	Secret string `yaml:"secret"`
	MaxAge int    `yaml:"max_age"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func (a *APIConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type raw struct {
		URL     string `yaml:"url"`
		APIKey  string `yaml:"api_key"`
		Timeout string `yaml:"timeout"`
	}
	var r raw
	if err := unmarshal(&r); err != nil {
		return err
	}
	a.URL = r.URL
	a.APIKey = r.APIKey
	if r.Timeout != "" {
		d, err := time.ParseDuration(r.Timeout)
		if err != nil {
			return fmt.Errorf("invalid api timeout: %w", err)
		}
		a.Timeout = d
	}
	return nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	expanded := os.ExpandEnv(string(data))

	cfg := &Config{
		Server: ServerConfig{Listen: ":8081"},
		API: APIConfig{
			URL:     "http://127.0.0.1:8080",
			Timeout: 10 * time.Second,
		},
		Session: SessionConfig{MaxAge: 86400},
		Database: DatabaseConfig{Path: "/var/lib/nginx-waf-ui/data.db"},
		Logging: LogConfig{Level: "info", Format: "text"},
	}

	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.Server.Listen == "" {
		return fmt.Errorf("server.listen is required")
	}
	if c.API.URL == "" {
		return fmt.Errorf("api.url is required")
	}
	if c.Session.Secret == "" {
		return fmt.Errorf("session.secret is required")
	}
	return nil
}
