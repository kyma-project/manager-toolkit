package config

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kyma-project/manager-toolkit/logging/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
	"go.uber.org/zap/zaptest/observer"
)

func TestReconfigureOnConfigChange(t *testing.T) {
	t.Run("should update level when format changes", func(t *testing.T) {
		// given
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")

		initialConfig := `logLevel: info
logFormat: json
`
		err := os.WriteFile(cfgPath, []byte(initialConfig), 0600)
		require.NoError(t, err)

		atomicLevel := zap.NewAtomicLevelAt(zapcore.InfoLevel)
		loggerWrapper, err := logger.NewWithAtomicLevel(logger.JSON, atomicLevel)
		require.NoError(t, err)
		log := loggerWrapper.WithContext()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		go func() {
			ReconfigureOnConfigChange(ctx, log, atomicLevel, cfgPath)
			close(done)
		}()

		time.Sleep(100 * time.Millisecond)

		// when
		updatedConfig := `logLevel: debug
logFormat: console
`
		err = os.WriteFile(cfgPath, []byte(updatedConfig), 0600)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		// then
		assert.Equal(t, zapcore.DebugLevel, atomicLevel.Level())

		cancel()
		select {
		case <-done:
			// Context canceled and watcher stopped
		case <-time.After(2 * time.Second):
			t.Fatal("watcher did not stop after context cancellation")
		}
	})

	t.Run("should update level when only level changes", func(t *testing.T) {
		// given
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")

		initialConfig := `logLevel: info
logFormat: json
`
		err := os.WriteFile(cfgPath, []byte(initialConfig), 0600)
		require.NoError(t, err)

		atomicLevel := zap.NewAtomicLevelAt(zapcore.InfoLevel)
		loggerWrapper, err := logger.NewWithAtomicLevel(logger.JSON, atomicLevel)
		require.NoError(t, err)
		log := loggerWrapper.WithContext()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		go func() {
			ReconfigureOnConfigChange(ctx, log, atomicLevel, cfgPath)
			close(done)
		}()

		time.Sleep(100 * time.Millisecond)

		// when
		updatedConfig := `logLevel: warn
logFormat: json
`
		err = os.WriteFile(cfgPath, []byte(updatedConfig), 0600)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		// then
		assert.Equal(t, zapcore.WarnLevel, atomicLevel.Level())

		cancel()
		select {
		case <-done:
			// Context canceled and watcher stopped
		case <-time.After(2 * time.Second):
			t.Fatal("watcher did not stop after context cancellation")
		}
	})

	t.Run("should not change level when only format changes", func(t *testing.T) {
		// given
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")

		initialConfig := `logLevel: info
logFormat: json
`
		err := os.WriteFile(cfgPath, []byte(initialConfig), 0600)
		require.NoError(t, err)

		atomicLevel := zap.NewAtomicLevelAt(zapcore.InfoLevel)
		loggerWrapper, err := logger.NewWithAtomicLevel(logger.JSON, atomicLevel)
		require.NoError(t, err)
		log := loggerWrapper.WithContext()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		go func() {
			ReconfigureOnConfigChange(ctx, log, atomicLevel, cfgPath)
			close(done)
		}()

		time.Sleep(100 * time.Millisecond)

		// when: only format changes, level stays info
		updatedConfig := `logLevel: info
logFormat: console
`
		err = os.WriteFile(cfgPath, []byte(updatedConfig), 0600)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		// then: level should remain unchanged
		assert.Equal(t, zapcore.InfoLevel, atomicLevel.Level())

		cancel()
		select {
		case <-done:
			// Context canceled and watcher stopped
		case <-time.After(2 * time.Second):
			t.Fatal("watcher did not stop after context cancellation")
		}
	})

	t.Run("should work with shared atomic level", func(t *testing.T) {
		// given
		tmpDir := t.TempDir()
		cfgPath := filepath.Join(tmpDir, "config.yaml")

		initialConfig := `logLevel: info
logFormat: json
`
		err := os.WriteFile(cfgPath, []byte(initialConfig), 0600)
		require.NoError(t, err)

		atomicLevel := zap.NewAtomicLevelAt(zapcore.InfoLevel)

		observedZapCore, observedLogs := observer.New(atomicLevel)

		encoderConfig := zap.NewProductionEncoderConfig()
		encoderConfig.EncodeTime = zapcore.RFC3339TimeEncoder
		encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		encoderConfig.TimeKey = "timestamp"
		encoderConfig.MessageKey = "message"
		encoder := zapcore.NewJSONEncoder(encoderConfig)
		defaultCore := zapcore.NewCore(encoder, zapcore.Lock(os.Stderr), atomicLevel)

		testLogger := zap.New(zapcore.NewTee(observedZapCore, defaultCore), zap.AddCaller()).Sugar()

		testLogger.Info("before reconfiguration")
		require.Equal(t, 1, observedLogs.Len())

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan struct{})
		go func() {
			ReconfigureOnConfigChange(ctx, testLogger, atomicLevel, cfgPath)
			close(done)
		}()

		time.Sleep(100 * time.Millisecond)

		// when
		updatedConfig := `logLevel: debug
logFormat: console
`
		err = os.WriteFile(cfgPath, []byte(updatedConfig), 0600)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		// then
		testLogger.Debug("after reconfiguration")
		require.Greater(t, observedLogs.Len(), 1, "observer core should still work via shared atomicLevel")
		assert.Equal(t, zapcore.DebugLevel, atomicLevel.Level())

		cancel()
		select {
		case <-done:
			// Context canceled and watcher stopped
		case <-time.After(2 * time.Second):
			t.Fatal("watcher did not stop after context cancellation")
		}
	})
}

func TestSetInitialFormat(t *testing.T) {
	t.Run("should set and get format", func(t *testing.T) {
		// when
		SetInitialFormat("json")

		// then
		assert.Equal(t, "json", GetCurrentFormat())
	})

	t.Run("should update format", func(t *testing.T) {
		// given
		SetInitialFormat("json")

		// when
		SetInitialFormat("console")

		// then
		assert.Equal(t, "console", GetCurrentFormat())
	})
}

func TestReconfigureOnConfigChangeWithRestart(t *testing.T) {
	t.Run("should update log level without restart", func(t *testing.T) {
		// given
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")

		initialConfig := `logLevel: info
logFormat: json
`
		err := os.WriteFile(configPath, []byte(initialConfig), 0600)
		require.NoError(t, err)

		cfg, err := LoadConfig(configPath)
		require.NoError(t, err)

		atomic := zap.NewAtomicLevel()
		atomic.SetLevel(zapcore.InfoLevel)

		testLogger := zaptest.NewLogger(t).Sugar()

		SetInitialFormat(cfg.LogFormat)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		go ReconfigureOnConfigChangeWithRestart(ctx, testLogger, atomic, configPath)

		time.Sleep(100 * time.Millisecond)

		// when
		updatedConfig := `logLevel: debug
logFormat: json
`
		err = os.WriteFile(configPath, []byte(updatedConfig), 0600)
		require.NoError(t, err)

		time.Sleep(2 * time.Second)

		// then
		assert.Equal(t, zapcore.DebugLevel, atomic.Level())
		assert.Equal(t, "json", GetCurrentFormat())
	})
}
