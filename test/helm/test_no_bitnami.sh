#!/usr/bin/env bash
set -euo pipefail

# Path setup
INFRA_CHART_DIR="tools/helm/tacito-square-infra"
CHART_YAML="${INFRA_CHART_DIR}/Chart.yaml"
VALUES_YAML="${INFRA_CHART_DIR}/values.yaml"

echo "=== Running Bitnami Avoidance Validation ==="

FAILED=0

# 1. Check Chart.yaml for Bitnami repository references
echo "Checking ${CHART_YAML}..."
if grep -qi "bitnami" "${CHART_YAML}"; then
  echo "❌ FAIL: Bitnami reference found in ${CHART_YAML}!"
  grep -ni "bitnami" "${CHART_YAML}"
  FAILED=1
else
  echo "✓ OK: No Bitnami references found in ${CHART_YAML}"
fi

# 2. Check values.yaml for Bitnami repository/image references
echo "Checking ${VALUES_YAML}..."
if grep -qi -E "bitnami|bitnamilegacy" "${VALUES_YAML}"; then
  echo "❌ FAIL: Bitnami or bitnamilegacy reference found in ${VALUES_YAML}!"
  grep -n -i -E "bitnami|bitnamilegacy" "${VALUES_YAML}"
  FAILED=1
else
  echo "✓ OK: No Bitnami or bitnamilegacy references found in ${VALUES_YAML}"
fi

# 3. Check helm dependencies for Bitnami references
echo "Checking Helm dependency list..."
if [ -f "${CHART_YAML}" ]; then
  if grep -q "repository:.*bitnami" "${CHART_YAML}"; then
    echo "❌ FAIL: Bitnami chart dependencies are defined in ${CHART_YAML}!"
    FAILED=1
  fi
fi

if [ $FAILED -ne 0 ]; then
  echo "=== Validation FAILED ==="
  exit 1
else
  echo "=== Validation PASSED successfully ==="
  exit 0
fi
