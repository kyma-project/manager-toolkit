package config

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ReconfigureOnConfigChange monitors config changes and updates log level dynamically.
// It watches the configuration file at cfgPath and updates the atomic log level when changes are detected.
func ReconfigureOnConfigChange(ctx context.Context, log *zap.SugaredLogger, atomic zap.AtomicLevel, cfgPath string) {
	RunOnConfigChange(ctx, log, cfgPath, func(cfg Config) {
		level, err := zapcore.ParseLevel(cfg.LogLevel)
		if err != nil {
			log.Error(err)
			return
		}

		atomic.SetLevel(level)
		log.Infof("loggers reconfigured with level '%s' and format '%s'", cfg.LogLevel, cfg.LogFormat)
	})
}
