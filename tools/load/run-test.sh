#!/usr/bin/env bash
set -euo pipefail

# ---------------------------------------------------------------------------
# KWOK load test — deploys fake resources in increasing steps and collects
# peak memory for all pods in the cluster (kubectl top pod -A).
#
# Env vars:
#   RESOURCE_COUNTS        Space-separated resource counts to test  (default: 500 1000 2000 3000 4000 5000)
#   RESOURCE_TEMPLATE_PATH Path to a YAML template for one resource (default: <script-dir>/resources/fake-pod.yaml)
#                          Template may use envsubst variables:
#                            ${RESOURCE_INDEX}   — 1-based counter
#                            ${TEST_NAMESPACE}   — name of the per-step namespace
#                            ${KWOK_NODE_NAME}   — name of the fake KWOK node
#   SAMPLE_DURATION        Seconds to sample after resources created (default: 60)
#   KWOK_VERSION           KWOK release tag                          (default: v0.7.0)
#   SETUP_KWOK             Install KWOK controller and fake node (true/false) (default: true)
#   CLEANUP_KWOK           Delete KWOK namespace on exit (true/false) (default: true)
#   DELETE_TIMEOUT         Seconds to wait for step namespace deletion (default: 120)
#   REPORT_PATH            Save report to this .md file path         (default: "")
#   TEST_NS_LABEL          Extra label applied to test ns            (default: "")
#
# Example:
#   RESOURCE_COUNTS="500 1000 2000" bash hack/load/run-test.sh
#   RESOURCE_TEMPLATE_PATH=my-resource.yaml bash hack/load/run-test.sh
# ---------------------------------------------------------------------------

log() { echo "[$(date +%T)] $*"; }
die() { echo "ERROR: $*" >&2; exit 1; }

POD_COUNTS="${POD_COUNTS:-}" # kept for backward compat — overrides RESOURCE_COUNTS
RESOURCE_COUNTS="${POD_COUNTS:-${RESOURCE_COUNTS:-500 1000 2000 3000 4000 5000}}"
SAMPLE_DURATION="${SAMPLE_DURATION:-60}"
KWOK_VERSION="${KWOK_VERSION:-v0.7.0}"
CLEANUP_KWOK="${CLEANUP_KWOK:-true}"
SETUP_KWOK="${SETUP_KWOK:-true}"
DELETE_TIMEOUT="${DELETE_TIMEOUT:-300}"
REPORT_PATH="${REPORT_PATH:-}"

TEST_NS_LABEL="${TEST_NS_LABEL:-}"

KWOK_NAMESPACE="kwok-system"
TEST_NAMESPACE_PREFIX="load-test"
export KWOK_NODE_NAME="kwok-node-0"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_TEMPLATE_PATH="${RESOURCE_TEMPLATE_PATH:-${SCRIPT_DIR}/resources/pod-template.yaml}"

# Results accumulator: parallel arrays
RESULT_COUNTS=()
RESULT_FAILURES=()
# Per-step per-pod peaks: RESULT_STEP_PEAKS[step_index] = "ns/name:Mi ns/name:Mi ..."
RESULT_STEP_PEAKS=()

# ---------------------------------------------------------------------------
# Teardown
# ---------------------------------------------------------------------------
cleanup_kwok() {
  if [[ "${CLEANUP_KWOK}" == "true" ]]; then
    log "Deleting fake node '${KWOK_NODE_NAME}'"
    kubectl delete node "${KWOK_NODE_NAME}" --wait=false > /dev/null 2>&1 || true
    log "Deleting KWOK namespace ${KWOK_NAMESPACE}"
    kubectl delete namespace "${KWOK_NAMESPACE}" --wait=false > /dev/null 2>&1 || true
  else
    log "Skipping KWOK cleanup (CLEANUP_KWOK=${CLEANUP_KWOK})"
  fi
}

cleanup() {
  stop_sampler
  cleanup_kwok
  if [[ -n "${REPORT_PATH}" ]]; then
    print_report | tee "${REPORT_PATH}"
    log "Report saved to ${REPORT_PATH}"
  else
    print_report
  fi
}
trap cleanup EXIT

check_prerequisites() {
  for cmd in kubectl envsubst; do
    command -v "${cmd}" >/dev/null 2>&1 || die "'${cmd}' not found on PATH"
  done
  [[ -f "${RESOURCE_TEMPLATE_PATH}" ]] || die "RESOURCE_TEMPLATE_PATH not found: ${RESOURCE_TEMPLATE_PATH}"
}

