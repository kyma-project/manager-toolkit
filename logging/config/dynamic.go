package config

import (
	"context"

	"github.com/kyma-project/manager-toolkit/logging/logger"
	"go.uber.org/zap"
)

// ReconfigureOnConfigChange monitors config changes and updates log level and format dynamically.
// It watches the configuration file at cfgPath, updates the atomic log level, and recreates the logger
// with the new format whenever the config file changes.
func ReconfigureOnConfigChange(ctx context.Context, log *zap.SugaredLogger, atomicLevel zap.AtomicLevel, cfgPath string) {
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

		// Recreate logger with current format from config
		format, err := logger.MapFormat(cfg.LogFormat)
		if err != nil {
			log.Errorf("failed to parse format '%s': %v", cfg.LogFormat, err)
			return
		}

		newLogger, err := logger.NewWithAtomicLevel(format, atomicLevel)
		if err != nil {
			log.Errorf("failed to create logger: %v", err)
			return
		}

		*log = *newLogger.WithContext()

		log.Infof("logger reconfigured with level '%s' and format '%s'", cfg.LogLevel, cfg.LogFormat)
	})
}
