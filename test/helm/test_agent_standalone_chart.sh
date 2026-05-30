#!/usr/bin/env bash
# test/helm/test_agent_standalone_chart.sh — Dry-run template validation tests for standalone agent Helm chart
# Part of TASK-M5.7.1 TDD cycle.
# Run: bash test/helm/test_agent_standalone_chart.sh

set -euo pipefail

AGENT_CHART="tools/helm/tacito-agent"
RELEASE="my-agent"

PASS=0
FAIL=0
ERRORS=()

pass() { ((PASS++)); printf "  \033[32m✓ PASS\033[0m %s\n" "$1"; }
fail() { ((FAIL++)); ERRORS+=("$1"); printf "  \033[31m✗ FAIL\033[0m %s\n" "$1"; }

section() { printf "\n\033[1m── %s ──\033[0m\n" "$1"; }

# ── 1. Chart existence check ──────────────────────────────
section "Chart Directory Check"

if [ -d "${AGENT_CHART}" ]; then
  pass "Standalone chart directory exists"
else
  fail "Standalone chart directory exists"
fi

# ── 2. Chart.yaml validation ──────────────────────────────
section "Chart Metadata Check"

if [ -f "${AGENT_CHART}/Chart.yaml" ]; then
  pass "Chart.yaml exists"
  if grep -q "name: tacito-agent" "${AGENT_CHART}/Chart.yaml"; then
    pass "Chart name is correct (tacito-agent)"
  else
    fail "Chart name is correct (tacito-agent)"
  fi
else
  fail "Chart.yaml exists"
fi

# ── 3. Template Rendering & ConfigMap Verification ────────
section "ConfigMap Rendering and Keys Check"

TEMPLATE_FILE=$(mktemp)
trap "rm -f ${TEMPLATE_FILE}" EXIT

if helm template "${RELEASE}" "${AGENT_CHART}" > "${TEMPLATE_FILE}" 2>/dev/null; then
  pass "helm template succeeds"
else
  fail "helm template succeeds"
fi

# ConfigMap metadata checks
if grep -q "name: ${RELEASE}-tacito-agent-config" "${TEMPLATE_FILE}" 2>/dev/null; then
  pass "ConfigMap named '${RELEASE}-tacito-agent-config' rendered"
else
  fail "ConfigMap named '${RELEASE}-tacito-agent-config' rendered"
fi

# ConfigMap standard environment variable keys assertions
REQUIRED_KEYS=(
  "TS_AGENT_ID"
  "TS_AGENT_NAME"
  "TS_AGENT_TENANT_ID"
  "TS_AGENT_COMMUNITY_REF"
  "TS_AGENT_LLM_MODEL"
  "TS_AGENT_LLM_ENDPOINT"
  "TS_AGENT_LLM_TEMPERATURE"
  "TS_AGENT_LLM_MAX_TOKENS"
  "TS_AGENT_SYSTEM_PROMPT"
  "TS_AGENT_REDIS_URL"
  "TS_AGENT_REDIS_KEY_NAMESPACE"
  "TS_AGENT_REDIS_TTL_SECONDS"
  "TS_AGENT_QDRANT_URL"
  "TS_AGENT_QDRANT_COLLECTION_NAME"
  "TS_AGENT_QDRANT_VECTOR_DIMENSION"
  "TS_AGENT_NATS_URL"
  "TS_AGENT_S3_URL"
  "TS_AGENT_S3_BUCKET"
)

for key in "${REQUIRED_KEYS[@]}"; do
  if grep -q "${key}:" "${TEMPLATE_FILE}" 2>/dev/null; then
    pass "ConfigMap contains environment variable: ${key}"
  else
    fail "ConfigMap contains environment variable: ${key}"
  fi
done

# ── 4. Deployment & Environment Injection Verification ──
section "Deployment and Environment Injection Check"

if grep -q "kind: Deployment" "${TEMPLATE_FILE}" 2>/dev/null; then
  pass "Deployment resource rendered"
else
  fail "Deployment resource rendered"
fi

if grep -q "name: ${RELEASE}-tacito-agent" "${TEMPLATE_FILE}" 2>/dev/null; then
  pass "Deployment is named '${RELEASE}-tacito-agent'"
