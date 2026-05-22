#!/usr/bin/env bash

# test_dockerfiles.sh - Validates that Tacito Square Dockerfiles conform to SPEC-FR-M2.7.
# Ensures multi-stage, CGO_ENABLED=0, minimal ldflags, distroless nonroot runtime.

set -euo pipefail

COMPONENTS=("keeper" "agent" "operator" "bff")
FAILED=0

echo "=== Starting Dockerfile Validation ==="

for COMP in "${COMPONENTS[@]}"; do
    FILE="tools/docker/Dockerfile.${COMP}"
    echo "Checking ${COMP} Dockerfile at ${FILE}..."

    # 1. Check existence
    if [ ! -f "${FILE}" ]; then
        echo "  [FAIL] Dockerfile does not exist"
        FAILED=1
        continue
    fi

    # 2. Check Build stage uses golang base
    if ! grep -q -E "^FROM golang:" "${FILE}"; then
        echo "  [FAIL] Build stage does not use a 'golang:' base image"
        FAILED=1
    fi

    # 3. Check CGO_ENABLED=0
    if ! grep -q "CGO_ENABLED=0" "${FILE}"; then
        echo "  [FAIL] Build stage does not set CGO_ENABLED=0"
        FAILED=1
    fi

    # 4. Check ldflags="-s -w"
    if ! grep -q -- "-ldflags=\"-s -w\"" "${FILE}" && ! grep -q -- "-ldflags='-s -w'" "${FILE}" && ! grep -q -- "-ldflags=\"-w -s\"" "${FILE}"; then
        echo "  [FAIL] Build stage does not specify ldflags for minimal size (-s -w)"
        FAILED=1
    fi

    # 5. Check Runtime stage uses distroless nonroot
    if ! grep -q "FROM gcr.io/distroless/base-nossl-debian13:nonroot" "${FILE}"; then
        echo "  [FAIL] Runtime stage does not use 'gcr.io/distroless/base-nossl-debian13:nonroot'"
        FAILED=1
    fi

    # 6. Check USER is nonroot
    if ! grep -q "USER nonroot" "${FILE}"; then
        echo "  [FAIL] Dockerfile does not specify 'USER nonroot' or similar"
        FAILED=1
    fi

    echo "  [PASS] All structural patterns conform to SPEC-FR-M2.7"
done

echo ""
if [ "${FAILED}" -eq 0 ]; then
    echo "=== [SUCCESS] All Dockerfiles are compliant! ==="
    exit 0
else
    echo "=== [FAILURE] One or more Dockerfiles failed validation! ==="
    exit 1
fi
