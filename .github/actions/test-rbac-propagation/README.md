# test-rbac-propagation

A GitHub action that verifies RBAC propagation for the aggregated `admin`, `edit`, and `view` ClusterRoles in a Kubernetes cluster.

`admin`, `edit`, and `view` are standard Kubernetes ClusterRoles, which are always present in every cluster.

It reads a YAML config file that declares which rules each role **must** contain and which rules are **forbidden**, then cross-checks those expectations against the live ClusterRoles via the Kubernetes API.

## How It Works

Kubernetes aggregates RBAC rules upward through the hierarchy `view ⊂ edit ⊂ admin`.  
The action models this automatically:

| ClusterRole checked | Rules evaluated |
|---------------------|-----------------|
| `view`              | `view.rules` |
| `edit`              | `edit.rules` + `view.rules` |
| `admin`             | `admin.rules` + `edit.rules` + `view.rules` |

`forbidden` entries are checked **per-role only** — they do **not** inherit upward. The step fails if any forbidden rule is found in that role.

## Prerequisites

| Requirement | Details |
|-------------|---------|
| **Go** | Must be available in `PATH`. Use [`kyma-project/manager-toolkit/.github/actions/setup-go`](../setup-go) before this action. |
| **kubeconfig** | A valid kubeconfig pointing to the target cluster must be present (e.g. set via `KUBECONFIG` env var or default `~/.kube/config`). |

## Inputs

| Input | Required | Description |
|-------|----------|-------------|
| `config` | ✅ | Path to the RBAC config YAML file (see format below). |

## Config file format

```yaml
admin:
  rules: []
  forbidden: []

edit:
  rules:
    - apiGroups:
        - operator.kyma-project.io
      resources:
        - dockerregistries
      verbs:
        - create
        - delete
        - get
        - list
        - patch
        - update
        - watch
    - apiGroups:
        - operator.kyma-project.io
      resources:
        - dockerregistries/status
      verbs:
        - get
  forbidden:
    - apiGroups:
        - ""
      resources:
        - namespaces
      verbs:
        - create

view:
  rules:
    # kyma-docker-registry-view
    - apiGroups:
        - operator.kyma-project.io
      resources:
        - dockerregistries
        - dockerregistries/status
      verbs:
        - get
        - list
        - watch
  forbidden: []
```

Each section (`admin`, `edit`, `view`) accepts:
- **`rules`** – permissions that **must** be present in the ClusterRole (after inheritance is applied).
- **`forbidden`** – permissions that **must not** be present in that specific ClusterRole (no inheritance). The step fails if any are found.

## Usage

```yaml
- uses: actions/checkout@v4

- uses: kyma-project/manager-toolkit/.github/actions/setup-go@main

- name: Your component setup
  run: your component setup steps here, e.g. installing the operator
- uses: kyma-project/manager-toolkit/.github/actions/test-rbac-propagation@main
  with:
    config: ${{ github.workspace }}/path/to/your/config.yaml
```

## Exit behaviour

- Prints `PASS: clusterrole/<name>` for every role that satisfies all expectations.
- Prints `FAIL: clusterrole/<name>: …` with a list of missing/forbidden permissions for any role that does not.
