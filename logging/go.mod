module github.com/kyma-project/manager-toolkit/logging

go 1.24.0

toolchain go1.24.3

require (
	github.com/fsnotify/fsnotify v1.10.1
	github.com/go-logr/zapr v1.3.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.12.1
	github.com/vrischmann/envconfig v1.4.1
	go.uber.org/zap v1.28.0
	gopkg.in/yaml.v3 v3.0.1
	k8s.io/klog/v2 v2.130.1
)

require (
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/rogpeppe/go-internal v1.13.1 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sys v0.21.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
)