# ---------------------------------------------------------------------------
# KWOK setup
# ---------------------------------------------------------------------------
setup_kwok() {
  log "Creating namespace ${KWOK_NAMESPACE}"
  kubectl create namespace "${KWOK_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

  log "Applying KWOK CRDs and controller manifests (namespace: ${KWOK_NAMESPACE})"
  curl -sSfL "https://github.com/kubernetes-sigs/kwok/releases/download/${KWOK_VERSION}/kwok.yaml" \
    | sed "s/namespace: kube-system/namespace: ${KWOK_NAMESPACE}/g" \
    | kubectl apply -f -

  log "Waiting for KWOK controller to be ready"
  kubectl wait -n "${KWOK_NAMESPACE}" \
    --for=condition=Available \
    --timeout=120s \
    deployment/kwok-controller

  log "Applying KWOK lifecycle stages (stage-fast.yaml)"
  kubectl apply -f "https://github.com/kubernetes-sigs/kwok/releases/download/${KWOK_VERSION}/stage-fast.yaml"

  log "Creating fake node '${KWOK_NODE_NAME}'"
  kubectl apply -f "${SCRIPT_DIR}/resources/fake-node.yaml"
  log "Fake node created"
}

setup_test_namespace() {
  local ns=$1
  log "Creating test namespace '${ns}'"
  kubectl create namespace "${ns}" --dry-run=client -o yaml \
    | kubectl apply -f - > /dev/null
  if [[ -n "${TEST_NS_LABEL}" ]]; then
    local key="${TEST_NS_LABEL%%=*}"
    local val="${TEST_NS_LABEL##*=}"
    kubectl label namespace "${ns}" "${key}=${val}" --overwrite > /dev/null
  fi
}

# ---------------------------------------------------------------------------
# Resource management
# ---------------------------------------------------------------------------
create_resources() {
  local count=$1
  local ns=$2
  log "Creating ${count} resources in ${ns} namespace"

  local manifest_file
  manifest_file="$(mktemp).yaml"

  for (( i=1; i<=count; i++ )); do
    RESOURCE_INDEX="${i}" TEST_NAMESPACE="${ns}" \
      envsubst '${RESOURCE_INDEX} ${TEST_NAMESPACE} ${KWOK_NODE_NAME}' \
      < "${RESOURCE_TEMPLATE_PATH}" >> "${manifest_file}"
    echo "---" >> "${manifest_file}"
  done

  log "Manifest generated (${count} resources), applying..."
  kubectl apply --server-side -f "${manifest_file}" > /dev/null
  rm -f "${manifest_file}"
  log "Applied ${count} resources"
}

delete_step_namespace() {
  local ns=$1
  log "Deleting namespace ${ns} (timeout: ${DELETE_TIMEOUT}s)"
  kubectl delete namespace "${ns}" --wait=true --timeout="${DELETE_TIMEOUT}s" > /dev/null 2>&1 || \
    log "WARNING: namespace ${ns} not fully deleted within ${DELETE_TIMEOUT}s"
  log "Namespace ${ns} deleted"
}

# ---------------------------------------------------------------------------
# Memory sampling — collects kubectl top pod -A every second.
# Per-pod peaks stored in SAMPLER_DIR/<namespace>_<pod-name> files.
# ---------------------------------------------------------------------------
SAMPLER_PID=""
SAMPLER_DIR=""

start_sampler() {
  SAMPLER_DIR="$(mktemp -d)"

  (
    while true; do
      while IFS= read -r line; do
        # line format: NAMESPACE   NAME   CPU   MEMORY
        local ns name mem_raw mem_mi
        ns=$(echo "${line}"   | awk '{print $1}')
        name=$(echo "${line}" | awk '{print $2}')
        mem_raw=$(echo "${line}" | awk '{print $4}')
        mem_mi="${mem_raw//Mi/}"
        [[ "${mem_mi}" =~ ^[0-9]+$ ]] || continue
        local key="${ns}_${name}"
        local peak_file="${SAMPLER_DIR}/${key}"
        local current=0
        [[ -f "${peak_file}" ]] && current=$(cat "${peak_file}")
        if (( mem_mi > current )); then
          echo "${mem_mi}" > "${peak_file}"
        fi
      done < <(kubectl top pod -A --no-headers 2>/dev/null || true)
      sleep 1
    done
  ) &
  SAMPLER_PID=$!
}

stop_sampler() {
  if [[ -n "${SAMPLER_PID}" ]]; then
    kill "${SAMPLER_PID}" 2>/dev/null || true
    wait "${SAMPLER_PID}" 2>/dev/null || true
    SAMPLER_PID=""
  fi
}

