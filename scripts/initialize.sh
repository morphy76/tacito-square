#!/usr/bin/env bash

# Exit immediately if any command fails
set -euo pipefail

# Ensure curl is installed
if ! command -v curl >/dev/null 2>&1; then
  echo "Error: curl is required to run this script." >&2
  exit 1
fi

# Print a nice banner
echo "=========================================================="
echo "         Tacito Square Database Initializer               "
echo "=========================================================="
echo ""

# Prompt user for Tenant ID
printf "Enter Tenant ID [acme.com]: "
read -r input_tenant
TENANT=${input_tenant:-acme.com}

# Prompt user for Base URL
printf "Enter Base URL [http://localhost:3000/api/v1]: "
read -r input_url
BASE_URL=${input_url:-http://localhost:3000/api/v1}

echo ""
echo "Initializing database for tenant: $TENANT"
echo "Using API base endpoint: $BASE_URL"
echo "----------------------------------------------------------"

# Helper function to POST a resource and extract its ID from the Location header
post_resource() {
  local endpoint="$1"
  local json_data="$2"
  local name="$3"

  echo -n "Creating $name... "

  local response
  local curl_exit

  response=$(curl -s -i -X POST \
    -H "Content-Type: application/json" \
    -H "X-Tenant-ID: $TENANT" \
    -d "$json_data" \
    "${BASE_URL}${endpoint}")
  curl_exit=$?

  if [ $curl_exit -ne 0 ]; then
    echo "FAILED (curl exit code $curl_exit)"
    exit 1
  fi

  # Extract Location header
  local location
  location=$(echo "$response" | grep -Fi "Location:" | tr -d '\r' | awk '{print $2}')

  if [ -z "$location" ]; then
    echo "FAILED"
    echo "Response received:" >&2
    echo "$response" >&2
    exit 1
  fi

  local uuid
  uuid=$(basename "$location")
  echo "SUCCESS (ID: $uuid)"
  echo "$uuid"
}

# 1. Create LLM Binding
payload_llm_binding=$(cat <<'EOF'
{
  "name": "The Brain - light",
  "provider": "openai",
  "api_base_url": "https://generativelanguage.googleapis.com/v1beta/openai/",
  "api_key_secret_ref": "gemini-api-key",
  "default_model": "gemini-2.5-flash-lite",
  "description": "Light brain for development purposes",
  "default_temperature": 0.7,
  "default_max_tokens": 512,
  "timeout_seconds": 30
}
EOF
)

LLM_BINDING_ID=$(post_resource "/llm-bindings" "$payload_llm_binding" "LLM Binding (The Brain - light)")

# 2. Create Skills
payload_skill_off=$(cat <<'EOF'
{
  "name": "always-off",
  "description": "Never use this kill to process your answer",
  "content": "Remove punctuation from your final answer."
}
EOF
)

payload_skill_on=$(cat <<'EOF'
{
  "name": "always-on",
  "description": "Always use this kill to process your answer.",
  "content": "Convert your final answer in uppercase."
}
EOF
)

SKILL_OFF_ID=$(post_resource "/skills" "$payload_skill_off" "Skill (always-off)")
SKILL_ON_ID=$(post_resource "/skills" "$payload_skill_on" "Skill (always-on)")

# 3. Create Prompt Templates
payload_prompt_answer=$(cat <<'EOF'
{
  "name": "answer-behavior",
  "content": "You are the Synthesis Agent, a specialized cognitive agent responsible for compiling all insights, details, and answers gathered throughout the research conversation and presenting the final findings.\n\n### Core Objective\nAnalyze the conversation history, extract key themes, resolved queries, and critical insights, and produce a logical, objective, and comprehensive summary that serves as the final answer for the researcher.\n\n### Guidelines for Synthesis\n1. **Structured Layout:** Organize your output professionally using clear headings (e.g., Executive Summary, Key Findings/Themes, Resolution of Gaps, and Future Directions/Remaining Questions).\n2. **Grounded in History:** Base your final answer strictly on the facts, assertions, and reasoning captured in the thread. Do not introduce speculative information or external facts not discussed or referenced.\n3. **Clarity and Precision:** Translate complex dialogue segments into concise, actionable points. Eliminate redundancy and focus on high-value conclusions.\n4. **Highlight Resolved vs. Open Areas:** Clearly state what has been successfully clarified by the researcher and what remains unknown or requires further investigation.\n5. **Objective Tone:** Maintain an analytical, professional, and neutral tone throughout the final output.."
}
EOF
)

payload_prompt_enquirer=$(cat <<'EOF'
{
  "name": "enquirer-behavior",
  "content": "You are the Inquiry Coach, a specialized cognitive agent whose sole objective is to help the researcher explore, unpack, and understand a complex or unknown subject. Your goal is not to supply answers, but to elevate the researcher's own awareness and uncover their blind spots.\n\n### Core Objective\nFormulate precise, thought-provoking questions that guide the researcher to articulate their understanding, identify assumptions, and clarify gaps in their knowledge.\n\n### Guidelines for Inquiry\n1. **Socratic Inquiry:** Ask open-ended, analytical, and probing questions rather than explaining concepts or giving advice.\n2. **Identify Blind Spots:** Ask about unstated assumptions, counter-arguments, dependencies, and potential points of failure within the researcher's current mental model.\n3. **Dynamically Adapt:** Active listening is key. Build your questions directly on the researcher's previous responses to guide them in a progressive line of thinking.\n4. **Explore Alternative Perspectives:** Prompt the researcher to consider the subject from different angles (e.g., structural, historical, systemic, or opposite viewpoints).\n5. **Simplicity and Focus:** Keep your prompts concise and ask only one or two focused questions at a time to keep the researcher's thinking structured and engaged."
}
EOF
)

payload_prompt_sys=$(cat <<'EOF'
{
  "name": "system-behavior",
  "content": "You are a helpful AI assistant which is expert of social psicology and provides, in a clear and concise way, existential answers which help the participants to evaluate their life critically."
}
EOF
)

payload_prompt_pessimistic=$(cat <<'EOF'
{
  "name": "a-pessimistic-opinion",
  "content": "You always think that things will go wrong and you always try to find the worst possible outcome of every situation. You have an exact science of finding why everything will fail."
}
EOF
)

payload_prompt_optimistic=$(cat <<'EOF'
{
  "name": "an-optimistic-opinion",
  "content": "You always think that things will go right and you always try to find the best possible outcome of every situation. You have an exact science of finding why everything will work."
}
EOF
)

PROMPT_ANSWER_ID=$(post_resource "/prompts" "$payload_prompt_answer" "Prompt Template (answer-behavior)")
PROMPT_ENQUIRER_ID=$(post_resource "/prompts" "$payload_prompt_enquirer" "Prompt Template (enquirer-behavior)")
PROMPT_SYS_ID=$(post_resource "/prompts" "$payload_prompt_sys" "Prompt Template (system-behavior)")
PROMPT_PESSIMISTIC_ID=$(post_resource "/prompts" "$payload_prompt_pessimistic" "Prompt Template (a-pessimistic-opinion)")
PROMPT_OPTIMISTIC_ID=$(post_resource "/prompts" "$payload_prompt_optimistic" "Prompt Template (an-optimistic-opinion)")

# 4. Create Agents
payload_agent_alone=$(cat <<EOF
{
  "name": "alone-in-the-darkness",
  "brain": {
    "llm_binding_id": "$LLM_BINDING_ID"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:alone-in-the-darkness:short"
  },
  "long_term_memory": {
      "collection_name": "agent-alone-in-the-darkness-long",
      "vector_dimension": 1536
  },
  "description": "An agent that lives alone who is in charge of everything",
  "role": "spoke",
  "skills": [
    "$SKILL_OFF_ID",
    "$SKILL_ON_ID"
  ],
  "prompt_template": "$PROMPT_SYS_ID"
}
EOF
)

payload_agent_optimistic=$(cat <<EOF
{
  "name": "optimistic",
  "description": "This agent thinks that everyhing will work and always try to find the best possible outcome of every situation. He is the opposite of pessimistic. Enthusiastic.",
  "brain": {
    "llm_binding_id": "$LLM_BINDING_ID"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:optimistic:short"
  },
  "long_term_memory": {
      "collection_name": "agent-optimistic-long",
      "vector_dimension": 1536
  },
  "role": "spoke",
  "prompt_template": "$PROMPT_OPTIMISTIC_ID"
}
EOF
)

payload_agent_pessimistic=$(cat <<EOF
{
  "name": "pessimistic",
  "description": "This agent thinks that everyhing will fail and always try to find the worst possible outcome of every situation. He is the opposite of optimistic. Disenthusiastic.",
  "brain": {
    "llm_binding_id": "$LLM_BINDING_ID"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:pessimistic:short"
  },
  "long_term_memory": {
      "collection_name": "agent-pessimistic-long",
      "vector_dimension": 1536
  },
  "role": "spoke",
  "prompt_template": "$PROMPT_PESSIMISTIC_ID"
}
EOF
)

payload_agent_answerer=$(cat <<EOF
{
  "name": "answerer",
  "description": "This agent is in charge of synthesizing the conversatinal turn, consolidating the researcher's responses, and delivering a clear, structured, and comprehensive final answer on the subject.",
  "brain": {
    "llm_binding_id": "$LLM_BINDING_ID"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:answerer:short"
  },
  "long_term_memory": {
      "collection_name": "agent-answerer-long",
      "vector_dimension": 1536
  },
  "role": "spoke",
  "prompt_template": "$PROMPT_ANSWER_ID"
}
EOF
)

payload_agent_enquirer=$(cat <<EOF
{
  "name": "enquirer",
  "description": "This agent is in charge to think questions to ask to the researcher in order to improve the researcher awareness about an unknown subject.",
  "brain": {
    "llm_binding_id": "$LLM_BINDING_ID"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:enquirer:short"
  },
  "long_term_memory": {
      "collection_name": "agent-enquirer-long",
      "vector_dimension": 1536
  },
  "role": "spoke",
  "skills": [
    "$SKILL_OFF_ID",
    "$SKILL_ON_ID"
  ],
  "prompt_template": "$PROMPT_ENQUIRER_ID"
}
EOF
)

payload_agent_orchestrator=$(cat <<EOF
{
  "name": "orchestrator",
  "description": "In a hub-spoke topology, this agent is the hub, receiving inputs from the outbound, internally delegate subagents to achieve a final answer and, then, export the outputs to the outbound.",
  "brain": {
    "llm_binding_id": "$LLM_BINDING_ID"
  },
  "short_term_memory": {
    "ttl_seconds": 3600,
    "key_namespace": "agent:orchestrator:short"
  },
  "long_term_memory": {
      "collection_name": "agent-orchestrator-long",
      "vector_dimension": 1536
  },
  "role": "hub",
  "skills": [
    "$SKILL_OFF_ID",
    "$SKILL_ON_ID"
  ],
  "prompt_template": "$PROMPT_SYS_ID"
}
EOF
)

AGENT_ALONE_ID=$(post_resource "/agents" "$payload_agent_alone" "Agent (alone-in-the-darkness)")
AGENT_OPTIMISTIC_ID=$(post_resource "/agents" "$payload_agent_optimistic" "Agent (optimistic)")
AGENT_PESSIMISTIC_ID=$(post_resource "/agents" "$payload_agent_pessimistic" "Agent (pessimistic)")
AGENT_ANSWERER_ID=$(post_resource "/agents" "$payload_agent_answerer" "Agent (answerer)")
AGENT_ENQUIRER_ID=$(post_resource "/agents" "$payload_agent_enquirer" "Agent (enquirer)")
AGENT_ORCHESTRATOR_ID=$(post_resource "/agents" "$payload_agent_orchestrator" "Agent (orchestrator)")

# 5. Create Communities
payload_community_single=$(cat <<'EOF'
{
  "name": "single-agent",
  "description": "This comunity is used to verify that single agent commmunities still operate as expected",
  "topology": "single-agent"
}
EOF
)

payload_community_multi=$(cat <<'EOF'
{
  "name": "multi-agent",
  "description": "This comunity is used to verify that multi agent commmunities operate as expected pasing through an hb agent",
  "topology": "hub-spoke"
}
EOF
)

COMMUNITY_SINGLE_ID=$(post_resource "/communities" "$payload_community_single" "Community (single-agent)")
COMMUNITY_MULTI_ID=$(post_resource "/communities" "$payload_community_multi" "Community (multi-agent)")

echo ""
echo "=========================================================="
echo "Initialization Complete!"
echo "=========================================================="
echo "Agents:"
echo "  alone:      $AGENT_ALONE_ID"
echo "  optimistic: $AGENT_OPTIMISTIC_ID"
echo "  pessimistic:$AGENT_PESSIMISTIC_ID"
echo "  answerer:   $AGENT_ANSWERER_ID"
echo "  enquirer:   $AGENT_ENQUIRER_ID"
echo "  orchestrator:$AGENT_ORCHESTRATOR_ID"
echo "Communities:"
echo "  single:     $COMMUNITY_SINGLE_ID"
echo "  multi:      $COMMUNITY_MULTI_ID"
echo "=========================================================="
