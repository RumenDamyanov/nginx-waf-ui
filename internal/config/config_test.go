package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValid(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	content := `
server:
  listen: ":8081"
api:
  url: http://localhost:8080
  api_key: secret
  timeout: 5s
session:
  secret: test-secret
  max_age: 3600
database:
  path: /tmp/test.db
logging:
  level: debug
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Listen != ":8081" {
		t.Errorf("listen = %q", cfg.Server.Listen)
	}
	if cfg.API.URL != "http://localhost:8080" {
		t.Errorf("api.url = %q", cfg.API.URL)
	}
	if cfg.Session.MaxAge != 3600 {
		t.Errorf("session.max_age = %d", cfg.Session.MaxAge)
	}
}

func TestValidateMissingSecret(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	content := `
api:
  url: http://localhost:8080
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(cfgFile)
	if err == nil {
		t.Fatal("expected error for missing session.secret")
	}
}

func TestEnvExpansion(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")
	t.Setenv("TEST_UI_SECRET", "from-env")
	content := `
api:
  url: http://localhost:8080
session:
  secret: ${TEST_UI_SECRET}
`
	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Session.Secret != "from-env" {
		t.Errorf("secret = %q, want from-env", cfg.Session.Secret)
	}
}
