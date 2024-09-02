package main

import (
    "os"
    "testing"

    "github.com/spf13/viper"
)

func TestLoadConfig(t *testing.T) {
    // temp config test
    configFile := "test_config.yaml"
    config := []byte(`
urls:
  - https://example.com
  - https://google.com
user_agents:
  - "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3"
  - "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_12_6) AppleWebKit/603.3.8 (KHTML, like Gecko) Version/10.1.2 Safari/603.3.8"
rate_limit: 1
`)
    err := os.WriteFile(configFile, config, 0644)
    if err != nil {
        t.Errorf("Failed to create test config file: %v", err)
    }
    defer os.Remove(configFile)

    // set path
    viper.SetConfigFile(configFile)
    
    cfg := loadConfig()

    // is correct load
    if len(cfg.URLs) != 2 {
        t.Errorf("Expected 2 URLs, got %d", len(cfg.URLs))
    }
    if len(cfg.UserAgents) != 2 {
        t.Errorf("Expected 2 User-Agents, got %d", len(cfg.UserAgents))
    }
    if cfg.RateLimit != 1 {
        t.Errorf("Expected rate limit 1, got %d", cfg.RateLimit)
    }
}

