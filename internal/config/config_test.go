package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoad(t *testing.T) {
	path := write(t, `
urls:
  - https://example.com
user_agents:
  - "ua-a"
  - "ua-b"
rate_limit: 2
timeout: 15
websocket_timeout: 10
torrent_link: "magnet:?xt=urn:btih:00"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.URLs) != 1 || len(cfg.UserAgents) != 2 {
		t.Fatalf("got %d urls and %d user agents", len(cfg.URLs), len(cfg.UserAgents))
	}
	if cfg.RateLimit != 2 {
		t.Fatalf("rate_limit: got %d, want 2", cfg.RateLimit)
	}
	if cfg.TorrentLink == "" {
		t.Fatal("torrent_link was dropped")
	}
}

func TestLoadRejectsMisspelledKeys(t *testing.T) {
	path := write(t, `
urls: [https://example.com]
user_agents: [ua-a]
ratelimit: 5
`)
	if _, err := Load(path); err == nil {
		t.Fatal("a misspelled key loaded without a word")
	}
}

func TestLoadRejectsEmptyUserAgents(t *testing.T) {
	path := write(t, "urls: [https://example.com]\n")
	if _, err := Load(path); !errors.Is(err, ErrNoUserAgents) {
		t.Fatalf("got %v, want ErrNoUserAgents", err)
	}
}

func TestLoadAllowsUrlsToBeEmpty(t *testing.T) {
	path := write(t, "user_agents: [ua-a]\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("a stand-only config needs no urls: %v", err)
	}
	if len(cfg.URLs) != 0 {
		t.Fatalf("urls came from somewhere: %v", cfg.URLs)
	}
}

func TestSaveRoundTrips(t *testing.T) {
	path := write(t, "urls: [https://a.test]\nuser_agents: [ua-a]\n")
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	cfg.URLs = append(cfg.URLs, "https://b.test")
	if err := cfg.Save(path); err != nil {
		t.Fatal(err)
	}

	again, err := Load(path)
	if err != nil {
		t.Fatalf("reload after save: %v", err)
	}
	if len(again.URLs) != 2 {
		t.Fatalf("saved urls did not survive the round trip: %v", again.URLs)
	}
}
