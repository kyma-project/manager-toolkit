package config

import (
	"context"

	"github.com/kyma-project/manager-toolkit/logging/logger"
	"go.uber.org/zap"
)

// ReconfigureOnConfigChange monitors config changes and updates log level dynamically.
// It watches the configuration file at cfgPath and updates the atomic log level
// whenever the config file changes.
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

		log.Infof("logging level set to '%s'.", cfg.LogLevel)
	})
}
