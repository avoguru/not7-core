package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config represents the NOT7 configuration
type Config struct {
	OpenAI  OpenAIConfig
	Server  ServerConfig
	Builtin BuiltinConfig
}

// OpenAIConfig holds OpenAI-specific configuration
type OpenAIConfig struct {
	APIKey             string
	DefaultModel       string
	DefaultTemperature float64
	DefaultMaxTokens   int
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Port          int
	ExecutionsDir string
	LogDir        string
}

// BuiltinConfig holds built-in tool provider settings
type BuiltinConfig struct {
	SerpAPIKey string
}

var globalConfig *Config

// LoadConfig loads configuration from a simple key-value file
func LoadConfig(filepath string) (*Config, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	cfg := &Config{
		// Set defaults
		OpenAI: OpenAIConfig{
			DefaultModel:       "gpt-4",
			DefaultTemperature: 0.7,
			DefaultMaxTokens:   2000,
		},
		Server: ServerConfig{
			Port:          8080,
			ExecutionsDir: "./executions",
			LogDir:        "./logs",
		},
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=value (standard .env format)
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("line %d: invalid format (expected: KEY=value)", lineNum)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Set config values based on key
		if err := setConfigValue(cfg, key, value); err != nil {
			return nil, fmt.Errorf("line %d: %w", lineNum, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Validate required fields
	if cfg.OpenAI.APIKey == "" {
		return nil, fmt.Errorf("OpenAI.api_key is required in config")
	}

	globalConfig = cfg
	return cfg, nil
}

// setConfigValue sets a configuration value based on key
func setConfigValue(cfg *Config, key, value string) error {
	switch key {
	// OpenAI settings
	case "OPENAI_API_KEY":
		cfg.OpenAI.APIKey = value
	case "OPENAI_DEFAULT_MODEL":
		cfg.OpenAI.DefaultModel = value
	case "OPENAI_DEFAULT_TEMPERATURE":
		temp, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("invalid temperature value: %s", value)
		}
		cfg.OpenAI.DefaultTemperature = temp
	case "OPENAI_DEFAULT_MAX_TOKENS":
		tokens, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid max_tokens value: %s", value)
		}
		cfg.OpenAI.DefaultMaxTokens = tokens

	// Server settings
	case "SERVER_PORT":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid port value: %s", value)
		}
		cfg.Server.Port = port
	case "SERVER_EXECUTIONS_DIR":
		cfg.Server.ExecutionsDir = value
	case "SERVER_LOG_DIR":
		cfg.Server.LogDir = value

	// Builtin tool settings
	case "SERP_API_KEY":
		cfg.Builtin.SerpAPIKey = value

	default:
		return fmt.Errorf("unknown config key: %s", key)
	}

	return nil
}

// Get returns the global configuration
func Get() *Config {
	if globalConfig == nil {
		panic("config not loaded - call LoadConfig first")
	}
	return globalConfig
}