else
  fail "Deployment is named '${RELEASE}-tacito-agent'"
fi

# ConfigMap binding via envFrom check
if grep -A 10 "containers:" "${TEMPLATE_FILE}" | grep -q "envFrom:" 2>/dev/null; then
  pass "Deployment uses envFrom for ConfigMap reference"
else
  fail "Deployment uses envFrom for ConfigMap reference"
fi

if grep -A 15 "envFrom:" "${TEMPLATE_FILE}" | grep -q "name: ${RELEASE}-tacito-agent-config" 2>/dev/null; then
  pass "envFrom points to config map '${RELEASE}-tacito-agent-config'"
else
  fail "envFrom points to config map '${RELEASE}-tacito-agent-config'"
fi

# Secure credentials secret check (TS_AGENT_LLM_API_KEY)
if grep -q "name: TS_AGENT_LLM_API_KEY" "${TEMPLATE_FILE}" 2>/dev/null; then
  pass "TS_AGENT_LLM_API_KEY env var is defined"
else
  fail "TS_AGENT_LLM_API_KEY env var is defined"
fi

if grep -A 5 "name: TS_AGENT_LLM_API_KEY" "${TEMPLATE_FILE}" | grep -q "secretKeyRef:" 2>/dev/null; then
  pass "TS_AGENT_LLM_API_KEY uses secretKeyRef"
else
  fail "TS_AGENT_LLM_API_KEY uses secretKeyRef"
fi

if grep -A 5 "name: TS_AGENT_LLM_API_KEY" "${TEMPLATE_FILE}" | grep -qE "name: \"?tacito-agent-llm-secret\"?" 2>/dev/null; then
  pass "TS_AGENT_LLM_API_KEY references correct secret name"
else
  fail "TS_AGENT_LLM_API_KEY references correct secret name"
fi

# Workload probes check
if grep -A 5 "livenessProbe:" "${TEMPLATE_FILE}" | grep -q "path: /healthz" 2>/dev/null; then
  pass "Liveness probe is configured for /healthz"
else
  fail "Liveness probe is configured for /healthz"
fi

if grep -A 5 "readinessProbe:" "${TEMPLATE_FILE}" | grep -q "path: /readyz" 2>/dev/null; then
  pass "Readiness probe is configured for /readyz"
else
  fail "Readiness probe is configured for /readyz"
fi

# ── 5. Client Deployment Verification ──────────────────
section "Client Deployment Check"

if grep -q "name: ${RELEASE}-tacito-agent-client" "${TEMPLATE_FILE}" 2>/dev/null; then
  pass "Client Deployment named '${RELEASE}-tacito-agent-client' rendered"
else
  fail "Client Deployment named '${RELEASE}-tacito-agent-client' rendered"
fi

if grep -A 10 "name: ${RELEASE}-tacito-agent-client" "${TEMPLATE_FILE}" | grep -q "component: client" 2>/dev/null; then
  pass "Client Deployment has correct component label (client)"
else
  fail "Client Deployment has correct component label (client)"
fi

if grep -A 30 "name: ${RELEASE}-tacito-agent-client" "${TEMPLATE_FILE}" | grep -q "synadia/nats-box" 2>/dev/null; then
  pass "Client Deployment runs synadia/nats-box container"
else
  fail "Client Deployment runs synadia/nats-box container"
fi

if grep -A 35 "name: ${RELEASE}-tacito-agent-client" "${TEMPLATE_FILE}" | grep -q "NATS_URL" 2>/dev/null; then
  pass "Client Deployment configures NATS_URL environment variable"
else
  fail "Client Deployment configures NATS_URL environment variable"
fi




# ── Summary ─────────────────────────────────────────────
section "Summary"
TOTAL=$((PASS + FAIL))
printf "\n  %d/%d checks passed\n" "${PASS}" "${TOTAL}"

if [ "${FAIL}" -gt 0 ]; then
  printf "\n  \033[31mFailed checks:\033[0m\n"
  for err in "${ERRORS[@]}"; do
    printf "    - %s\n" "${err}"
  done
  printf "\n"
  exit 1
fi

printf "  \033[32mAll checks passed!\033[0m\n\n"
exit 0
