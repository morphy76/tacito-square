#!/usr/bin/env bash
# test/helm/test_secure_provisioning.sh — Validation tests for SPEC-FR-M2.11
# Checks that the infrastructure chart enforces TLS, secure authentication,
# and auto-provisions tenant resources (database, schema, realm, buckets).
# Run: bash test/helm/test_secure_provisioning.sh

set -euo pipefail

INFRA_CHART="tools/helm/tacito-square-infra"
APP_CHART="tools/helm/tacito-square"
RELEASE="ts-infra"

PASS=0
FAIL=0
ERRORS=()

pass() { ((PASS++)); printf "  \033[32m✓ PASS\033[0m %s\n" "$1"; }
fail() { ((FAIL++)); ERRORS+=("$1"); printf "  \033[31m✗ FAIL\033[0m %s\n" "$1"; }

section() { printf "\n\033[1m── %s ──\033[0m\n" "$1"; }

# ── Render templates ────────────────────────────────────
TEMPLATE_FILE=$(mktemp)
trap "rm -f ${TEMPLATE_FILE}" EXIT

if ! helm template "${RELEASE}" "${INFRA_CHART}" > "${TEMPLATE_FILE}" 2>/dev/null; then
  echo "ERROR: helm template failed — cannot proceed with validation."
  exit 1
fi

# ══════════════════════════════════════════════════════════
# 1. TLS Certificate Infrastructure
# ══════════════════════════════════════════════════════════
section "TLS Certificate Infrastructure"

# 1a. TLS generator Job exists
if grep -q "kind: Job" "${TEMPLATE_FILE}" && grep -q "tls-generate" "${TEMPLATE_FILE}"; then
  pass "TLS certificate generator Job exists"
else
  fail "TLS certificate generator Job exists"
fi

# 1b. CA bundle ConfigMap for client trust (created dynamically by TLS Job)
if grep -q "ca-bundle" "${TEMPLATE_FILE}"; then
  pass "CA bundle ConfigMap exists for client trust export"
else
  fail "CA bundle ConfigMap exists for client trust export"
fi

# 1c. RBAC resources for TLS Job
if grep -q "ServiceAccount" "${TEMPLATE_FILE}" && grep -q "tls-generate" "${TEMPLATE_FILE}"; then
  pass "TLS Job RBAC (ServiceAccount) exists"
else
  fail "TLS Job RBAC (ServiceAccount) exists"
fi

# ══════════════════════════════════════════════════════════
# 2. PostgreSQL Security (TASK-M2.11.1)
# ══════════════════════════════════════════════════════════
section "PostgreSQL Security"

# 2a. SSL enabled in PostgreSQL config
if grep -q "ssl = on" "${TEMPLATE_FILE}"; then
  pass "PostgreSQL SSL is enabled (ssl = on)"
else
  fail "PostgreSQL SSL is enabled (ssl = on)"
fi

# 2b. TLS secret volume mounted for PostgreSQL
if grep -q "${RELEASE}-pg-tls" "${TEMPLATE_FILE}"; then
  pass "PostgreSQL TLS secret reference exists"
else
  fail "PostgreSQL TLS secret reference exists"
fi

# 2c. Database provisioning — tacito database and user
INFRA_VALUES="${INFRA_CHART}/values.yaml"
if grep -q "value: tacito$" "${INFRA_VALUES}" 2>/dev/null || grep -qP "value:\s+tacito\s*$" "${INFRA_VALUES}" 2>/dev/null; then
  pass "PostgreSQL provisions 'tacito' database/user"
else
  fail "PostgreSQL provisions 'tacito' database/user"
fi

# 2d. Init script for schema creation
if grep -q "CREATE SCHEMA" "${TEMPLATE_FILE}"; then
  pass "PostgreSQL init script creates schema"
else
  fail "PostgreSQL init script creates schema"
fi

# ══════════════════════════════════════════════════════════
# 3. Keycloak Security (TASK-M2.11.2)
# ══════════════════════════════════════════════════════════
section "Keycloak Security"

# 3a. Keycloak NOT in dev mode
if grep -q "start-dev" "${TEMPLATE_FILE}"; then
  fail "Keycloak must NOT use 'start-dev' (use 'start' for HTTPS)"
else
  pass "Keycloak does not use 'start-dev'"
fi

# 3b. Keycloak HTTPS certificate args
if grep -q "https-certificate-file" "${TEMPLATE_FILE}"; then
  pass "Keycloak uses HTTPS certificate file argument"
else
  fail "Keycloak uses HTTPS certificate file argument"
fi

# 3c. Keycloak TLS secret mounted
if grep -q "${RELEASE}-keycloak-tls" "${TEMPLATE_FILE}"; then
  pass "Keycloak TLS secret reference exists"
else
  fail "Keycloak TLS secret reference exists"
fi

