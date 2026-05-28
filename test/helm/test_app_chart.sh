#!/usr/bin/env bash
# test/helm/test_app_chart.sh — Validation tests for the application Helm chart
# RED phase: defines expected behavior of component templates and CRDs.
# Run: bash test/helm/test_app_chart.sh

set -euo pipefail

APP_CHART="tools/helm/tacito-square"
RELEASE="tacito-test"

PASS=0
FAIL=0
ERRORS=()

pass() { ((PASS++)); printf "  \033[32m✓ PASS\033[0m %s\n" "$1"; }
fail() { ((FAIL++)); ERRORS+=("$1"); printf "  \033[31m✗ FAIL\033[0m %s\n" "$1"; }

section() { printf "\n\033[1m── %s ──\033[0m\n" "$1"; }

TEMPLATE_FILE=$(mktemp)
trap "rm -f ${TEMPLATE_FILE}" EXIT

# ── 1. Helm lint ────────────────────────────────────────
section "Helm Lint"

if helm lint "${APP_CHART}" > /dev/null 2>&1; then
  pass "helm lint passes"
else
  fail "helm lint passes"
fi

# ── 2. No infrastructure dependencies ──────────────────
section "No Infrastructure Dependencies"

dep_count=$(grep -cE '^\s+- name:' "${APP_CHART}/Chart.yaml" || true)
if [ "${dep_count}" -eq 0 ] 2>/dev/null; then
  pass "Chart.yaml has no sub-chart dependencies"
else
  fail "Chart.yaml has no sub-chart dependencies (found ${dep_count})"
fi

# ── 3. Template rendering ──────────────────────────────
section "Template Rendering"

if helm template "${RELEASE}" "${APP_CHART}" --set operator.enabled=true --set bff.enabled=true > "${TEMPLATE_FILE}" 2>/dev/null; then
  pass "helm template succeeds (all components enabled)"
else
  fail "helm template succeeds (all components enabled)"
fi

# ── 4. Keeper templates ────────────────────────────────
section "Keeper Templates"

if grep -q "kind: Deployment" "${TEMPLATE_FILE}" && grep -q "app.kubernetes.io/component: keeper" "${TEMPLATE_FILE}"; then
  pass "keeper Deployment rendered"
else
  fail "keeper Deployment rendered"
fi

if grep -q "kind: Service" "${TEMPLATE_FILE}" && grep -q "app.kubernetes.io/component: keeper" "${TEMPLATE_FILE}"; then
  pass "keeper Service rendered"
else
  fail "keeper Service rendered"
fi

if grep -q "kind: ServiceAccount" "${TEMPLATE_FILE}" && grep -q "keeper" "${TEMPLATE_FILE}"; then
  pass "keeper ServiceAccount rendered"
else
  fail "keeper ServiceAccount rendered"
fi

if grep -q "kind: Role" "${TEMPLATE_FILE}" && grep -q "keeper" "${TEMPLATE_FILE}"; then
  pass "keeper Role rendered"
else
  fail "keeper Role rendered"
fi

if grep -q "kind: RoleBinding" "${TEMPLATE_FILE}" && grep -q "keeper" "${TEMPLATE_FILE}"; then
  pass "keeper RoleBinding rendered"
else
  fail "keeper RoleBinding rendered"
fi

# ── 5. Agent templates ─────────────────────────────────
section "Agent Templates"

if grep -q "app.kubernetes.io/component: agent" "${TEMPLATE_FILE}"; then
  pass "agent Deployment rendered"
else
  fail "agent Deployment rendered"
fi

# ── 6. Operator templates (conditional) ────────────────
section "Operator Templates (conditional)"

if grep -q "app.kubernetes.io/component: operator" "${TEMPLATE_FILE}"; then
  pass "operator Deployment rendered (enabled=true)"
else
  fail "operator Deployment rendered (enabled=true)"
fi

if grep -q "kind: ServiceAccount" "${TEMPLATE_FILE}" && grep -q "operator" "${TEMPLATE_FILE}"; then
  pass "operator ServiceAccount rendered"
else
  fail "operator ServiceAccount rendered"
fi

if grep -q "kind: ClusterRole" "${TEMPLATE_FILE}" && grep -q "operator" "${TEMPLATE_FILE}"; then
  pass "operator ClusterRole rendered"
else
  fail "operator ClusterRole rendered"
fi

if grep -q "kind: ClusterRoleBinding" "${TEMPLATE_FILE}" && grep -q "operator" "${TEMPLATE_FILE}"; then
  pass "operator ClusterRoleBinding rendered"
else
  fail "operator ClusterRoleBinding rendered"
fi

# Verify operator is absent when disabled
DISABLED_FILE=$(mktemp)
helm template "${RELEASE}" "${APP_CHART}" --set operator.enabled=false > "${DISABLED_FILE}" 2>/dev/null || true
if grep -q "app.kubernetes.io/component: operator" "${DISABLED_FILE}"; then
  fail "operator absent when disabled"
else
  pass "operator absent when disabled"
fi
rm -f "${DISABLED_FILE}"

# ── 7. BFF templates (conditional) ─────────────────────
section "BFF Templates (conditional)"

if grep -q "app.kubernetes.io/component: bff" "${TEMPLATE_FILE}"; then
  pass "bff Deployment rendered (enabled=true)"
else
  fail "bff Deployment rendered (enabled=true)"
fi

# Verify BFF is absent when disabled
DISABLED_FILE=$(mktemp)
helm template "${RELEASE}" "${APP_CHART}" > "${DISABLED_FILE}" 2>/dev/null || true
if grep -q "app.kubernetes.io/component: bff" "${DISABLED_FILE}"; then
  fail "bff absent when disabled"
else
  pass "bff absent when disabled"
fi
rm -f "${DISABLED_FILE}"

# ── 8. Health probes ───────────────────────────────────
section "Health Probes"

if grep -q "/healthz" "${TEMPLATE_FILE}"; then
  pass "liveness probe path /healthz found"
else
  fail "liveness probe path /healthz found"
fi

if grep -q "/readyz" "${TEMPLATE_FILE}"; then
  pass "readiness probe path /readyz found"
else
  fail "readiness probe path /readyz found"
fi

# ── 9. CRDs ────────────────────────────────────────────
section "Custom Resource Definitions"

if [ -d "${APP_CHART}/crds" ]; then
  pass "crds/ directory exists"
else
  fail "crds/ directory exists"
fi

# CRDs are not rendered by helm template — check file content directly
CRD_CONTENT=$(cat "${APP_CHART}"/crds/*.yaml 2>/dev/null || true)

if echo "${CRD_CONTENT}" | grep -q "TacitoAgent"; then
  pass "TacitoAgent CRD defined"
else
  fail "TacitoAgent CRD defined"
fi

if echo "${CRD_CONTENT}" | grep -q "TacitoCommunity"; then
  pass "TacitoCommunity CRD defined"
else
  fail "TacitoCommunity CRD defined"
fi

if echo "${CRD_CONTENT}" | grep -q "tacito.square.io"; then
  pass "CRDs use API group tacito.square.io"
else
  fail "CRDs use API group tacito.square.io"
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
