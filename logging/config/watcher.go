package config

import (
	"context"
	"errors"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

const (
	notificationDelay = 1 * time.Second
)

// CallbackFn is a function that is called when the configuration changes
type CallbackFn func(Config)

// RunOnConfigChange monitors a configuration file for changes and runs callback functions
// when the file is modified. This is designed to work with Kubernetes ConfigMaps which
// update files via atomic symlink changes.
func RunOnConfigChange(ctx context.Context, log *zap.SugaredLogger, path string, callbacks ...CallbackFn) {
	log.Info("config notifier started")

	for {
		err := fireCallbacksOnConfigChange(ctx, log, path, callbacks...)
		if err != nil && errors.Is(err, context.Canceled) {
			log.Info("context canceled")
			return
		}
		if err != nil {
			log.Error(err)
			// wait 1 sec not to burn out the container when errors occur repeatedly
			time.Sleep(notificationDelay)
		}
	}
}

func fireCallbacksOnConfigChange(ctx context.Context, log *zap.SugaredLogger, path string, callbacks ...CallbackFn) error {
	err := notifyModification(ctx, path)
	if err != nil {
		return err
	}

	log.Info("config file change detected")

	cfg, err := LoadConfig(path)
	if err != nil {
		return err
	}

	log.Debugf("firing '%d' callbacks", len(callbacks))

	fireCallbacks(cfg, callbacks...)
	return nil
}

func fireCallbacks(cfg Config, funcs ...CallbackFn) {
	for i := range funcs {
		fn := funcs[i]
		fn(cfg)
	}
}

// notifyModification watches for file modifications using fsnotify.
// It watches both the file directly and its parent directory to catch both:
// - Direct file modifications (Write events)
// - Kubernetes ConfigMap updates (which use atomic symlink changes, triggering Create events)
func notifyModification(ctx context.Context, path string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer func() {
		_ = watcher.Close()
	}()

	// Watch the file directly to catch Write events
	if err := watcher.Add(path); err != nil {
		return err
	}

	// Also watch the directory to catch Kubernetes ConfigMap atomic updates
	configDir := filepath.Dir(path)
	if err := watcher.Add(configDir); err != nil {
		return err
	}

	// Wait for any filesystem event
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-watcher.Events:
		return nil
	case err := <-watcher.Errors:
		return err
	}
}
