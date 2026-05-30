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
