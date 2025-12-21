package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestRunOnConfigChange_ContextCancellation(t *testing.T) {
	t.Run("should stop when context is canceled", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		initialConfig := `logLevel: info
logFormat: json
`
		err := os.WriteFile(configPath, []byte(initialConfig), 0600)
		require.NoError(t, err)

		logger := zaptest.NewLogger(t).Sugar()
		ctx, cancel := context.WithCancel(context.Background())

		done := make(chan struct{})
		callback := func(cfg Config) {}

		// when
		go func() {
			RunOnConfigChange(ctx, logger, configPath, callback)
			close(done)
		}()

		// wait a bit for the goroutine to start
		time.Sleep(100 * time.Millisecond)

		cancel()

		// then
		select {
		case <-done:
			// success
		case <-time.After(3 * time.Second):
			t.Fatal("RunOnConfigChange did not stop after context cancellation")
		}
	})
}

func TestFireCallbacks(t *testing.T) {
	t.Run("should fire all callbacks", func(t *testing.T) {
		// given
		cfg := Config{LogLevel: "debug", LogFormat: "json"}
		called1 := false
		called2 := false

		callback1 := func(c Config) {
			called1 = true
			assert.Equal(t, "debug", c.LogLevel)
		}

		callback2 := func(c Config) {
			called2 = true
			assert.Equal(t, "json", c.LogFormat)
		}

		// when
		fireCallbacks(cfg, callback1, callback2)

		// then
		assert.True(t, called1)
		assert.True(t, called2)
	})
}
