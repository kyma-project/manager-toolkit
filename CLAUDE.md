# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Manager Toolkit is a multi-module Go repository that provides shared utilities for Kyma Manager projects. It is composed of three independent Go modules:

- `logging/` — Unified logging via `go.uber.org/zap`, with dynamic log level adjustment, JSON/text formats, and config-file watching.
- `installation/base/` — Kubernetes resource utilities: annotation disclaimers, resource deletion/verification, and predicates. Depends on `controller-runtime`.
- `installation/chart/` — Helm chart install/uninstall/verify lifecycle management with manifest caching and dry-run templating. Depends on `installation/base` and `helm.sh/helm/v3`.

There is also a custom GitHub Action under `.github/actions/test-rbac-propagation/` that is its own Go module for verifying Kubernetes RBAC propagation.

## Commands

Each module is developed independently — commands must be run from within the module's directory.

### Tests

```bash
cd logging && go test -v ./...
cd installation/base && go test -v ./...
cd installation/chart && go test -v ./...
```

Run a single test:

```bash
cd installation/chart && go test -v -run TestFunctionName ./...
```

### Lint

Each module has its own `.golangci.yml`. Run lint per module:

```bash
cd logging && golangci-lint run ./...
cd installation/base && golangci-lint run ./...
cd installation/chart && golangci-lint run ./...
```

## Architecture

### Module dependency graph

```
installation/chart  →  installation/base
installation/chart  →  helm.sh/helm/v3, controller-runtime, k8s.io/client-go
installation/base   →  controller-runtime, k8s.io/api, k8s.io/apimachinery
logging             →  go.uber.org/zap, go-logr, k8s.io/klog/v2, fsnotify
```

### Versioning

On every push to `main`, a CI job creates Git tags in the format `v0.YYMMDD.HHMMSS-<hash>`. For each changed submodule (e.g. `installation/chart`), an additional module-scoped tag is created: `installation/chart/v0.YYMMDD.HHMMSS-<hash>`. This means consumers pin to module-specific tags, not the repo root tag.

## Load Testing Tool

`tools/load/` contains a KWOK-based shell script that measures peak memory of all cluster pods under increasing fake resource load. It is **not a Go module** — no compilation needed.

### Run

```bash
bash tools/load/run-test.sh
```

Key env vars: `RESOURCE_COUNTS`, `RESOURCE_TEMPLATE_PATH`, `SAMPLE_DURATION`, `SETUP_KWOK`, `CLEANUP_KWOK`, `REPORT_PATH`, `TEST_NS_LABEL`. See `tools/load/README.md` for full reference.

Prerequisites: `kubectl` and `envsubst` on PATH, a configured kubeconfig pointing at the target cluster.

### Custom Actions

The `.github/actions/` directory contains reusable composite actions intended for use by other Kyma repos:

- `setup-go` — Sets up Go using the root `go.mod` version with caching.
- `create-k3d-cluster` — Spins up a local k3d cluster.
- `configure-kubeconfig-clusterrole` — Configures kubeconfig with a ClusterRole.
- `test-rbac-propagation` — Verifies RBAC rule propagation through the `view ⊂ edit ⊂ admin` ClusterRole hierarchy against a live cluster. Takes a YAML config declaring required and forbidden rules per role.
