#!/usr/bin/env bash

# test_documentation.sh - Validates that Tacito Square documentation files exist and conform to SPEC-FR-M2.9.

set -euo pipefail

FAILED=0

echo "=== Starting Documentation Validation ==="

# 1. Validate Root README.md
echo "Checking root README.md..."
if [ ! -f "README.md" ]; then
    echo "  [FAIL] Root README.md does not exist"
    FAILED=1
else
    # Check for architecture summary
    if ! grep -q -i "architecture" README.md; then
        echo "  [FAIL] Root README.md is missing an architecture summary or diagram reference"
        FAILED=1
    fi

    # Check for prerequisites
    if ! grep -q -i "prerequisites" README.md; then
        echo "  [FAIL] Root README.md is missing prerequisites"
        FAILED=1
    fi

    # Check for build instructions
    if ! grep -q "make build" README.md || ! grep -q "make test" README.md; then
        echo "  [FAIL] Root README.md is missing build/test instructions (make build/test)"
        FAILED=1
    fi

    # Check for decoupled local dev workflow reference
    if ! grep -q "tacito-square-infra" README.md && ! grep -q "tacito-infra" README.md; then
        echo "  [FAIL] Root README.md is missing references to the new infrastructure chart local dev workflow"
        FAILED=1
    fi
    echo "  [PASS] Root README.md structural check passed"
fi

# 2. Validate Infrastructure Helm Chart README.md
echo "Checking tools/helm/tacito-square-infra/README.md..."
INFRA_README="tools/helm/tacito-square-infra/README.md"
if [ ! -f "${INFRA_README}" ]; then
    echo "  [FAIL] Infrastructure Helm chart README.md does not exist"
    FAILED=1
else
    # Check for services included
    if ! grep -q -i "services included" "${INFRA_README}"; then
        echo "  [FAIL] Infrastructure Helm chart README.md is missing services summary"
        FAILED=1
    fi

    # Check for installation
    if ! grep -q -i "install" "${INFRA_README}"; then
        echo "  [FAIL] Infrastructure Helm chart README.md is missing installation instructions"
        FAILED=1
    fi

    # Check for configuration reference
    if ! grep -q -i "configuration" "${INFRA_README}"; then
        echo "  [FAIL] Infrastructure Helm chart README.md is missing configuration reference"
        FAILED=1
    fi
    echo "  [PASS] Infrastructure Helm chart README.md structural check passed"
fi

# 3. Validate Application Helm Chart README.md
echo "Checking tools/helm/tacito-square/README.md..."
APP_README="tools/helm/tacito-square/README.md"
if [ ! -f "${APP_README}" ]; then
    echo "  [FAIL] Application Helm chart README.md does not exist"
    FAILED=1
else
    # Check for components included (keeper, agent, etc.)
    if ! grep -q -i "components" "${APP_README}"; then
        echo "  [FAIL] Application Helm chart README.md is missing components summary"
        FAILED=1
    fi

    # Check for binding interfaces (TS_* environment variables)
    if ! grep -q "TS_KEEPER_" "${APP_README}" && ! grep -q "TS_AGENT_" "${APP_README}"; then
        echo "  [FAIL] Application Helm chart README.md is missing binding interface documentation (TS_*) environment variables"
        FAILED=1
    fi

    # Check for installation instructions (stating prerequisite: infra chart)
    if ! grep -q -i "prerequisite" "${APP_README}" && ! grep -q "tacito-square-infra" "${APP_README}" && ! grep -q "tacito-infra" "${APP_README}"; then
        echo "  [FAIL] Application Helm chart README.md is missing prerequisite infrastructure chart references"
        FAILED=1
    fi

    # Ensure no claims that NATS, Redis, PostgreSQL, Qdrant, Keycloak are BUNDLED directly in this chart (should only refer to them as external dependencies)
    # Check that under "Components Deployed" there are no lines showing nats/redis/etc as "Enabled" by default in this chart.
    if grep -A 10 "## Components Deployed" "${APP_README}" 2>/dev/null | grep -E -q -i "(nats|redis|postgresql|qdrant|keycloak).*enabled"; then
        echo "  [FAIL] Application Helm chart README.md mistakenly references infrastructure components as BUNDLED/ENABLED by default"
        FAILED=1
    fi
    echo "  [PASS] Application Helm chart README.md structural check passed"
fi

echo ""
if [ "${FAILED}" -eq 0 ]; then
    echo "=== [SUCCESS] All documentation conforms to SPEC-FR-M2.9! ==="
    exit 0
else
    echo "=== [FAILURE] One or more documentation checks failed! ==="
    exit 1
fi
