// Package config loads the generator settings from YAML.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	URLs             []string `yaml:"urls"`
	UserAgents       []string `yaml:"user_agents"`
	RateLimit        int      `yaml:"rate_limit"`
	Timeout          int      `yaml:"timeout"`
	MaxRetries       int      `yaml:"max_retries"`
	WebSocketTimeout int      `yaml:"websocket_timeout"`
	TorrentLink      string   `yaml:"torrent_link"`
}

var (
	ErrNoURLs       = errors.New("urls is empty")
	ErrNoUserAgents = errors.New("user_agents is empty")
)

func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c Config
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	// a misspelled key fails here instead of decoding as zero
	dec.KnownFields(true)
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if len(c.URLs) == 0 {
		return nil, fmt.Errorf("%s: %w", path, ErrNoURLs)
	}
	if len(c.UserAgents) == 0 {
		return nil, fmt.Errorf("%s: %w", path, ErrNoUserAgents)
	}
	return &c, nil
}

func (c *Config) Save(path string) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	// two spaces keeps the file looking like the repo copy, yaml.v3 defaults to four
	enc.SetIndent(2)
	if err := enc.Encode(c); err != nil {
		return err
	}
	if err := enc.Close(); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
