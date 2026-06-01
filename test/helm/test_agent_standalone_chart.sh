#!/bin/bash
set -eo pipefail

CHART_DIR="tools/helm/tacito-agent"

echo "=== Running Standalone Agent Chart Validation Tests ==="

# Render the Helm templates into a temporary file
RENDERED=$(helm template my-agent "$CHART_DIR")

# Assert MockServer resources exist
echo "Checking for MockServer deployment..."
if ! echo "$RENDERED" | grep -q "name: my-agent-tacito-agent-mocks"; then
    echo "FAIL: MockServer deployment not found in template rendering"
    exit 1
fi

echo "Checking for MockServer expectations configmap..."
if ! echo "$RENDERED" | grep -q "MOCKSERVER_INITIALIZATION_JSON_PATH"; then
    echo "FAIL: MockServer initialization path not found in template rendering"
    exit 1
fi

# Assert new brain environment keys are injected
echo "Checking for TS_AGENT_BRAIN_PROVIDER env key..."
if ! echo "$RENDERED" | grep -q "TS_AGENT_BRAIN_PROVIDER"; then
    echo "FAIL: TS_AGENT_BRAIN_PROVIDER env key not found in agent configmap"
    exit 1
fi

echo "Checking for TS_AGENT_OPENAI_ENDPOINT env key..."
if ! echo "$RENDERED" | grep -q "TS_AGENT_OPENAI_ENDPOINT"; then
    echo "FAIL: TS_AGENT_OPENAI_ENDPOINT env key not found in agent configmap"
    exit 1
fi

echo "Checking for TS_AGENT_OLLAMA_ENDPOINT env key..."
if ! echo "$RENDERED" | grep -q "TS_AGENT_OLLAMA_ENDPOINT"; then
    echo "FAIL: TS_AGENT_OLLAMA_ENDPOINT env key not found in agent configmap"
    exit 1
fi

echo "Checking for TS_AGENT_OTEL_ENDPOINT env key..."
if ! echo "$RENDERED" | grep -q "TS_AGENT_OTEL_ENDPOINT"; then
    echo "FAIL: TS_AGENT_OTEL_ENDPOINT env key not found in agent configmap"
    exit 1
fi

echo "Checking for TS_AGENT_MCP_CLIENTS rendering..."
RENDERED_WITH_MCP=$(helm template my-agent "$CHART_DIR" --set "mcpClients[0].name=sqlite-mcp,mcpClients[0].transport=stdio,mcpClients[0].command=npx,mcpClients[0].allowedTools[0]=query")

if ! echo "$RENDERED_WITH_MCP" | grep -q "TS_AGENT_MCP_CLIENTS"; then
    echo "FAIL: TS_AGENT_MCP_CLIENTS not found when mcpClients is supplied in values"
    exit 1
fi

if ! echo "$RENDERED_WITH_MCP" | grep -q "sqlite-mcp"; then
    echo "FAIL: sqlite-mcp config not serialized inside TS_AGENT_MCP_CLIENTS value"
    exit 1
fi

echo "PASS: Standalone Agent Chart template validation successful!"
exit 0
