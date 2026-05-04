# Load Tests

KWOK-based memory load test that deploys fake resources in increasing steps and measures peak memory usage of all pods in the cluster.

## How it works

```text
run-test.sh
│
├─ [SETUP_KWOK=true]
│   ├── Deploy KWOK controller (kwok-system namespace)
│   └── Create fake node (kwok-node-0)
│
├─ Start background memory sampler (kubectl top pod -A every 1s)
│
├─ Baseline: sample for SAMPLE_DURATION seconds (no resources)
│   └── snapshot peak memory → "baseline" column
│
└─ For each count in RESOURCE_COUNTS:
    ├── Create namespace  load-test-<count>
    │     └── [TEST_NS_LABEL] apply extra label
    ├── Render N resources from RESOURCE_TEMPLATE_PATH (envsubst)
    │     └── kubectl apply --server-side
    ├── Sample memory for SAMPLE_DURATION seconds
    │     └── snapshot peak memory → "<count> res" column
    ├── Check for failing pods cluster-wide
    └── Delete namespace load-test-<count> (wait up to DELETE_TIMEOUT)

[on exit]
├─ Stop sampler
├─ [CLEANUP_KWOK=true] Delete fake node, delete kwok-system namespace
└─ Print report (stdout + REPORT_PATH if set)
```

## Prerequisites

| Tool       | Purpose                     |
| ---------- | --------------------------- |
| `kubectl`  | cluster interaction         |
| `envsubst` | resource template rendering |

## Quick start

```bash
# Run the load test against the current kubeconfig cluster
bash hack/load/run-test.sh
```

## Scripts

### run-test.sh

Runs the load test against an existing cluster.

| Variable                 | Default                        | Description                                                       |
| ------------------------ | ------------------------------ | ----------------------------------------------------------------- |
| `RESOURCE_COUNTS`        | `500 1000 2000 3000 4000 5000` | Space-separated resource counts per step                          |
| `RESOURCE_TEMPLATE_PATH` | `resources/pod-template.yaml`  | Path to a YAML template for one resource                          |
| `SAMPLE_DURATION`        | `60`                           | Seconds to sample memory after resources are created              |
| `KWOK_VERSION`           | `v0.7.0`                       | KWOK release tag                                                  |
| `SETUP_KWOK`             | `true`                         | Install KWOK controller and fake node                             |
| `CLEANUP_KWOK`           | `true`                         | Delete KWOK namespace and fake node on exit                       |
| `DELETE_TIMEOUT`         | `300`                          | Seconds to wait for step namespace full deletion before next step |
| `REPORT_PATH`            | ``                             | Save report to this `.md` file path in addition to stdout         |
| `TEST_NS_LABEL`          | ``                             | Extra label applied to the per-step test namespace                |

```bash
# Custom counts
RESOURCE_COUNTS="500 1000 2000" bash hack/load/run-test.sh

# Custom resource template
RESOURCE_TEMPLATE_PATH=hack/load/resources/secret-template.yaml bash hack/load/run-test.sh

# Skip KWOK install (if already running) and keep it after the test
SETUP_KWOK=false CLEANUP_KWOK=false bash hack/load/run-test.sh

# Save report to a file
REPORT_PATH=results.md bash hack/load/run-test.sh

# Apply a namespace label (e.g. to enable Warden validation)
TEST_NS_LABEL="namespaces.warden.kyma-project.io/validate=system" bash hack/load/run-test.sh
```

## Resource templates

Templates are YAML files for a single resource. The script stamps out N copies per step using `envsubst`. Available variables:

| Variable            | Description                |
| ------------------- | -------------------------- |
| `${RESOURCE_INDEX}` | 1-based counter            |
| `${TEST_NAMESPACE}` | per-step namespace name    |
| `${KWOK_NODE_NAME}` | name of the fake KWOK node |

| Template                            | Kind      | Size                                                                   |
| ----------------------------------- | --------- | ---------------------------------------------------------------------- |
| `resources/pod-template.yaml`       | Pod       | 92 labels, 91 annotations, 5 containers, 2 init containers, 20 volumes |
| `resources/secret-template.yaml`    | Secret    | 100 labels, 100 annotations, 50 data fields (1000 random words each)   |
| `resources/configmap-template.yaml` | ConfigMap | 100 labels, 100 annotations, 50 data fields (1000 random words each)   |

## Report

The report is printed at the end (and on any exit). Each row is one pod; columns are steps. Baseline (pre-load) memory is always the first column.

```text
## KWOK Load Test — Peak Memory per Pod (Mi)
## Resource template: .../pod-template.yaml

| pod (namespace/name)              | baseline | 500 res | 1000 res |
| --------------------------------- | -------- | ------- | -------- |
| kube-system/metrics-server-xxx    | 279 Mi   | 295 Mi  | 348 Mi   |
| kyma-system/warden-operator-xxx   | 146 Mi   | 389 Mi  | 706 Mi   |
| kyma-system/some-crashing-pod-xxx | 12 Mi    | FAILING | FAILING  |
```

- Memory values show peak Mi observed during the sampling window.
- `FAILING` is shown instead of memory when the pod was in `CrashLoopBackOff`, `Error`, or `OOMKilled` state during that step.
