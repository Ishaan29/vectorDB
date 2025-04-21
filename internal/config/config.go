package config

import (
	"os"

	"github.com/ishaan29/vectorDB/internal/logger"
	"gopkg.in/yaml.v3"
)

// Config holds all configuration for the vector database
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Storage  StorageConfig  `yaml:"storage"`
	Index    IndexConfig    `yaml:"index"`
	Database DatabaseConfig `yaml:"database"`
	Logging  logger.Config  `yaml:"logging"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// StorageConfig holds storage-specific configuration
type StorageConfig struct {
	Path string `yaml:"path"`
}

// IndexConfig holds indexing-specific configuration
type IndexConfig struct {
	Type       string `yaml:"type"`
	Dimensions int    `yaml:"dimensions"`
}

// DatabaseConfig holds database-specific configuration
type DatabaseConfig struct {
	MaxVectors int `yaml:"max_vectors"`
}

// Load reads the configuration file and returns a Config struct
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, err
	}

	return config, nil
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Storage: StorageConfig{
			Path: "data",
		},
		Index: IndexConfig{
			Type:       "hnsw",
			Dimensions: 128,
		},
		Database: DatabaseConfig{
			MaxVectors: 1000000,
		},
	}
}
