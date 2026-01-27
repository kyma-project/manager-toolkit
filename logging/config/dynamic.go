package config

import (
	"context"
	"os"
	"sync"

	"github.com/kyma-project/manager-toolkit/logging/logger"
	"go.uber.org/zap"
)

var (
	globalCurrentFormat string
	globalFormatMutex   sync.RWMutex
)

// ReconfigureOnConfigChange monitors config changes and updates log level and format dynamically.
// It watches the configuration file at cfgPath, updates the atomic log level, and recreates the logger
// with the new format whenever the config file changes.
func ReconfigureOnConfigChange(ctx context.Context, log *zap.SugaredLogger, atomicLevel zap.AtomicLevel, cfgPath string) {
	RunOnConfigChange(ctx, log, cfgPath, func(cfg Config) {
		// Update log level dynamically
		level, err := logger.MapLevel(cfg.LogLevel)
		if err != nil {
			log.Error(err)
			return
		}
		zapLevel, err := level.ToZapLevel()
		if err != nil {
			log.Error(err)
			return
		}
		atomicLevel.SetLevel(zapLevel)

		log.Infof("logger reconfigured with level '%s'. Format changes require pod restart (requested format: '%s')", cfg.LogLevel, cfg.LogFormat)
	})
}

// ReconfigureOnConfigChangeWithRestart monitors config changes and updates log level dynamically.
// When log format changes, it triggers a graceful pod restart by calling os.Exit(0).
func ReconfigureOnConfigChangeWithRestart(ctx context.Context, log *zap.SugaredLogger, atomicLevel zap.AtomicLevel, cfgPath string) {
	RunOnConfigChange(ctx, log, cfgPath, func(cfg Config) {
		// Update log level
		level, err := logger.MapLevel(cfg.LogLevel)
		if err != nil {
			log.Error(err)
			return
		}
		zapLevel, err := level.ToZapLevel()
		if err != nil {
			log.Error(err)
			return
		}
		atomicLevel.SetLevel(zapLevel)

		// Check if format changed
		globalFormatMutex.RLock()
		oldFormat := globalCurrentFormat
		formatChanged := globalCurrentFormat != "" && globalCurrentFormat != cfg.LogFormat
		globalFormatMutex.RUnlock()

		if formatChanged {
			log.Infof("log format changed from '%s' to '%s', restarting pod to apply new format", oldFormat, cfg.LogFormat)
			// We need to exit here to enable format change
			os.Exit(0)
		}

		log.Infof("logger reconfigured with level '%s' and format '%s'", cfg.LogLevel, cfg.LogFormat)
	})
}

// SetInitialFormat sets the initial log format for change detection.
// This must be called at application startup before ReconfigureOnConfigChangeWithRestart.
func SetInitialFormat(format string) {
	globalFormatMutex.Lock()
	globalCurrentFormat = format
	globalFormatMutex.Unlock()
}

// GetCurrentFormat returns the currently set log format.
func GetCurrentFormat() string {
	globalFormatMutex.RLock()
	defer globalFormatMutex.RUnlock()
	return globalCurrentFormat
}
