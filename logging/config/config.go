package config

import (
	"os"
	"path/filepath"

	"github.com/vrischmann/envconfig"
	"gopkg.in/yaml.v3"
)

// Config represents the logging configuration
type Config struct {
	LogLevel  string `envconfig:"default=info" yaml:"logLevel"`
	LogFormat string `envconfig:"default=json" yaml:"logFormat"`
}

// GetConfig loads logging configuration from environment variables with the given prefix.
// For example, with prefix "APP", it will look for APP_LOGLEVEL and APP_LOGFORMAT.
func GetConfig(prefix string) (Config, error) {
	cfg := Config{}
	err := envconfig.InitWithPrefix(&cfg, prefix)
	return cfg, err
}

// LoadConfig loads logging configuration from a YAML file.
func LoadConfig(path string) (Config, error) {
	cfg := Config{}
	cleanPath := filepath.Clean(path)
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(data, &cfg)
	return cfg, err
}