# Snapshot current per-pod peaks into RESULT_STEP_PEAKS for the current step,
# then clear the sampler dir for the next step.
snapshot_and_reset_peaks() {
  if [[ -z "${SAMPLER_DIR}" || ! -d "${SAMPLER_DIR}" ]]; then
    RESULT_STEP_PEAKS+=("")
    return
  fi
  local entries=""
  for peak_file in "${SAMPLER_DIR}"/*; do
    [[ -f "${peak_file}" ]] || continue
    local key val
    key="$(basename "${peak_file}")"
    val="$(cat "${peak_file}")"
    # Convert key ns_podname back to ns/podname (first underscore is separator).
    local ns="${key%%_*}"
    local name="${key#*_}"
    entries+="${ns}/${name}:${val} "
    rm -f "${peak_file}"
  done
  RESULT_STEP_PEAKS+=("${entries% }")
}

# ---------------------------------------------------------------------------
# Load loop
# ---------------------------------------------------------------------------
run_load_loop() {
  log "Starting load loop with resource counts: ${RESOURCE_COUNTS}"
  log "Resource template: ${RESOURCE_TEMPLATE_PATH}"
  start_sampler

  # Baseline — sample before any resources are created.
  log "=== Baseline: sampling memory before any resources (${SAMPLE_DURATION}s) ==="
  sleep "${SAMPLE_DURATION}"
  snapshot_and_reset_peaks
  local baseline_failures
  baseline_failures=$(kubectl get pods -A --no-headers 2>/dev/null \
    | awk '($4 ~ /CrashLoopBackOff|Error|OOMKilled/) {print $1"/"$2"("$4")"}' \
    | tr '\n' ' ' | sed 's/ $//')
  RESULT_COUNTS+=("baseline")
  RESULT_FAILURES+=("${baseline_failures}")
  log "Baseline done"

  for count in ${RESOURCE_COUNTS}; do
    local step_ns="${TEST_NAMESPACE_PREFIX}-${count}"
    log "=== Step: ${count} resources (namespace: ${step_ns}) ==="

    setup_test_namespace "${step_ns}"
    create_resources "${count}" "${step_ns}"

    log "Sampling memory for ${SAMPLE_DURATION}s..."
    sleep "${SAMPLE_DURATION}"

    snapshot_and_reset_peaks

    # Check for failing pods cluster-wide (excluding test namespace).
    local failing_pods
    failing_pods=$(kubectl get pods -A --no-headers 2>/dev/null \
      | awk -v skip="${step_ns}" '$1 != skip && ($4 ~ /CrashLoopBackOff|Error|OOMKilled/) {print $1"/"$2"("$4")"}' \
      | tr '\n' ' ' | sed 's/ $//')
    local step_status
    if [[ -z "${failing_pods}" ]]; then
      step_status="ok"
    else
      step_status="FAIL: ${failing_pods}"
    fi

    log "Step ${count}: cluster status = ${step_status}"
    RESULT_COUNTS+=("${count}")
    RESULT_FAILURES+=("${failing_pods}")

    delete_step_namespace "${step_ns}"
  done

  stop_sampler
}

# ---------------------------------------------------------------------------
# Report — rows: pods, columns: steps
# ---------------------------------------------------------------------------
print_report() {
  if [[ ${#RESULT_COUNTS[@]} -eq 0 ]]; then return; fi

  # Collect all unique pod keys across all steps.
  local all_pods_str=""
  for step_data in "${RESULT_STEP_PEAKS[@]}"; do
    for entry in ${step_data}; do
      local pod="${entry%%:*}"
      # Append only if not already present.
      case " ${all_pods_str} " in
        *" ${pod} "*) ;;
        *) all_pods_str="${all_pods_str} ${pod}" ;;
      esac
    done
  done

  # Sort pod names.
  local -a sorted_pods=()
  IFS=$'\n' sorted_pods=($(tr ' ' '\n' <<<"${all_pods_str# }" | sort)); unset IFS

  # Print header.
  printf "| %-45s" "pod (namespace/name)"
  for count in "${RESULT_COUNTS[@]}"; do
    if [[ "${count}" == "baseline" ]]; then
      printf " | %10s" "baseline"
    else
      printf " | %6s res" "${count}"
    fi
  done
  printf " |\n"

  # Separator.
  printf "|%s" "$(printf '%0.s-' {1..47})"
  for _ in "${RESULT_COUNTS[@]}"; do
    printf "|%s" "$(printf '%0.s-' {1..12})"
  done
  printf "|\n"

  # One row per pod.
  for pod in "${sorted_pods[@]}"; do
    printf "| %-45s" "${pod}"
    local step_i=0
    for step_data in "${RESULT_STEP_PEAKS[@]}"; do
      local val="-"
      # Count how many times this pod appears in the failures for this step.
      local fail_count=0
      for token in ${RESULT_FAILURES[$step_i]}; do
        if [[ "${token%%(*}" == "${pod}" ]]; then
          fail_count=$(( fail_count + 1 ))
        fi
      done
      if (( fail_count > 0 )); then
        val="FAILING"
      else
        for entry in ${step_data}; do
          if [[ "${entry%%:*}" == "${pod}" ]]; then
            val="${entry##*:} Mi"
            break
          fi
        done
      fi
      printf " | %10s" "${val}"
      step_i=$(( step_i + 1 ))
    done
    printf " |\n"
  done
  echo ""
}

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------
main() {
  check_prerequisites
  echo ""
  echo "## KWOK Load Test — Peak Memory per Pod (Mi)"
  echo "## Resource template: ${RESOURCE_TEMPLATE_PATH}"
  echo ""
  if [[ "${SETUP_KWOK}" == "true" ]]; then
    setup_kwok
  else
    log "Skipping KWOK setup (SETUP_KWOK=${SETUP_KWOK})"
  fi
  run_load_loop
}

main "$@"