# 3d. Realm sslRequired is NOT "none"
if grep -q '"sslRequired": "none"' "${TEMPLATE_FILE}"; then
  fail "Keycloak realm sslRequired must NOT be 'none'"
else
  pass "Keycloak realm sslRequired is not 'none'"
fi

# 3e. Realm sslRequired is "external"
if grep -q '"sslRequired": "external"' "${TEMPLATE_FILE}"; then
  pass "Keycloak realm sslRequired is 'external'"
else
  fail "Keycloak realm sslRequired is 'external'"
fi

# ══════════════════════════════════════════════════════════
# 4. Redis Security (TASK-M2.11.3)
# ══════════════════════════════════════════════════════════
section "Redis Security"

# 4a. Redis TLS enabled
if grep -q "tls-port" "${TEMPLATE_FILE}" || grep -q "tacito-infra-redis-tls" "${TEMPLATE_FILE}"; then
  pass "Redis TLS is configured"
else
  fail "Redis TLS is configured"
fi

# 4b. Redis password authentication
if grep -q "requirepass" "${TEMPLATE_FILE}" || grep -q "REDIS_PASSWORD" "${TEMPLATE_FILE}"; then
  pass "Redis password authentication is configured"
else
  fail "Redis password authentication is configured"
fi

# ══════════════════════════════════════════════════════════
# 5. MinIO Security (TASK-M2.11.3)
# ══════════════════════════════════════════════════════════
section "MinIO Security"

# 5a. MinIO TLS enabled (check values.yaml since MinIO is chart-level config)
if grep -qA5 "^minio:" "${INFRA_VALUES}" | grep -q "enabled: true"; then
  # Only check TLS if MinIO is enabled
  if grep -q "tacito-infra-minio-tls" "${INFRA_VALUES}" 2>/dev/null || grep -q "tls:" "${INFRA_VALUES}" 2>/dev/null; then
    pass "MinIO TLS is configured"
  else
    fail "MinIO TLS is configured"
  fi
else
  pass "MinIO disabled — TLS check skipped"
fi

# 5b. MinIO bucket named 'tacito' with private policy
if grep -q "name: tacito$" "${INFRA_VALUES}" 2>/dev/null && grep -q "policy: none" "${INFRA_VALUES}" 2>/dev/null; then
  pass "MinIO bucket 'tacito' with private policy"
else
  fail "MinIO bucket 'tacito' with private policy"
fi

# ══════════════════════════════════════════════════════════
# 6. NATS Security (TASK-M2.11.3)
# ══════════════════════════════════════════════════════════
section "NATS Security"

# 6a. NATS TLS configured
if grep -q "tacito-infra-nats-tls" "${TEMPLATE_FILE}" || grep -q "nats-tls" "${INFRA_VALUES}"; then
  pass "NATS TLS is configured"
else
  fail "NATS TLS is configured"
fi

# 6b. NATS authentication configured
if grep -q "authorization" "${INFRA_VALUES}" 2>/dev/null || grep -q "token" "${INFRA_VALUES}" 2>/dev/null; then
  pass "NATS authentication is configured"
else
  fail "NATS authentication is configured"
fi

# ══════════════════════════════════════════════════════════
# 7. Application Chart — Secure Endpoints Only
# ══════════════════════════════════════════════════════════
section "Application Chart Secure Endpoints"

APP_VALUES="${APP_CHART}/values.yaml"

# 7a. No plaintext HTTP endpoints to infrastructure services (excluding keeper URL which is app-internal)
INFRA_HTTP=$(grep -E "TS_.*_(OIDC_ISSUER|S3_ENDPOINT)" "${APP_VALUES}" | grep -c "http://" || true)
if [ "${INFRA_HTTP}" -eq 0 ]; then
  pass "No plaintext HTTP endpoints to infrastructure services"
else
  fail "No plaintext HTTP endpoints to infrastructure services (found ${INFRA_HTTP})"
fi

# 7b. Redis uses rediss:// (TLS)
REDIS_PLAIN=$(grep -c "redis://" "${APP_VALUES}" || true)
if [ "${REDIS_PLAIN}" -eq 0 ]; then
  pass "Redis connections use rediss:// (TLS)"
else
  fail "Redis connections use rediss:// (TLS) — found ${REDIS_PLAIN} plaintext redis:// references"
fi

# 7c. PostgreSQL sslmode
if grep -q "sslmode" "${APP_VALUES}" || grep -q "DB_SSLMODE" "${APP_VALUES}"; then
  pass "PostgreSQL sslmode is configured in application chart"
else
  fail "PostgreSQL sslmode is configured in application chart"
fi

# 7d. CA bundle mount reference in application chart or values
if grep -qi "ca-bundle\|ca\.crt\|CA_CERT\|truststore" "${APP_VALUES}"; then
  pass "Application chart references CA trust bundle"
else
  fail "Application chart references CA trust bundle"
fi

# ══════════════════════════════════════════════════════════
# Summary
# ══════════════════════════════════════════════════════════
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
