#!/usr/bin/env bash
# test/helm/test_infra_chart.sh — Validation tests for the infrastructure Helm chart
# RED phase: these tests define the expected behavior of the chart.
# Run: bash test/helm/test_infra_chart.sh

set -euo pipefail

INFRA_CHART="tools/helm/tacito-square-infra"
APP_CHART="tools/helm/tacito-square"
RELEASE="tacito-infra-test"

PASS=0
FAIL=0
ERRORS=()

pass() { ((PASS++)); printf "  \033[32m✓ PASS\033[0m %s\n" "$1"; }
fail() { ((FAIL++)); ERRORS+=("$1"); printf "  \033[31m✗ FAIL\033[0m %s\n" "$1"; }

section() { printf "\n\033[1m── %s ──\033[0m\n" "$1"; }

# ── 1. Chart structure ──────────────────────────────────
section "Chart Structure"

if [ -f "${INFRA_CHART}/Chart.yaml" ]; then
  pass "Chart.yaml exists"
else
  fail "Chart.yaml exists"
fi

if [ -f "${INFRA_CHART}/values.yaml" ]; then
  pass "values.yaml exists"
else
  fail "values.yaml exists"
fi

if [ -d "${INFRA_CHART}/templates" ]; then
  pass "templates/ directory exists"
else
  fail "templates/ directory exists"
fi

# ── 2. Helm lint ────────────────────────────────────────
section "Helm Lint"

if helm lint "${INFRA_CHART}" > /dev/null 2>&1; then
  pass "helm lint passes"
else
  fail "helm lint passes"
fi

# ── 3. Helm template renders valid YAML ─────────────────
section "Helm Template"

TEMPLATE_FILE=$(mktemp)
trap "rm -f ${TEMPLATE_FILE}" EXIT

if helm template "${RELEASE}" "${INFRA_CHART}" > "${TEMPLATE_FILE}" 2>/dev/null; then
  pass "helm template succeeds"
else
  fail "helm template succeeds"
fi

# ── 4. Sub-chart presence in template output ────────────
section "Sub-chart Presence"

SUBCHART_CHECKS=(
  "nats"
  "redis"
  "postgresql"
  "qdrant"
  "otel-collector:opentelemetry-collector"
  "keycloak"
)

for entry in "${SUBCHART_CHECKS[@]}"; do
  name="${entry%%:*}"
  # Search for the subchart name in the template output (case-insensitive)
  if grep -qi "${name}" "${TEMPLATE_FILE}"; then
    pass "sub-chart '${name}' present in template output"
  else
    fail "sub-chart '${name}' present in template output"
  fi
done

# ── 4b. New Service presence in template output ───────────
section "New Custom Telemetry Services Presence"

TELEMETRY_SERVICES=(
  "tempo"
  "mimir"
  "grafana"
)

for service in "${TELEMETRY_SERVICES[@]}"; do
  if grep -qi "tacito-square-infra/templates/${service}.yaml" "${TEMPLATE_FILE}"; then
    pass "telemetry service '${service}' present in template output"
  else
    fail "telemetry service '${service}' present in template output"
  fi
done

# ── 5. Conditional toggling ─────────────────────────────
section "Conditional Toggling"

TOGGLEABLE=(
  "nats:nats"
  "redis:redis"
  "postgresql:postgresql"
  "qdrant:qdrant"
  "otel-collector:otel-collector"
  "keycloak:keycloak"
)

for entry in "${TOGGLEABLE[@]}"; do
  name="${entry%%:*}"
  condition="${entry##*:}"
  toggle_file=$(mktemp)
  helm template "${RELEASE}" "${INFRA_CHART}" --set "${condition}.enabled=false" > "${toggle_file}" 2>/dev/null || true
  # Check for the chart-specific source comment (e.g. "tacito-square-infra/charts/nats/")
  # For postgresql, Keycloak's own sub-chart also renders postgresql resources, so we
  # must check specifically for the top-level chart path, not just the name.
  if grep -q "tacito-square-infra/charts/${name}/" "${toggle_file}"; then
    fail "disabling '${condition}' removes '${name}' from output"
  else
    pass "disabling '${condition}' removes '${name}' from output"
  fi
  rm -f "${toggle_file}"
done

# ── 5b. New Service Conditional Toggling ──────────────────
section "New Service Conditional Toggling"

for service in "${TELEMETRY_SERVICES[@]}"; do
  toggle_file=$(mktemp)
  helm template "${RELEASE}" "${INFRA_CHART}" --set "${service}.enabled=false" > "${toggle_file}" 2>/dev/null || true
  if grep -qi "tacito-square-infra/templates/${service}.yaml" "${toggle_file}"; then
    fail "disabling '${service}' removes it from output"
  else
    pass "disabling '${service}' removes it from output"
  fi
  rm -f "${toggle_file}"
done


# ── 6. Application chart has no infrastructure deps ─────
section "Application Chart Isolation"

if [ -f "${APP_CHART}/Chart.yaml" ]; then
  dep_count=$(grep -cE '^\s+- name:' "${APP_CHART}/Chart.yaml" || true)
  if [ "${dep_count}" -eq 0 ] 2>/dev/null; then
    pass "application chart has no sub-chart dependencies"
  else
    fail "application chart has no sub-chart dependencies (found ${dep_count})"
  fi
else
  fail "application Chart.yaml exists"
fi

# ── 7. Keycloak realm configuration ─────────────────────
section "Keycloak Realm"

if grep -q "tacito-keeper" "${TEMPLATE_FILE}"; then
  pass "Keycloak realm contains 'tacito-keeper' client"
else
  fail "Keycloak realm contains 'tacito-keeper' client"
fi

if grep -q "keeper-admin" "${TEMPLATE_FILE}"; then
  pass "Keycloak realm contains 'keeper-admin' role"
else
  fail "Keycloak realm contains 'keeper-admin' role"
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
