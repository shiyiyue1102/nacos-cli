package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the Nacos CLI configuration
type Config struct {
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	AuthType  string `yaml:"authType"` // nacos | aliyun
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	AccessKey string `yaml:"accessKey"` // Aliyun AK（AuthType=aliyun 时使用）
	SecretKey string `yaml:"secretKey"` // Aliyun SK
	Namespace string `yaml:"namespace"`
}

// LoadConfig loads configuration from a file
func LoadConfig(configPath string) (*Config, error) {
	// Expand home directory if needed
	if configPath == "~" || (len(configPath) > 1 && configPath[:2] == "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		if configPath == "~" {
			configPath = homeDir
		} else {
			configPath = filepath.Join(homeDir, configPath[2:])
		}
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check if file exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", absPath)
	}

	// Read file
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse YAML
	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// GetServerAddr returns the server address in format "host:port"
func (c *Config) GetServerAddr() string {
	if c.Host == "" {
		return ""
	}
	// If port is 0 or not set, check if host already contains port
	if c.Port == 0 {
		// Check if host already contains ":"
		if strings.Contains(c.Host, ":") {
			return c.Host
		}
		// Default to port 8848 if not specified
		return fmt.Sprintf("%s:8848", c.Host)
	}
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
