#!/usr/bin/env bash
# test/make/test_infra_targets.sh — Validation tests for Makefile infrastructure targets
# RED phase: these tests define the expected Makefile targets before implementation.
# Run: bash test/make/test_infra_targets.sh

set -euo pipefail

PASS=0
FAIL=0
ERRORS=()

pass() { ((PASS++)); printf "  \033[32m✓ PASS\033[0m %s\n" "$1"; }
fail() { ((FAIL++)); ERRORS+=("$1"); printf "  \033[31m✗ FAIL\033[0m %s\n" "$1"; }

section() { printf "\n\033[1m── %s ──\033[0m\n" "$1"; }

# ── 1. Target existence (dry-run) ───────────────────────
section "Target Existence (make -n)"

EXPECTED_TARGETS=(
  "helm-infra-deps"
  "helm-infra-lint"
  "helm-infra-template"
  "helm-infra-install"
  "helm-infra-uninstall"
)

for target in "${EXPECTED_TARGETS[@]}"; do
  if make -n "${target}" > /dev/null 2>&1; then
    pass "target '${target}' exists"
  else
    fail "target '${target}' exists"
  fi
done

# ── 2. Help output ──────────────────────────────────────
section "Help Output"

HELP_OUTPUT=$(make help 2>/dev/null || true)

for target in "${EXPECTED_TARGETS[@]}"; do
  if echo "${HELP_OUTPUT}" | grep -q "${target}"; then
    pass "'${target}' listed in 'make help'"
  else
    fail "'${target}' listed in 'make help'"
  fi
done

# ── 3. Variables ────────────────────────────────────────
section "Variables"

# Dry-run output should reference the infra chart path
DRYRUN_OUTPUT=$(make -n helm-infra-lint 2>/dev/null || true)

if echo "${DRYRUN_OUTPUT}" | grep -q "tacito-square-infra"; then
  pass "helm-infra-lint references infra chart path"
else
  fail "helm-infra-lint references infra chart path"
fi

# ── 4. Functional validation (requires chart to exist) ──
section "Functional Validation"

if make helm-infra-lint > /dev/null 2>&1; then
  pass "make helm-infra-lint succeeds"
else
  fail "make helm-infra-lint succeeds"
fi

if make helm-infra-template > /dev/null 2>&1; then
  pass "make helm-infra-template succeeds"
else
  fail "make helm-infra-template succeeds"
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
