package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig(t *testing.T) {
	t.Run("should load config from environment variables with prefix", func(t *testing.T) {
		// given
		os.Setenv("APP_LOGLEVEL", "debug")
		os.Setenv("APP_LOGFORMAT", "console")
		defer os.Unsetenv("APP_LOGLEVEL")
		defer os.Unsetenv("APP_LOGFORMAT")

		// when
		cfg, err := GetConfig("APP")

		// then
		require.NoError(t, err)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, "console", cfg.LogFormat)
	})

	t.Run("should use default values when env vars not set", func(t *testing.T) {
		// when
		cfg, err := GetConfig("NONEXISTENT")

		// then
		require.NoError(t, err)
		assert.Equal(t, "info", cfg.LogLevel)
		assert.Equal(t, "json", cfg.LogFormat)
	})
}

func TestLoadConfig(t *testing.T) {
	t.Run("should load valid config", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		configContent := `logLevel: debug
logFormat: console
`
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		require.NoError(t, err)

		// when
		cfg, err := LoadConfig(configPath)

		// then
		require.NoError(t, err)
		assert.Equal(t, "debug", cfg.LogLevel)
		assert.Equal(t, "console", cfg.LogFormat)
	})

	t.Run("should return error for non-existent file", func(t *testing.T) {
		// when
		_, err := LoadConfig("/non/existent/path.yaml")

		// then
		require.Error(t, err)
	})

	t.Run("should return error for invalid YAML", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		configContent := `invalid: yaml: content: [[[`
		err := os.WriteFile(configPath, []byte(configContent), 0600)
		require.NoError(t, err)

		// when
		_, err = LoadConfig(configPath)

		// then
		require.Error(t, err)
	})
}
